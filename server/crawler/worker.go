package crawler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/u2u-labs/layerg-core/server"
	crawlerQuery "github.com/u2u-labs/layerg-core/server/crawler/crawler_query"
	"github.com/u2u-labs/layerg-core/server/crawler/utils"
	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
	u2u "github.com/unicornultrafoundation/go-u2u"
	"github.com/unicornultrafoundation/go-u2u/accounts/abi"
	"github.com/unicornultrafoundation/go-u2u/common"
	"github.com/unicornultrafoundation/go-u2u/ethclient"
	"go.uber.org/zap"
)

func StartWorker(db *sql.DB, rdb *redis.Client, queueClient *asynq.Client, config server.Config) {
	// defer queueClient.Close()

	var (
		ctx    = context.Background()
		logger = &zap.Logger{}
	)
	logger, _ = zap.NewProduction()
	// if viper.GetString("LOG_LEVEL") == "PROD" {
	// 	logger, _ = zap.NewProduction()
	// } else {
	// 	logger, _ = zap.NewDevelopment()
	// }
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	// conn, err := sql.Open(
	// 	viper.GetString("COCKROACH_DB_DRIVER"),
	// 	viper.GetString("COCKROACH_DB_URL"),
	// )
	// if err != nil {
	// 	log.Fatalf("Could not connect to database: %v", err)
	// }

	// sqlDb := dbCon.New(conn)

	// if err != nil {
	// 	panic(err)
	// }
	// redisConfig := config.GetRedisDbConfig()
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     redisConfig.Url,
	// 	Password: redisConfig.Password,
	// 	DB:       0,
	// })

	// queueClient := asynq.NewClient(asynq.RedisClientOpt{
	// 	Addr:     redisConfig.Url,
	// 	Password: redisConfig.Password,
	// 	DB:       0,
	// })
	var err error
	if utils.ERC20ABI, err = abi.JSON(strings.NewReader(utils.ERC20ABIStr)); err != nil {
		panic(err)
	}
	if utils.ERC721ABI, err = abi.JSON(strings.NewReader(utils.ERC721ABIStr)); err != nil {
		panic(err)
	}
	if utils.ERC1155ABI, err = abi.JSON(strings.NewReader(utils.ERC1155ABIStr)); err != nil {
		panic(err)
	}

	InitBackfillProcessor(ctx, sugar, db, rdb, queueClient, config)
}

func InitBackfillProcessor(ctx context.Context, sugar *zap.SugaredLogger, db *sql.DB, rdb *redis.Client, queueClient *asynq.Client, config server.Config) error {
	// Get all chains
	chains, err := crawlerQuery.GetAllChain(ctx, db)
	if err != nil {
		return err
	}
	for _, chain := range chains {
		client, err := initChainClient(&chain)
		if err != nil {
			return err
		}

		redisConfig := config.GetRedisDbConfig()

		// handle queue
		srv := asynq.NewServer(
			asynq.RedisClientOpt{Addr: redisConfig.Url},
			asynq.Config{
				Concurrency: 5,
			},
		)

		// mux maps a type to a handler
		mux := asynq.NewServeMux()
		taskName := BackfillCollection + ":" + strconv.Itoa(int(chain.ID))
		mux.Handle(taskName, NewBackfillProcessor(sugar, client, db, &chain, rdb))
		sugar.Infof("Starting server for chain %s with task name %s", chain.Name, taskName)

		// go func(server *asynq.Server, mux *asynq.ServeMux, chainName string) {
		// 	if err := server.Run(mux); err != nil {
		// 		sugar.Errorf("Server for chain %s failed: %v", chainName, err)
		// 	}
		// }(srv, mux, chain.Name)

		// if err := srv.Run(mux); err != nil {
		// 	log.Fatalf("could not run server: %v", err)
		// }
		go func() {
			if err := srv.Run(mux); err != nil {
				sugar.Fatalf("Failed to run asynq server: %v", err)
			}
		}()
	}

	return nil
}

// ----------------------------------------------
// Task
// ----------------------------------------------
const (
	BackfillCollection = "backfill_collection"
)

//----------------------------------------------
// Write a function NewXXXTask to create a task.
// A task consists of a type and a payload.
//----------------------------------------------

func NewBackfillCollectionTask(bf *models.GetCrawlingBackfillCrawlerRow) (*asynq.Task, error) {

	payload, err := json.Marshal(bf)
	if err != nil {
		return nil, err
	}

	taskName := BackfillCollection + ":" + strconv.Itoa(int(bf.ChainID))
	fmt.Print("Add new backfill task")
	return asynq.NewTask(taskName, payload), nil
}

