package crawler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	u2u "github.com/unicornultrafoundation/go-u2u"
	"github.com/unicornultrafoundation/go-u2u/common"
	utypes "github.com/unicornultrafoundation/go-u2u/core/types"
	"github.com/unicornultrafoundation/go-u2u/ethclient"
	"github.com/unicornultrafoundation/go-u2u/rpc"
	"go.uber.org/zap"

	"github.com/hibiken/asynq"
	"github.com/u2u-labs/layerg-core/server/crawler/utils"
	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
)

const (
	AssetTypeERC20   = "ERC20"
	AssetTypeERC721  = "ERC721"
	AssetTypeERC1155 = "ERC1155"
)

var contractType = make(map[int32]map[string]models.Asset)

var blockProcessingMutex sync.Mutex

func StartChainCrawler(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client, db *sql.DB, chain *models.Chain, rdb *redis.Client) {
	// sugar.Infow("Start chain crawler", "chain", chain)
	// timer := time.NewTimer(1)
	// defer timer.Stop()
	// errChan := make(chan error, 1)
	// go func() {
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		case <-timer.C:
	// 			go func() {
	// 				// Ensure only one instance runs at a time
	// 				blockProcessingMutex.Lock()
	// 				defer blockProcessingMutex.Unlock()

	// 				if err := ProcessLatestBlocks(ctx, sugar, client, db, chain, rdb); err != nil {
	// 					select {
	// 					case errChan <- err:
	// 					default:
	// 					}
	// 				}
	// 			}()
	// 			timer.Reset(1)
	// 		}
	// 	}
	// }()

	// select {
	// case <-ctx.Done():
	// 	sugar.Infow("Chain crawler stopped due to context cancellation", "chain", chain.Name)
	// 	return
	// case err := <-errChan:
	// 	sugar.Errorw("Chain crawler stopped due to error", "chain", chain.Name, "error", err)
	// 	return
	// }
	sugar.Infow("Start chain crawler", "chain", chain)
	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	timer := time.NewTimer(time.Second) // 1-second interval
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait() // Ensure all goroutines finish
			sugar.Infow("Chain crawler stopped due to context cancellation", "chain", chain.Name)
			return
		case <-timer.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Process blocks with mutex
				blockProcessingMutex.Lock()
				defer blockProcessingMutex.Unlock()

				if err := ProcessLatestBlocks(ctx, sugar, client, db, chain, rdb); err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
			}()
			timer.Reset(time.Second)
		case err := <-errChan:
			wg.Wait() // Wait for all workers to complete
			sugar.Errorw("Chain crawler stopped due to error", "chain", chain.Name, "error", err)
			return
		}
	}
}

func ProcessLatestBlocks(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client, db *sql.DB, chain *models.Chain, rdb *redis.Client) error {
	latest, err := client.BlockNumber(ctx)
	if err != nil {
		sugar.Errorw("Failed to fetch latest blocks", "err", err, "chain", chain)
		return err
	}

	if chain.LatestBlock >= int64(latest) {
		return nil // Nothing to process
	}

	// Use a worker pool to process blocks in parallel
	numWorkers := 2 // Adjust based on system capabilities
	blockChan := make(chan int64, numWorkers)
	errChan := make(chan error, 1)
	doneChan := make(chan bool, numWorkers)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go func() {
			for blockNum := range blockChan {
				select {
				case <-ctx.Done():
					doneChan <- true
					return
				default:
					if blockNum%50 == 0 {
						sugar.Infow("Importing block receipts", "chain", chain.Name, "block", blockNum, "latest", latest)
					}

					receipts, err := client.BlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockNum)))
					if err != nil {
						select {
						case errChan <- fmt.Errorf("failed to fetch block receipts at height %d: %v", blockNum, err):
						default:
						}
						doneChan <- true
						return
					}
					// Update latest block processed
					query := "UPDATE chains SET latest_block = $1 WHERE id = $2"
					_, err = db.ExecContext(ctx, query, latest, chain.ID)
					if err != nil {
						sugar.Errorw("Failed to update chain latest blocks in DB", "err", err, "chain", chain)
						select {
						case errChan <- fmt.Errorf("failed to update block at height %d: %v", blockNum, err):
						default:
						}
						doneChan <- true
						return
					}

					if err = FilterEvents(ctx, sugar, db, client, chain, rdb, receipts); err != nil {
						select {
						case errChan <- fmt.Errorf("failed to filter events at height %d: %v", blockNum, err):
						default:
						}
						doneChan <- true
						return
					}
				}
			}
			doneChan <- true
		}()
	}

	// Send blocks to workers
	go func() {
		for i := chain.LatestBlock + 1; i <= int64(latest); i++ {
			select {
			case <-ctx.Done():
				return
			case blockChan <- i:
			}
		}
		close(blockChan)
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

	chain.LatestBlock = int64(latest)
	return nil
}

