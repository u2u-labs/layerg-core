package crawler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	crawlerQuery "github.com/u2u-labs/layerg-core/server/crawler/crawler_query"
	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
	"go.uber.org/zap"
)

func CrawlSupportedChains(ctx context.Context, logger *zap.Logger, db *sql.DB, rdb *redis.Client) error {
	// Query all supported chains with all necessary fields
	rows, err := db.QueryContext(ctx, `
		SELECT id, chain, name, rpc_url, chain_id, explorer, latest_block, block_time 
		FROM chains
	`)
	if err != nil {
		return fmt.Errorf("error querying chains: %v", err)
	}
	defer rows.Close()

	var chains []models.Chain
	for rows.Next() {
		var chain models.Chain
		if err := rows.Scan(
			&chain.ID,
			&chain.Chain,
			&chain.Name,
			&chain.RpcUrl,
			&chain.ChainID,
			&chain.Explorer,
			&chain.LatestBlock,
			&chain.BlockTime,
		); err != nil {
			logger.Error("Error scanning chain row", zap.Error(err))
			continue
		}
		// Set default block time if not set
		if chain.BlockTime == 0 {
			chain.BlockTime = 2000 // 2 seconds default
		}
		chains = append(chains, chain)
	}

	for _, c := range chains {
		// Delete chain from cache
		models.ContractType[c.ID] = make(map[string]models.Asset)
		if err = deleteChainFromCache(ctx, rdb, c.ID); err != nil {
			return err
		}
		if err = deleteChainAssetsFromCache(ctx, rdb, c.ID); err != nil {
			return err
		}

		// Query assets with pagination
		offset := 0
		limit := 10
		var assets []models.Asset

		for {
			query := `
                SELECT id, chain_id, collection_address, type, created_at, updated_at, 
                       decimal_data, initial_block, last_updated
                FROM assets 
                WHERE chain_id = $1 
                LIMIT $2 OFFSET $3
            `
			rows, err := db.QueryContext(ctx, query, c.ID, limit, offset)
			if err != nil {
				return fmt.Errorf("error querying assets: %v", err)
			}

			var pageAssets []models.Asset
			for rows.Next() {
				var asset models.Asset
				if err := rows.Scan(
					&asset.ID,
					&asset.ChainID,
					&asset.CollectionAddress,
					&asset.Type,
					&asset.CreatedAt,
					&asset.UpdatedAt,
					&asset.DecimalData,
					&asset.InitialBlock,
					&asset.LastUpdated,
				); err != nil {
					logger.Error("Error scanning asset row", zap.Error(err))
					continue
				}
				pageAssets = append(pageAssets, asset)
			}
			rows.Close()

			assets = append(assets, pageAssets...)
			if len(pageAssets) < limit {
				break
			}
			offset += limit
		}

		// Cache chain and assets
		if err = setChainToCache(ctx, rdb, &c); err != nil {
			return err
		}
		if err = setAssetsToCache(ctx, rdb, assets); err != nil {
			return err
		}

		for _, a := range assets {
			models.ContractType[a.ChainID][a.CollectionAddress] = a
		}

		// Start chain crawler
		client, err := initChainClient(&c)
		if err != nil {
			return err
		}
		go StartChainCrawler(ctx, logger.Sugar(), client, db, &c, rdb)
	}
	return nil
}

func ProcessNewChains(ctx context.Context, logger *zap.Logger, rdb *redis.Client, db *sql.DB) error {
	sugar := logger.Sugar()
	chains, err := getCachedPendingChains(ctx, rdb)
	if err != nil {
		sugar.Errorw("Failed to get cached pending chains", "err", err)
		return err
	}

	if len(chains) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(chains))
	doneChan := make(chan struct{})

	// Goroutine to collect errors and signal completion
	go func() {
		defer close(doneChan)
		for _, chain := range chains {
			wg.Add(1)
			go func(chain models.Chain) {
				defer wg.Done()
				client, err := initChainClient(&chain)
				if err != nil {
					errChan <- fmt.Errorf("failed to init chain client for chain %s: %v", chain.Name, err)
					return
				}
				go StartChainCrawler(ctx, sugar, client, db, &chain, rdb)
				sugar.Infow("Initiated new chain crawler", "chain", chain.Name)
			}(chain)
		}
		wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-doneChan:
		if len(errChan) > 0 {
			return <-errChan
		}
	}

	if err := deletePendingChainsFromCache(ctx, rdb); err != nil {
		sugar.Errorw("Failed to delete cached pending chains", "err", err)
		return err
	}

	return nil
}

func ProcessNewChainAssets(ctx context.Context, logger *zap.Logger, rdb *redis.Client) error {
	sugar := logger.Sugar()
	assets, err := getCachedPendingAssets(ctx, rdb)
	if err != nil {
		sugar.Errorw("Failed to get cached pending assets", "err", err)
		return err
	}

	if len(assets) == 0 {
		return nil
	}

	// Process assets in parallel using worker pool
	numWorkers := 1 // Adjust based on system capabilities
	assetChan := make(chan models.Asset, len(assets))
	errChan := make(chan error, 1)
	doneChan := make(chan bool, numWorkers)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go func() {
			for asset := range assetChan {
				select {
				case <-ctx.Done():
					doneChan <- true
					return
				default:
					models.ContractType[asset.ChainID][asset.CollectionAddress] = asset
					sugar.Infow("Processing new asset",
						"chain", asset.ChainID,
						"address", asset.CollectionAddress,
						"type", asset.Type,
					)
					// Add your asset processing logic here
				}
			}
			doneChan <- true
		}()
	}

	// Send assets to workers
	go func() {
		for _, asset := range assets {
			select {
			case <-ctx.Done():
				return
			case assetChan <- asset:
			}
		}
		close(assetChan)
	}()

	// Wait for workers to finish or error
	for i := 0; i < numWorkers; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			return err
		case <-doneChan:
			continue
		}
	}

	// Only delete from cache if all assets were processed successfully
	if err = deletePendingAssetsFromCache(ctx, rdb); err != nil {
		sugar.Errorw("Failed to delete cached pending assets", "err", err)
		return err
	}

	return nil
}

