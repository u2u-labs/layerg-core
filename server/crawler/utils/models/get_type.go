package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
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
	ChainID      int32     `json:"chainId"`
	CollectionID string    `json:"collectionId"`
	Owner        string    `json:"owner"`
	Balance      string    `json:"balance"`
	UpdatedBy    uuid.UUID `json:"updatedBy"`
	Signature    string    `json:"signature"`
}

type Add721AssetParams struct {
	ChainID      int32          `json:"chainId"`
	CollectionID string         `json:"collectionId"`
	TokenID      string         `json:"tokenId"`
	Owner        string         `json:"owner"`
	Attributes   sql.NullString `json:"attributes"`
	UpdatedBy    uuid.UUID      `json:"updatedBy"`
	Signature    string         `json:"signature"`
}

type Add1155AssetParams struct {
	ChainID      int32          `json:"chainId"`
	CollectionID string         `json:"collectionId"`
	TokenID      string         `json:"tokenId"`
	Owner        string         `json:"owner"`
	Balance      string         `json:"balance"`
	Attributes   sql.NullString `json:"attributes"`
	UpdatedBy    uuid.UUID      `json:"updatedBy"`
	Signature    string         `json:"signature"`
}

type AddBackfillCrawlerParams struct {
	ChainID           int32  `json:"chainId"`
	CollectionAddress string `json:"collectionAddress"`
	CurrentBlock      int64  `json:"currentBlock"`
}

type AddNewAssetParams struct {
	ID                string        `json:"id"`
	ChainID           int32         `json:"chainId"`
	CollectionAddress string        `json:"collectionAddress"`
	Type              AssetType     `json:"type"`
	DecimalData       sql.NullInt16 `json:"decimalData"`
	InitialBlock      sql.NullInt64 `json:"initialBlock"`
	LastUpdated       sql.NullTime  `json:"lastUpdated"`
}