func FilterEvents(ctx context.Context, sugar *zap.SugaredLogger, db *sql.DB, client *ethclient.Client,
	chain *models.Chain, rdb *redis.Client, receipts utypes.Receipts) error {

	for _, r := range receipts {
		for _, l := range r.Logs {
			// Get asset type from database
			var assetType string
			err := db.QueryRowContext(ctx,
				"SELECT type FROM assets WHERE chain_id = $1 AND collection_address = $2",
				chain.ID, l.Address.Hex()).Scan(&assetType)
			if err == sql.ErrNoRows {
				continue // Skip if asset not found
			}
			if err != nil {
				sugar.Errorw("Error querying asset type", "err", err)
				return err
			}

			switch assetType {
			case AssetTypeERC20:
				if err := handleErc20Transfer(ctx, sugar, db, client, chain, rdb, l); err != nil {
					sugar.Errorw("handleErc20Transfer", "err", err)
					return err
				}
			case AssetTypeERC721:
				if err := handleErc721Transfer(ctx, sugar, db, client, chain, rdb, l); err != nil {
					sugar.Errorw("handleErc721Transfer", "err", err)
					return err
				}
			case AssetTypeERC1155:
				if l.Topics[0].Hex() == utils.TransferSingleSig {
					if err := handleErc1155TransferSingle(ctx, sugar, db, client, chain, rdb, l); err != nil {
						sugar.Errorw("handleErc1155TransferSingle", "err", err)
						return err
					}
				}
				if l.Topics[0].Hex() == utils.TransferBatchSig {
					if err := handleErc1155TransferBatch(ctx, sugar, db, client, chain, rdb, l); err != nil {
						sugar.Errorw("handleErc1155TransferBatch", "err", err)
						return err
					}
				}
			}
		}
	}
	return nil
}

func handleErc20Transfer(ctx context.Context, sugar *zap.SugaredLogger, db *sql.DB, client *ethclient.Client,
	chain *models.Chain, rdb *redis.Client, l *utypes.Log) error {
	if l.Topics[0].Hex() != utils.TransferEventSig {
		return nil
	}

	// Unpack the log data
	var event utils.Erc20TransferEvent
	err := utils.ERC20ABI.UnpackIntoInterface(&event, "Transfer", l.Data)
	if err != nil {
		sugar.Fatalf("Failed to unpack log: %v", err)
		return err
	}

	// Decode the indexed fields manually
	event.From = common.BytesToAddress(l.Topics[1].Bytes())
	event.To = common.BytesToAddress(l.Topics[2].Bytes())
	amount := event.Value.String()

	// Get asset ID
	var assetID string
	err = db.QueryRowContext(ctx,
		"SELECT id FROM assets WHERE chain_id = $1 AND collection_address = $2",
		chain.ID, l.Address.Hex()).Scan(&assetID)
	if err != nil {
		return fmt.Errorf("error getting asset ID: %v", err)
	}

	// Get balances for both addresses
	fromBalance, err := getErc20Balance(ctx, sugar, client, &l.Address, &event.From)
	if err != nil {
		sugar.Warnw("Failed to get ERC20 balance", "err", err, "address", event.From.Hex())
		fromBalance = big.NewInt(0)
	}

	toBalance, err := getErc20Balance(ctx, sugar, client, &l.Address, &event.To)
	if err != nil {
		sugar.Warnw("Failed to get ERC20 balance", "err", err, "address", event.To.Hex())
		toBalance = big.NewInt(0)
	}

	// Retry transaction loop
	// maxRetries := 3
	// for i := 0; i < maxRetries; i++ {
	// 	err := func() error {
	// 		// Start transaction
	// 		tx, err := db.BeginTx(ctx, &sql.TxOptions{
	// 			Isolation: sql.LevelSerializable,
	// 		})
	// 		if err != nil {
	// 			return err
	// 		}
	// 		defer tx.Rollback()

	// 		// Add transaction history
	// 		_, err = tx.ExecContext(ctx, `
	// 			INSERT INTO onchain_histories ("from", "to", asset_id, token_id, amount, tx_hash, timestamp)
	// 			VALUES ($1, $2, $3, $4, $5, $6, $7)
	// 		`, event.From.Hex(), event.To.Hex(), assetID, "0", amount, l.TxHash.Hex(), time.Now())
	// 		if err != nil {
	// 			return err
	// 		}

	// 		// Update or insert holder records
	// 		_, err = tx.ExecContext(ctx, `
	// 			INSERT INTO erc_20_collection_assets (asset_id, chain_id, owner, balance)
	// 			VALUES ($1, $2, $3, $4)
	// 			ON CONFLICT ON CONSTRAINT erc_20_collection_assets_owner_key DO UPDATE SET
	// 				balance = $4,
	// 				updated_at = CURRENT_TIMESTAMP
	// 		`, assetID, chain.ID, event.From.Hex(), fromBalance.String())
	// 		if err != nil {
	// 			sugar.Infow("error while inserting 2", err)
	// 			return err
	// 		}

	// 		_, err = tx.ExecContext(ctx, `
	// 			INSERT INTO erc_20_collection_assets (asset_id, chain_id, owner, balance)
	// 			VALUES ($1, $2, $3, $4)
	// 			ON CONFLICT ON CONSTRAINT erc_20_collection_assets_owner_key DO UPDATE SET
	// 				balance = $4,
	// 				updated_at = CURRENT_TIMESTAMP
	// 		`, assetID, chain.ID, event.To.Hex(), toBalance.String())
	// 		if err != nil {
	// 			sugar.Infow("error while inserting 2", err, event.To.String(), assetID, toBalance.String())
	// 			return err
	// 		}

	// 		return tx.Commit()
	// 	}()

	// 	if err == nil {
	// 		break // Transaction succeeded
	// 	}

	// 	if i == maxRetries-1 {
	// 		return fmt.Errorf("transaction failed after %d retries: %v", maxRetries, err)
	// 	}

	// 	// Check if it's a retry-able error
	// 	if strings.Contains(err.Error(), "RETRY_WRITE_TOO_OLD") || strings.Contains(err.Error(), "restart transaction") {
	// 		// sugar.Infow("Retrying transaction due to conflict", "attempt", i+1, "error", err)
	// 		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
	// 		continue
	// 	}

	// 	return err // Non-retryable error
	// }

	// Add transaction history
	// _, err = db.ExecContext(ctx, `
	// 			INSERT INTO onchain_histories ("from", "to", asset_id, token_id, amount, tx_hash, timestamp)
	// 			VALUES ($1, $2, $3, $4, $5, $6, $7)
	// 		`, event.From.Hex(), event.To.Hex(), assetID, "0", amount, l.TxHash.Hex(), time.Now())
	_, err = AddOnchainTransaction(ctx, db, models.AddOnchainTransactionParams{
		From:      event.From.Hex(),
		To:        event.To.Hex(),
		AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
		TokenID:   "0",
		Amount:    amount,
		TxHash:    l.TxHash.Hex(),
		Timestamp: time.Now(),
	})
	if err != nil {
		return err
	}

	// Update or insert holder records
	// _, err = db.ExecContext(ctx, `
	// 			INSERT INTO erc_20_collection_assets (asset_id, chain_id, owner, balance)
	// 			VALUES ($1, $2, $3, $4)
	// 			ON CONFLICT ON CONSTRAINT erc_20_collection_assets_owner_key DO UPDATE SET
	// 				balance = $4,
	// 				updated_at = CURRENT_TIMESTAMP
	// 		`, assetID, chain.ID, event.From.Hex(), fromBalance.String())
	// if err != nil {
	// 	sugar.Infow("error while inserting 2", err)
	// 	return err
	// }

	err = Add20Asset(ctx, db, models.Add20AssetParams{
		AssetID: assetID,
		ChainID: chain.ID,
		Owner:   event.From.Hex(),
		Balance: fromBalance.String(),
	})
	if err != nil {
		return err
	}

	// _, err = db.ExecContext(ctx, `
	// 			INSERT INTO erc_20_collection_assets (asset_id, chain_id, owner, balance)
	// 			VALUES ($1, $2, $3, $4)
	// 			ON CONFLICT ON CONSTRAINT erc_20_collection_assets_owner_key DO UPDATE SET
	// 				balance = $4,
	// 				updated_at = CURRENT_TIMESTAMP
	// 		`, assetID, chain.ID, event.To.Hex(), toBalance.String())
	// if err != nil {
	// 	sugar.Infow("error while inserting 2", err, event.To.String(), assetID, toBalance.String())
	// 	return err
	// }

	err = Add20Asset(ctx, db, models.Add20AssetParams{
		AssetID: assetID,
		ChainID: chain.ID,
		Owner:   event.To.Hex(),
		Balance: toBalance.String(),
	})
	if err != nil {
		return err
	}

	// Cache the transaction in Redis
	key := fmt.Sprintf("history:%s:%s", chain.ID, l.TxHash.Hex())
	return rdb.Set(ctx, key, amount, 24*time.Hour).Err()
}