// Redis cache helper functions
func deleteChainFromCache(ctx context.Context, rdb *redis.Client, chainID int32) error {
	return rdb.Del(ctx, fmt.Sprintf("chain:%d", chainID)).Err()
}

func deleteChainAssetsFromCache(ctx context.Context, rdb *redis.Client, chainID int32) error {
	return rdb.Del(ctx, fmt.Sprintf("chain:%d:assets", chainID)).Err()
}

func setChainToCache(ctx context.Context, rdb *redis.Client, chain *models.Chain) error {
	// Marshal chain to JSON for Redis storage
	jsonData, err := json.Marshal(chain)
	if err != nil {
		return fmt.Errorf("failed to marshal chain: %v", err)
	}
	return rdb.Set(ctx, fmt.Sprintf("chain:%d", chain.ID), jsonData, 0).Err()
}

func setAssetsToCache(ctx context.Context, rdb *redis.Client, assets []models.Asset) error {
	// Marshal assets to JSON for Redis storage
	for _, asset := range assets {
		jsonData, err := json.Marshal(asset)
		if err != nil {
			return fmt.Errorf("failed to marshal asset: %v", err)
		}
		key := fmt.Sprintf("asset:%d:%s", asset.ChainID, asset.CollectionAddress)
		if err := rdb.Set(ctx, key, jsonData, 0).Err(); err != nil {
			return err
		}
	}
	return nil
}

func getCachedPendingChains(ctx context.Context, rdb *redis.Client) ([]models.Chain, error) {
	// Get pending chains from Redis
	keys, err := rdb.Keys(ctx, "pending_chain:*").Result()
	if err != nil {
		return nil, err
	}

	chains := make([]models.Chain, 0, len(keys))
	for _, key := range keys {
		jsonData, err := rdb.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, err
		}

		var chain models.Chain
		if err := json.Unmarshal([]byte(jsonData), &chain); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chain: %v", err)
		}
		chains = append(chains, chain)
	}
	return chains, nil
}

func deletePendingChainsFromCache(ctx context.Context, rdb *redis.Client) error {
	keys, err := rdb.Keys(ctx, "pending_chain:*").Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return rdb.Del(ctx, keys...).Err()
	}
	return nil
}

func getCachedPendingAssets(ctx context.Context, rdb *redis.Client) ([]models.Asset, error) {
	// Get pending assets from Redis
	keys, err := rdb.Keys(ctx, "pending_asset:*").Result()
	if err != nil {
		return nil, err
	}

	assets := make([]models.Asset, 0, len(keys))
	for _, key := range keys {
		jsonData, err := rdb.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, err
		}

		var asset models.Asset
		if err := json.Unmarshal([]byte(jsonData), &asset); err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %v", err)
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func deletePendingAssetsFromCache(ctx context.Context, rdb *redis.Client) error {
	keys, err := rdb.Keys(ctx, "pending_asset:*").Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return rdb.Del(ctx, keys...).Err()
	}
	return nil
}

// func ProcessCrawlingBackfillCollection(ctx context.Context, sugar *zap.Logger, db *sql.DB, queueClient *asynq.Client) error {
// Get all Backfill Collection with status CRAWLING
// crawlingBackfill, err := GetCrawlingBackfillCrawler(ctx, db)

// if err != nil {
// 	return err
// }

// for _, c := range crawlingBackfill {
// 	sugar.Info("%v", zap.Any("hello", c))
// 	chain, err := GetChainById(ctx, db, c.ChainID)
// 	if err != nil {
// 		return err
// 	}

// 	client, err := initChainClient(&chain)
// 	if err != nil {
// 		return err
// 	}
// 	go AddBackfillCrawlerTask(ctx, sugar, client, db, &chain, &c, queueClient)

// }
// return nil
// }

func ProcessCrawlingBackfillCollection(ctx context.Context, sugar *zap.Logger, db *sql.DB, queueClient *asynq.Client) error {
	// Run in the background using a goroutine
	go func() {
		sugar.Info("Starting ProcessCrawlingBackfillCollection in the background...")

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				sugar.Info("Context canceled, stopping backfill crawling")
				return
			case <-ticker.C:
				// Get all Backfill Collections with status CRAWLING
				crawlingBackfill, err := crawlerQuery.GetCrawlingBackfillCrawler(ctx, db)
				if err != nil {
					sugar.Error("Failed to get crawling backfill collections", zap.Error(err))
					continue
				}

				if len(crawlingBackfill) == 0 {
					sugar.Info("No backfill tasks found")
					continue
				}

				for _, c := range crawlingBackfill {
					sugar.Info("Dispatching backfill task", zap.Any("collection", c))
					chain, err := crawlerQuery.GetChainById(ctx, db, c.ChainID)
					if err != nil {
						sugar.Error("Failed to get chain by ID", zap.Error(err))
						continue
					}

					client, err := initChainClient(&chain)
					if err != nil {
						sugar.Error("Failed to initialize chain client", zap.Error(err))
						continue
					}

					go AddBackfillCrawlerTask(ctx, sugar, client, db, &chain, &c, queueClient)
				}
			}
		}
	}()

	// Return immediately so that the caller can continue running
	return nil
}
