package models

import (
	"database/sql"
	"time"
)

type GetCrawlingBackfillCrawlerRow struct {
	ChainID           int32         `json:"chainId"`
	CollectionAddress string        `json:"collectionAddress"`
	CurrentBlock      int64         `json:"currentBlock"`
	Status            CrawlerStatus `json:"status"`
	CreatedAt         time.Time     `json:"createdAt"`
	Type              AssetType     `json:"type"`
	InitialBlock      sql.NullInt64 `json:"initialBlock"`
}

type UpdateCrawlingBackfillParams struct {
	ChainID           int32         `json:"chainId"`
	CollectionAddress string        `json:"collectionAddress"`
	Status            CrawlerStatus `json:"status"`
	CurrentBlock      int64         `json:"currentBlock"`
}

type AddOnchainTransactionParams struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	AssetID   string    `json:"assetId"`
	TokenID   string    `json:"tokenId"`
	Amount    string    `json:"amount"`
	TxHash    string    `json:"txHash"`
	Timestamp time.Time `json:"timestamp"`
}

type Add20AssetParams struct {
	AssetID string `json:"assetId"`
	ChainID int32  `json:"chainId"`
	Owner   string `json:"owner"`
	Balance string `json:"balance"`
}

type Add721AssetParams struct {
	AssetID    string         `json:"assetId"`
	ChainID    int32          `json:"chainId"`
	TokenID    string         `json:"tokenId"`
	Owner      string         `json:"owner"`
	Attributes sql.NullString `json:"attributes"`
}

type Add1155AssetParams struct {
	AssetID    string         `json:"assetId"`
	ChainID    int32          `json:"chainId"`
	TokenID    string         `json:"tokenId"`
	Owner      string         `json:"owner"`
	Balance    string         `json:"balance"`
	Attributes sql.NullString `json:"attributes"`
}