func getErc20Balance(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client,
	contractAddress *common.Address, ownerAddress *common.Address) (*big.Int, error) {
	// Prepare the function call data
	data, err := utils.ERC20ABI.Pack("balanceOf", ownerAddress)
	if err != nil {
		sugar.Errorf("Failed to pack data for balanceOf: %v", err)
		return nil, err
	}

	// Call the contract
	msg := u2u.CallMsg{
		To:   contractAddress,
		Data: data,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		sugar.Errorf("Failed to call contract: %v", err)
		return nil, err
	}

	// Unpack the result to get the balance
	var balance *big.Int
	err = utils.ERC20ABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		sugar.Errorf("Failed to unpack balanceOf: %v", err)
		return nil, err
	}

	return balance, nil
}

func getErc721OwnerOf(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client,
	contractAddress *common.Address, tokenId *big.Int) (common.Address, error) {

	// Prepare the function call data
	data, err := utils.ERC721ABI.Pack("ownerOf", tokenId)
	if err != nil {
		sugar.Errorf("Failed to pack data for balanceOf: %v", err)
		return common.Address{}, err
	}

	// Call the contract
	msg := u2u.CallMsg{
		To:   contractAddress,
		Data: data,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		sugar.Errorf("Failed to call contract: %v", err)
		return common.Address{}, err
	}

	// Unpack the result to get the balance
	var owner common.Address
	err = utils.ERC721ABI.UnpackIntoInterface(&owner, "ownerOf", result)

	if err != nil {
		sugar.Errorf("Failed to unpack ownerOf: %v", err)
		return common.Address{}, err
	}

	return owner, nil
}

func getErc721TokenURI(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client,
	contractAddress *common.Address, tokenId *big.Int) (string, error) {

	// / Prepare the function call data
	data, err := utils.ERC721ABI.Pack("tokenURI", tokenId)
	if err != nil {
		sugar.Errorf("Failed to pack data for tokenURI: %v", err)
		return "", err
	}

	// Call the contract
	msg := u2u.CallMsg{
		To:   contractAddress,
		Data: data,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		sugar.Errorf("Failed to call contract: %v", err)
		return "", err
	}

	// Unpack the result to get the token URI
	var tokenURI string
	err = utils.ERC721ABI.UnpackIntoInterface(&tokenURI, "tokenURI", result)
	if err != nil {
		sugar.Errorf("Failed to unpack tokenURI: %v", err)
		return "", err
	}
	return tokenURI, nil
}

func getErc1155TokenURI(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client,
	contractAddress *common.Address, tokenId *big.Int) (string, error) {
	// Prepare the function call data
	data, err := utils.ERC1155ABI.Pack("uri", tokenId)

	if err != nil {
		sugar.Errorf("Failed to pack data for tokenURI: %v", err)
		return "", err
	}

	// Call the contract
	msg := u2u.CallMsg{
		To:   contractAddress,
		Data: data,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		sugar.Errorf("Failed to call contract: %v", err)
		return "", err
	}

	// Unpack the result to get the token URI
	var tokenURI string
	err = utils.ERC1155ABI.UnpackIntoInterface(&tokenURI, "uri", result)
	if err != nil {
		sugar.Errorf("Failed to unpack tokenURI: %v", err)
		return "", err
	}
	// Replace {id} in the URI template with the actual token ID in hexadecimal form
	tokenIDHex := fmt.Sprintf("%x", tokenId)
	tokenURI = replaceTokenIDPlaceholder(tokenURI, tokenIDHex)
	return tokenURI, nil
}

