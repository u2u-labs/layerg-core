package crawler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
	"go.uber.org/zap"
)

func CrawlSupportedChains(ctx context.Context, logger *zap.Logger, db *sql.DB, rdb *redis.Client) error {
	// Query all supported chains
	rows, err := db.QueryContext(ctx, "SELECT id, name, rpc_url FROM chains")
	if err != nil {
		return fmt.Errorf("error querying chains: %v", err)
	}
	defer rows.Close()

	var chains []models.Chain
	for rows.Next() {
		var chain models.Chain
		if err := rows.Scan(&chain.ID, &chain.Name, &chain.RpcUrl); err != nil {
			logger.Error("Error scanning chain row", zap.Error(err))
			continue
		}
		chains = append(chains, chain)
	}

	for _, c := range chains {
		// Delete chain from cache
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
                SELECT chain_id, collection_address, type 
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
				if err := rows.Scan(&asset.ChainID, &asset.CollectionAddress, &asset.Type); err != nil {
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

		// Start chain crawler
		client, err := initChainClient(&c)
		if err != nil {
			return err
		}
		go StartChainCrawler(ctx, logger.Sugar(), client, db, &c, rdb)
	}
	return nil
}

func ProcessNewChains(ctx context.Context, logger *zap.Logger, rdb *redis.Client, db *sql.DB) {
	sugar := logger.Sugar()
	chains, err := getCachedPendingChains(ctx, rdb)
	if err != nil {
		sugar.Errorw("ProcessNewChains failed to get cached pending chains", "err", err)
		return
	}
	if err = deletePendingChainsFromCache(ctx, rdb); err != nil {
		sugar.Errorw("ProcessNewChains failed to delete cached pending chains", "err", err)
		return
	}
	for _, c := range chains {
		client, err := initChainClient(&c)
		if err != nil {
			sugar.Errorw("ProcessNewChains failed to init chain client", "err", err, "chain", c)
			return
		}
		go StartChainCrawler(ctx, sugar, client, db, &c, rdb)
		sugar.Infow("Initiated new chain, start crawling", "chain", c)
	}
}

func ProcessNewChainAssets(ctx context.Context, logger *zap.Logger, rdb *redis.Client) {
	sugar := logger.Sugar()
	assets, err := getCachedPendingAssets(ctx, rdb)
	if err != nil {
		sugar.Errorw("ProcessNewChainAssets failed to get cached pending assets", "err", err)
		return
	}
	if err = deletePendingAssetsFromCache(ctx, rdb); err != nil {
		sugar.Errorw("ProcessNewChainAssets failed to delete cached pending assets", "err", err)
		return
	}
	for _, a := range assets {
		sugar.Infow("Initiated new assets, start crawling",
			"chain", a.ChainID,
			"address", a.CollectionAddress,
			"type", a.Type,
		)
	}
}

// Redis cache helper functions
func deleteChainFromCache(ctx context.Context, rdb *redis.Client, chainID int32) error {
	return rdb.Del(ctx, fmt.Sprintf("chain:%d", chainID)).Err()
}

func deleteChainAssetsFromCache(ctx context.Context, rdb *redis.Client, chainID int32) error {
	return rdb.Del(ctx, fmt.Sprintf("chain:%d:assets", chainID)).Err()
}

func setChainToCache(ctx context.Context, rdb *redis.Client, chain *models.Chain) error {
	return rdb.Set(ctx, fmt.Sprintf("chain:%d", chain.ID), chain, 0).Err()
}

func setAssetsToCache(ctx context.Context, rdb *redis.Client, assets []models.Asset) error {
	// Implement Redis set logic for assets
	return nil
}

func getCachedPendingChains(ctx context.Context, rdb *redis.Client) ([]models.Chain, error) {
	// Implement Redis get logic for pending chains
	return nil, nil
}

func deletePendingChainsFromCache(ctx context.Context, rdb *redis.Client) error {
	// Implement Redis delete logic for pending chains
	return nil
}

func getCachedPendingAssets(ctx context.Context, rdb *redis.Client) ([]models.Asset, error) {
	// Implement Redis get logic for pending assets
	return nil, nil
}

func deletePendingAssetsFromCache(ctx context.Context, rdb *redis.Client) error {
	// Implement Redis delete logic for pending assets
	return nil
}