func (processor *BackfillProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var bf models.GetCrawlingBackfillCrawlerRow

	if err := json.Unmarshal(t.Payload(), &bf); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	// Generate Redis key for idempotency
	idempotencyKey := fmt.Sprintf("backfill:%d:%s:%d:%d", bf.ChainID, bf.CollectionAddress, bf.CurrentBlock, bf.CurrentBlock+100)

	// Check if the key already exists in Redis
	exists, err := processor.rdb.Exists(ctx, idempotencyKey).Result()
	if err != nil {
		processor.sugar.Errorf("Failed to check idempotency key: %v", err)
		return err
	}

	// Skip if key already exists (duplicate task detected)
	if exists > 0 {
		processor.sugar.Infof("Skipping already processed block range for key: %s", idempotencyKey)
		return nil
	}

	// Set the idempotency key in Redis
	err = processor.rdb.Set(ctx, idempotencyKey, "processing", 0).Err() // `0` means no expiry
	if err != nil {
		processor.sugar.Errorf("Failed to set idempotency key: %v", err)
		return err
	}

	blockRangeScan := int64(100)

	toScanBlock := bf.CurrentBlock + 100
	// get the nearest upper block that multiple of blockRangeScan
	if (bf.CurrentBlock % blockRangeScan) != 0 {
		toScanBlock = ((bf.CurrentBlock / blockRangeScan) + 1) * blockRangeScan
	}
	processor.sugar.Infof("good", toScanBlock, bf)
	if bf.InitialBlock.Valid && toScanBlock >= bf.InitialBlock.Int64 {
		toScanBlock = bf.InitialBlock.Int64
		bf.Status = models.CrawlerStatusCRAWLED
	}

	var transferEventSig []string

	if bf.Type == models.AssetTypeERC20 || bf.Type == models.AssetTypeERC721 {
		transferEventSig = []string{utils.TransferEventSig}
	} else if bf.Type == models.AssetTypeERC1155 {
		transferEventSig = []string{utils.TransferSingleSig, utils.TransferBatchSig}
	}

	// Initialize topics slice
	var topics [][]common.Hash

	// Populate the topics slice
	innerSlice := make([]common.Hash, len(transferEventSig))
	for i, sig := range transferEventSig {
		innerSlice[i] = common.HexToHash(sig) // Convert each signature to common.Hash
	}
	topics = append(topics, innerSlice) // Add the inner slice to topics

	logs, _ := processor.ethClient.FilterLogs(ctx, u2u.FilterQuery{
		Topics:    topics,
		BlockHash: nil,
		FromBlock: big.NewInt(bf.CurrentBlock),
		ToBlock:   big.NewInt(toScanBlock),
		Addresses: []common.Address{common.HexToAddress(bf.CollectionAddress)},
	})
	if bf.CurrentBlock%100 == 0 {
		processor.sugar.Infof("Batch Call from block %d to block %d for assetType %s, contractAddress %s", bf.CurrentBlock, toScanBlock, bf.Type, bf.CollectionAddress)
	}

	switch bf.Type {
	case models.AssetTypeERC20:
		handleErc20BackFill(ctx, processor.sugar, processor.db, processor.ethClient, processor.chain, logs)
	case models.AssetTypeERC721:
		handleErc721BackFill(ctx, processor.sugar, processor.db, processor.ethClient, processor.chain, logs)
	case models.AssetTypeERC1155:
		handleErc1155Backfill(ctx, processor.sugar, processor.db, processor.ethClient, processor.chain, logs)
	}

	bf.CurrentBlock = toScanBlock

	crawlerQuery.UpdateCrawlingBackfill(ctx, models.UpdateCrawlingBackfillParams{
		ChainID:           bf.ChainID,
		CollectionAddress: bf.CollectionAddress,
		Status:            bf.Status,
		CurrentBlock:      bf.CurrentBlock,
	}, processor.db)

	if bf.Status == models.CrawlerStatusCRAWLED {
		return nil
	}
	return nil
}

// BackfillProcessor implements asynq.Handler interface.
type BackfillProcessor struct {
	sugar     *zap.SugaredLogger
	ethClient *ethclient.Client
	db        *sql.DB
	chain     *models.Chain
	rdb       *redis.Client // Add Redis client here
}

func NewBackfillProcessor(sugar *zap.SugaredLogger, ethClient *ethclient.Client, db *sql.DB, chain *models.Chain, rdb *redis.Client) *BackfillProcessor {
	sugar.Infow("Initiated new chain backfill, start crawling", "chain", chain.Chain)
	return &BackfillProcessor{sugar, ethClient, db, chain, rdb}
}