func replaceTokenIDPlaceholder(uriTemplate, tokenIDHex string) string {
	return strings.ReplaceAll(uriTemplate, "{id}", tokenIDHex)

}

func retrieveNftMetadata(tokenURI string) ([]byte, error) {
	res, err := http.Get(tokenURI)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(res.Body)
}

func getErc1155BalanceOf(ctx context.Context, sugar *zap.SugaredLogger, client *ethclient.Client,
	contractAddress *common.Address, ownerAddress *common.Address, tokenId *big.Int) (*big.Int, error) {
	// Prepare the function call data
	data, err := utils.ERC1155ABI.Pack("balanceOf", ownerAddress, tokenId)
	if err != nil {
		sugar.Errorf("Failed to pack data for balanceOf: %v", err)
		return nil, err
	}

	// Call the contract
	msg := u2u.CallMsg{
		To:   contractAddress,
		Data: data,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		sugar.Errorf("Failed to call contract: %v", err)
		return nil, err
	}

	// Unpack the result to get the balance
	var balance *big.Int
	err = utils.ERC1155ABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		sugar.Errorf("Failed to unpack balanceOf: %v", err)
		return nil, err
	}

	return balance, nil
}

func handleErc721Transfer(ctx context.Context, sugar *zap.SugaredLogger, db *sql.DB, client *ethclient.Client,
	chain *models.Chain, rdb *redis.Client, l *utypes.Log) error {
	if l.Topics[0].Hex() != utils.TransferEventSig {
		return nil
	}

	// Decode the indexed fields manually
	event := utils.Erc721TransferEvent{
		From:    common.BytesToAddress(l.Topics[1].Bytes()),
		To:      common.BytesToAddress(l.Topics[2].Bytes()),
		TokenID: l.Topics[3].Big(),
	}

	// Get asset ID
	var assetID string
	err := db.QueryRowContext(ctx,
		"SELECT id FROM assets WHERE chain_id = $1 AND collection_address = $2",
		chain.ID, l.Address.Hex()).Scan(&assetID)
	if err != nil {
		return fmt.Errorf("error getting asset ID: %v", err)
	}

	// Get token URI
	uri, err := getErc721TokenURI(ctx, sugar, client, &l.Address, event.TokenID)
	if err != nil {
		sugar.Warnw("Failed to get ERC721 token URI", "err", err, "tokenID", event.TokenID)
		uri = "" // Continue even if URI fetch fails
	}

	// Retry transaction loop
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := func() error {
			// Start transaction
			tx, err := db.BeginTx(ctx, &sql.TxOptions{
				Isolation: sql.LevelSerializable,
			})
			if err != nil {
				return err
			}
			defer tx.Rollback()

			// Add transaction history
			// _, err = tx.ExecContext(ctx, `
			// 	INSERT INTO onchain_histories ("from", "to", asset_id, token_id, amount, tx_hash, timestamp)
			// 	VALUES ($1, $2, $3, $4, $5, $6, $7)
			// `, event.From.Hex(), event.To.Hex(), assetID, event.TokenID.String(), "1", l.TxHash.Hex(), time.Now())
			// if err != nil {
			// 	return err
			// }

			_, err = AddOnchainTransaction(ctx, db, models.AddOnchainTransactionParams{
				From:      event.From.Hex(),
				To:        event.To.Hex(),
				AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
				TokenID:   event.TokenID.String(),
				Amount:    "1",
				TxHash:    l.TxHash.Hex(),
				Timestamp: time.Now(),
			})
			if err != nil {
				return err
			}

			// Update NFT ownership
			// _, err = tx.ExecContext(ctx, `
			// 	INSERT INTO erc_721_collection_assets (asset_id, chain_id, token_id, owner, attributes)
			// 	VALUES ($1, $2, $3, $4, $5)
			// 	ON CONFLICT (asset_id, chain_id, token_id) DO UPDATE
			// 	SET owner = $4, attributes = $5, updated_at = CURRENT_TIMESTAMP
			// `, assetID, chain.ID, event.TokenID.String(), event.To.Hex(), uri)
			// if err != nil {
			// 	return err
			// }

			err = Add721Asset(ctx, db, models.Add721AssetParams{
				AssetID: assetID,
				ChainID: chain.ID,
				TokenID: event.TokenID.String(),
				Owner:   event.To.Hex(),
				Attributes: sql.NullString{
					Valid:  len(uri) > 0,
					String: uri,
				},
			})
			if err != nil {
				return err
			}

			return tx.Commit()
		}()

		if err == nil {
			break // Transaction succeeded
		}

		if i == maxRetries-1 {
			return fmt.Errorf("transaction failed after %d retries: %v", maxRetries, err)
		}

		// Check if it's a retry-able error
		if strings.Contains(err.Error(), "RETRY_WRITE_TOO_OLD") || strings.Contains(err.Error(), "restart transaction") {
			sugar.Infow("Retrying transaction due to conflict", "attempt", i+1, "error", err)
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		return err // Non-retryable error
	}

	// Cache the transaction in Redis
	key := fmt.Sprintf("history:%s:%s", chain.ID, l.TxHash.Hex())
	return rdb.Set(ctx, key, event.TokenID.String(), 24*time.Hour).Err()
}

func handleErc1155TransferSingle(ctx context.Context, sugar *zap.SugaredLogger, db *sql.DB, client *ethclient.Client,
	chain *models.Chain, rdb *redis.Client, l *utypes.Log) error {
	// Decode TransferSingle log
	var event utils.Erc1155TransferSingleEvent
	err := utils.ERC1155ABI.UnpackIntoInterface(&event, "TransferSingle", l.Data)
	if err != nil {
		sugar.Errorw("Failed to unpack TransferSingle log:", "err", err)
		return err
	}

	// Decode the indexed fields
	event.Operator = common.BytesToAddress(l.Topics[1].Bytes())
	event.From = common.BytesToAddress(l.Topics[2].Bytes())
	event.To = common.BytesToAddress(l.Topics[3].Bytes())

	// Get asset ID
	var assetID string
	err = db.QueryRowContext(ctx,
		"SELECT id FROM assets WHERE chain_id = $1 AND collection_address = $2",
		chain.ID, l.Address.Hex()).Scan(&assetID)
	if err != nil {
		return fmt.Errorf("error getting asset ID: %v", err)
	}

	// Get token URI and balance
	uri, err := getErc1155TokenURI(ctx, sugar, client, &l.Address, event.Id)
	if err != nil {
		sugar.Warnw("Failed to get ERC1155 token URI", "err", err, "tokenID", event.Id)
		uri = "" // Continue even if URI fetch fails
	}

	balance, err := getErc1155BalanceOf(ctx, sugar, client, &l.Address, &event.To, event.Id)
	if err != nil {
		sugar.Warnw("Failed to get ERC1155 balance", "err", err, "tokenID", event.Id)
		balance = big.NewInt(0) // Use 0 if balance fetch fails
	}

	// Retry transaction loop
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := func() error {
			// Start transaction
			tx, err := db.BeginTx(ctx, &sql.TxOptions{
				Isolation: sql.LevelSerializable,
			})
			if err != nil {
				return err
			}
			defer tx.Rollback()

			// Add transaction history
			// _, err = tx.ExecContext(ctx, `
			// 	INSERT INTO onchain_histories ("from", "to", asset_id, token_id, amount, tx_hash, timestamp)
			// 	VALUES ($1, $2, $3, $4, $5, $6, $7)
			// `, event.From.Hex(), event.To.Hex(), assetID, event.Id.String(), event.Value.String(), l.TxHash.Hex(), time.Now())
			// if err != nil {
			// 	return err
			// }

			_, err = AddOnchainTransaction(ctx, db, models.AddOnchainTransactionParams{
				From:      event.From.Hex(),
				To:        event.To.Hex(),
				AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
				TokenID:   event.Id.String(),
				Amount:    event.Value.String(),
				TxHash:    l.TxHash.Hex(),
				Timestamp: time.Now(),
			})
			if err != nil {
				return err
			}

			// Update token ownership and balance
			// _, err = tx.ExecContext(ctx, `
			// 	INSERT INTO erc_1155_collection_assets (asset_id, chain_id, token_id, owner, balance, attributes)
			// 	VALUES ($1, $2, $3, $4, $5, $6)
			// 	ON CONFLICT (asset_id, chain_id, token_id, owner) DO UPDATE
			// 	SET balance = $5, attributes = $6, updated_at = CURRENT_TIMESTAMP
			// `, assetID, chain.ID, event.Id.String(), event.To.Hex(), balance.String(), uri)
			err = Add1155Asset(ctx, db, models.Add1155AssetParams{
				AssetID: assetID,
				ChainID: chain.ID,
				TokenID: event.Id.String(),
				Owner:   event.To.Hex(),
				Balance: balance.String(),
				Attributes: sql.NullString{
					Valid:  len(uri) > 0,
					String: uri,
				},
			})
			if err != nil {
				return err
			}

			return tx.Commit()
		}()

		if err == nil {
			break // Transaction succeeded
		}

		if i == maxRetries-1 {
			return fmt.Errorf("transaction failed after %d retries: %v", maxRetries, err)
		}

		// Check if it's a retry-able error
		if strings.Contains(err.Error(), "RETRY_WRITE_TOO_OLD") || strings.Contains(err.Error(), "restart transaction") {
			sugar.Infow("Retrying transaction due to conflict", "attempt", i+1, "error", err)
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		return err // Non-retryable error
	}

	// Cache the transaction in Redis
	key := fmt.Sprintf("history:%s:%s", chain.ID, l.TxHash.Hex())
	return rdb.Set(ctx, key, event.Value.String(), 24*time.Hour).Err()
}

func handleErc1155TransferBatch(ctx context.Context, sugar *zap.SugaredLogger, db *sql.DB, client *ethclient.Client,
	chain *models.Chain, rdb *redis.Client, l *utypes.Log) error {
	// Decode TransferBatch log
	var event utils.Erc1155TransferBatchEvent
	err := utils.ERC1155ABI.UnpackIntoInterface(&event, "TransferBatch", l.Data)
	if err != nil {
		sugar.Errorw("Failed to unpack TransferBatch log:", "err", err)
		return err
	}

	// Decode the indexed fields
	event.Operator = common.BytesToAddress(l.Topics[1].Bytes())
	event.From = common.BytesToAddress(l.Topics[2].Bytes())
	event.To = common.BytesToAddress(l.Topics[3].Bytes())

	// Get asset ID
	var assetID string
	err = db.QueryRowContext(ctx,
		"SELECT id FROM assets WHERE chain_id = $1 AND collection_address = $2",
		chain.ID, l.Address.Hex()).Scan(&assetID)
	if err != nil {
		return fmt.Errorf("error getting asset ID: %v", err)
	}

	// Retry transaction loop
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := func() error {
			// Start transaction
			tx, err := db.BeginTx(ctx, &sql.TxOptions{
				Isolation: sql.LevelSerializable,
			})
			if err != nil {
				return err
			}
			defer tx.Rollback()

			// Process each token in the batch
			for i := range event.Ids {
				// Get token URI and balance
				uri, err := getErc1155TokenURI(ctx, sugar, client, &l.Address, event.Ids[i])
				if err != nil {
					sugar.Warnw("Failed to get ERC1155 token URI", "err", err, "tokenID", event.Ids[i])
					uri = "" // Continue even if URI fetch fails
				}

				balance, err := getErc1155BalanceOf(ctx, sugar, client, &l.Address, &event.To, event.Ids[i])
				if err != nil {
					sugar.Warnw("Failed to get ERC1155 balance", "err", err, "tokenID", event.Ids[i])
					balance = big.NewInt(0) // Use 0 if balance fetch fails
				}

				// Add transaction history
				// _, err = tx.ExecContext(ctx, `
				// 	INSERT INTO onchain_histories ("from", "to", asset_id, token_id, amount, tx_hash, timestamp)
				// 	VALUES ($1, $2, $3, $4, $5, $6, $7)
				// `, event.From.Hex(), event.To.Hex(), assetID, event.Ids[i].String(), event.Values[i].String(), l.TxHash.Hex(), time.Now())
				// if err != nil {
				// 	return err
				// }

				_, err = AddOnchainTransaction(ctx, db, models.AddOnchainTransactionParams{
					From:      event.From.Hex(),
					To:        event.To.Hex(),
					AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
					TokenID:   event.Ids[i].String(),
					Amount:    event.Values[i].String(),
					TxHash:    l.TxHash.Hex(),
					Timestamp: time.Now(),
				})
				if err != nil {
					return err
				}

				// Update token ownership and balance
				// _, err = tx.ExecContext(ctx, `
				// 	INSERT INTO erc_1155_collection_assets (asset_id, chain_id, token_id, owner, balance, attributes)
				// 	VALUES ($1, $2, $3, $4, $5, $6)
				// 	ON CONFLICT (asset_id, chain_id, token_id, owner) DO UPDATE
				// 	SET balance = $5, attributes = $6, updated_at = CURRENT_TIMESTAMP
				// `, assetID, chain.ID, event.Ids[i].String(), event.To.Hex(), balance.String(), uri)

				err = Add1155Asset(ctx, db, models.Add1155AssetParams{
					AssetID: assetID,
					ChainID: chain.ID,
					TokenID: event.Ids[i].String(),
					Owner:   event.To.Hex(),
					Balance: balance.String(),
					Attributes: sql.NullString{
						Valid:  len(uri) > 0,
						String: uri,
					},
				})
				if err != nil {
					return err
				}
			}

			return tx.Commit()
		}()

		if err == nil {
			break // Transaction succeeded
		}

		if i == maxRetries-1 {
			return fmt.Errorf("transaction failed after %d retries: %v", maxRetries, err)
		}

		// Check if it's a retry-able error
		if strings.Contains(err.Error(), "RETRY_WRITE_TOO_OLD") || strings.Contains(err.Error(), "restart transaction") {
			sugar.Infow("Retrying transaction due to conflict", "attempt", i+1, "error", err)
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // Exponential backoff
			continue
		}

		return err // Non-retryable error
	}

	// Cache the transaction in Redis
	key := fmt.Sprintf("history:%s:%s", chain.ID, l.TxHash.Hex())
	return rdb.Set(ctx, key, "batch", 24*time.Hour).Err()
}

func AddBackfillCrawlerTask(ctx context.Context, sugar *zap.Logger, client *ethclient.Client, db *sql.DB, chain *models.Chain, c *models.GetCrawlingBackfillCrawlerRow, queueClient *asynq.Client) error {
	blockRangeScan := int64(100) * 100
	if c.CurrentBlock%blockRangeScan == 0 {
		sugar.Info("Backfill crawler")
	}

	timer := time.NewTimer(time.Duration(chain.BlockTime) * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			task, err := NewBackfillCollectionTask(c)
			if err != nil {
				sugar.Error("Could not create task", zap.Error(err))
				return err
			}
			info, err := queueClient.Enqueue(task)
			sugar.Info("%v", zap.Any("queue here", info))
			if err != nil {
				sugar.Error("Could not enqueue task", zap.Error(err))
				return err
			}
		}
	}
}

func xf(bf *models.GetCrawlingBackfillCrawlerRow) (*asynq.Task, error) {

	payload, err := json.Marshal(bf)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(BackfillCollection, payload), nil
}

func handleErc20BackFill(ctx context.Context, sugar *zap.SugaredLogger, q *sql.DB, client *ethclient.Client,
	chain *models.Chain, logs []utypes.Log) error {

	// Initialize the AddressSet
	addressSet := utils.NewAddressSet()

	if len(logs) == 0 {
		return nil
	}

	var contractAddress *common.Address
	for _, l := range logs {
		contractAddress = &l.Address
		var event utils.Erc20TransferEvent

		err := utils.ERC20ABI.UnpackIntoInterface(&event, "Transfer", l.Data)
		if err != nil {
			sugar.Fatalf("Failed to unpack log: %v", err)
			return err
		}

		if l.Topics[0].Hex() != utils.TransferEventSig {
			return nil
		}

		event.From = common.BytesToAddress(l.Topics[1].Bytes())
		event.To = common.BytesToAddress(l.Topics[2].Bytes())
		amount := event.Value.String()

		_, err = AddOnchainTransaction(ctx, q, models.AddOnchainTransactionParams{
			From:      event.From.Hex(),
			To:        event.To.Hex(),
			AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
			TokenID:   "0",
			Amount:    amount,
			TxHash:    l.TxHash.Hex(),
			Timestamp: time.Now(),
		})

		// adding sender and receiver to the address set
		addressSet.AddAddress(event.From)
		addressSet.AddAddress(event.To)
	}

	rpcClient, _ := utils.InitNewRPCClient(chain.RpcUrl)

	addressList := addressSet.GetAddresses()

	results := make([]string, len(addressList))
	calls := make([]rpc.BatchElem, len(addressList))

	for i, addr := range addressList {
		// Pack the data for the balanceOf function
		data, err := utils.ERC20ABI.Pack("balanceOf", addr)
		if err != nil {
			sugar.Errorf("Failed to pack data for balanceOf: %v", err)
			return err
		}

		encodedData := "0x" + common.Bytes2Hex(data)

		// Append the BatchElem for the eth_call
		calls[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   contractAddress,
					"data": encodedData,
				},
				"latest",
			},
			Result: &results[i],
		}
	}

	// Execute batch call
	if err := rpcClient.BatchCallContext(ctx, calls); err != nil {
		log.Fatalf("Failed to execute batch call: %v", err)
	}

	// Iterate over the results and update the balances
	for i, result := range results {
		var balance *big.Int

		utils.ERC20ABI.UnpackIntoInterface(&balance, "balanceOf", common.FromHex(result))

		if err := Add20Asset(ctx, q, models.Add20AssetParams{
			AssetID: contractType[chain.ID][contractAddress.Hex()].ID,
			ChainID: chain.ID,
			Owner:   addressList[i].Hex(),
			Balance: balance.String(),
		}); err != nil {
			return err
		}
	}

	addressSet.Reset()
	return nil
}

func handleErc721BackFill(ctx context.Context, sugar *zap.SugaredLogger, q *sql.DB, client *ethclient.Client,
	chain *models.Chain, logs []utypes.Log) error {

	// Initialize the NewTokenIdSet
	tokenIdSet := utils.NewTokenIdSet()

	if len(logs) == 0 {
		return nil
	}

	var contractAddress *common.Address
	for _, l := range logs {
		contractAddress = &l.Address

		// Decode the indexed fields manually
		event := utils.Erc721TransferEvent{
			From:    common.BytesToAddress(l.Topics[1].Bytes()),
			To:      common.BytesToAddress(l.Topics[2].Bytes()),
			TokenID: l.Topics[3].Big(),
		}
		_, err := AddOnchainTransaction(ctx, q, models.AddOnchainTransactionParams{
			From:      event.From.Hex(),
			To:        event.To.Hex(),
			AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
			TokenID:   event.TokenID.String(),
			Amount:    "0",
			TxHash:    l.TxHash.Hex(),
			Timestamp: time.Now(),
		})
		if err != nil {
			return err
		}

		// adding token Id
		tokenIdSet.AddTokenId(event.TokenID)
	}

	rpcClient, _ := utils.InitNewRPCClient(chain.RpcUrl)

	tokenIdList := tokenIdSet.GetTokenIds()

	results := make([]string, len(tokenIdList)*2)
	calls := make([]rpc.BatchElem, len(tokenIdList)*2)

	for i, tokenId := range tokenIdList {
		// Pack the data for the tokenURI function
		data, err := utils.ERC721ABI.Pack("tokenURI", tokenId)
		if err != nil {
			sugar.Errorf("Failed to pack data for tokenURI: %v", err)
			return err
		}

		encodedUriData := "0x" + common.Bytes2Hex(data)

		// Append the BatchElem for the eth_call
		calls[2*i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   contractAddress,
					"data": encodedUriData,
				},
				"latest",
			},
			Result: &results[2*i],
		}

		// Pack the data for the ownerOf function
		ownerData, err := utils.ERC721ABI.Pack("ownerOf", tokenId)
		if err != nil {
			sugar.Errorf("Failed to pack data for ownerOf: %v", err)
			return err
		}

		encodedOwnerData := "0x" + common.Bytes2Hex(ownerData)

		// Append the BatchElem for the eth_call
		calls[2*i+1] = rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   contractAddress,
					"data": encodedOwnerData,
				},
				"latest",
			},
			Result: &results[2*i+1],
		}

	}

	// Execute batch call
	if err := rpcClient.BatchCallContext(ctx, calls); err != nil {
		log.Fatalf("Failed to execute batch call: %v", err)
	}

	// Iterate over the results and update the balances
	for i := 0; i < len(results); i += 2 {
		var uri string
		var owner common.Address
		utils.ERC721ABI.UnpackIntoInterface(&uri, "tokenURI", common.FromHex(results[i]))
		utils.ERC721ABI.UnpackIntoInterface(&owner, "ownerOf", common.FromHex(results[i+1]))

		if err := Add721Asset(ctx, q, models.Add721AssetParams{
			AssetID: contractType[chain.ID][contractAddress.Hex()].ID,
			ChainID: chain.ID,
			TokenID: tokenIdList[i/2].String(),
			Owner:   owner.Hex(),
			Attributes: sql.NullString{
				String: uri,
				Valid:  len(uri) > 0,
			},
		}); err != nil {
			return err
		}
	}

	tokenIdSet.Reset()
	return nil
}

func handleErc1155Backfill(ctx context.Context, sugar *zap.SugaredLogger, q *sql.DB, client *ethclient.Client,
	chain *models.Chain, logs []utypes.Log) error {

	// Initialize the NewTokenIdSet
	tokenIdContractAddressSet := utils.NewTokenIdContractAddressSet()

	if len(logs) == 0 {
		return nil
	}

	var contractAddress *common.Address
	for _, l := range logs {
		contractAddress = &l.Address
		if l.Topics[0].Hex() == utils.TransferSingleSig {
			// handleTransferSingle

			// Decode TransferSingle log
			var event utils.Erc1155TransferSingleEvent
			err := utils.ERC1155ABI.UnpackIntoInterface(&event, "TransferSingle", l.Data)
			if err != nil {
				sugar.Errorw("Failed to unpack TransferSingle log:", "err", err)
			}

			// Decode the indexed fields for TransferSingle
			event.Operator = common.BytesToAddress(l.Topics[1].Bytes())
			event.From = common.BytesToAddress(l.Topics[2].Bytes())
			event.To = common.BytesToAddress(l.Topics[3].Bytes())

			amount := event.Value.String()
			_, err = AddOnchainTransaction(ctx, q, models.AddOnchainTransactionParams{
				From:      event.From.Hex(),
				To:        event.To.Hex(),
				AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
				TokenID:   event.Id.String(),
				Amount:    amount,
				TxHash:    l.TxHash.Hex(),
				Timestamp: time.Now(),
			})
			if err != nil {
				return err
			}

			// adding data to set
			tokenIdContractAddressSet.AddTokenIdContractAddress(event.Id, event.From.Hex())
			tokenIdContractAddressSet.AddTokenIdContractAddress(event.Id, event.To.Hex())
		}

		if l.Topics[0].Hex() == utils.TransferBatchSig {
			var event utils.Erc1155TransferBatchEvent
			err := utils.ERC1155ABI.UnpackIntoInterface(&event, "TransferBatch", l.Data)
			if err != nil {
				sugar.Errorw("Failed to unpack TransferBatch log:", "err", err)
			}

			// Decode the indexed fields for TransferBatch
			event.Operator = common.BytesToAddress(l.Topics[1].Bytes())
			event.From = common.BytesToAddress(l.Topics[2].Bytes())
			event.To = common.BytesToAddress(l.Topics[3].Bytes())

			for i := range event.Ids {
				amount := event.Values[i].String()
				_, err := AddOnchainTransaction(ctx, q, models.AddOnchainTransactionParams{
					From:      event.From.Hex(),
					To:        event.To.Hex(),
					AssetID:   contractType[chain.ID][l.Address.Hex()].ID,
					TokenID:   event.Ids[i].String(),
					Amount:    amount,
					TxHash:    l.TxHash.Hex(),
					Timestamp: time.Now(),
				})
				if err != nil {
					return err
				}

				// adding data to set
				tokenIdContractAddressSet.AddTokenIdContractAddress(event.Ids[i], event.From.Hex())
				tokenIdContractAddressSet.AddTokenIdContractAddress(event.Ids[i], event.To.Hex())
			}
		}
	}

	rpcClient, _ := utils.InitNewRPCClient(chain.RpcUrl)

	tokenIdList := tokenIdContractAddressSet.GetTokenIdContractAddressses()

	results := make([]string, len(tokenIdList)*2)
	calls := make([]rpc.BatchElem, len(tokenIdList)*2)

	for i, pairData := range tokenIdList {
		tokenId := pairData.TokenId
		ownerAddress := common.HexToAddress(pairData.ContractAddress)

		// Pack the data for the tokenURI function
		data, err := utils.ERC1155ABI.Pack("uri", tokenId)
		if err != nil {
			sugar.Errorf("Failed to pack data for tokenURI: %v", err)
			return err
		}

		encodedUriData := "0x" + common.Bytes2Hex(data)

		// Append the BatchElem for the eth_call
		calls[2*i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   contractAddress,
					"data": encodedUriData,
				},
				"latest",
			},
			Result: &results[2*i],
		}

		// 	// Pack the data for the ownerOf function
		ownerData, err := utils.ERC1155ABI.Pack("balanceOf", ownerAddress, tokenId)
		if err != nil {
			sugar.Errorf("Failed to pack data for balanceOf: %v", err)
			return err
		}

		encodedBalanceData := "0x" + common.Bytes2Hex(ownerData)

		// 	// Append the BatchElem for the eth_call
		calls[2*i+1] = rpc.BatchElem{
			Method: "eth_call",
			Args: []interface{}{
				map[string]interface{}{
					"to":   contractAddress,
					"data": encodedBalanceData,
				},
				"latest",
			},
			Result: &results[2*i+1],
		}
	}

	// // Execute batch call
	if err := rpcClient.BatchCallContext(ctx, calls); err != nil {
		log.Fatalf("Failed to execute batch call: %v", err)
	}

	// // Iterate over the results and update the balances
	for i := 0; i < len(results); i += 2 {
		var uri string
		var balance *big.Int
		utils.ERC1155ABI.UnpackIntoInterface(&uri, "uri", common.FromHex(results[i]))
		utils.ERC1155ABI.UnpackIntoInterface(&balance, "balanceOf", common.FromHex(results[i+1]))

		if err := Add1155Asset(ctx, q, models.Add1155AssetParams{
			AssetID: contractType[chain.ID][contractAddress.Hex()].ID,
			ChainID: chain.ID,
			TokenID: tokenIdList[i/2].TokenId.String(),
			Owner:   tokenIdList[i/2].ContractAddress,
			Attributes: sql.NullString{
				String: uri,
				Valid:  len(uri) > 0,
			},
		}); err != nil {
			return err
		}
	}

	tokenIdContractAddressSet.Reset()
	return nil
}
