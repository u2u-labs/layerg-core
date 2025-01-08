package models

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AssetType string

const (
	AssetTypeERC721  AssetType = "ERC721"
	AssetTypeERC1155 AssetType = "ERC1155"
	AssetTypeERC20   AssetType = "ERC20"
)

var ContractType = make(map[int32]map[string]Asset)

func (e *AssetType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = AssetType(s)
	case string:
		*e = AssetType(s)
	default:
		return fmt.Errorf("unsupported scan type for AssetType: %T", src)
	}
	return nil
}

type NullAssetType struct {
	AssetType AssetType `json:"assetType"`
	Valid     bool      `json:"valid"` // Valid is true if AssetType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullAssetType) Scan(value interface{}) error {
	if value == nil {
		ns.AssetType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.AssetType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullAssetType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.AssetType), nil
}

type CrawlerStatus string

const (
	CrawlerStatusCRAWLING CrawlerStatus = "CRAWLING"
	CrawlerStatusCRAWLED  CrawlerStatus = "CRAWLED"
)

func (e *CrawlerStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CrawlerStatus(s)
	case string:
		*e = CrawlerStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for CrawlerStatus: %T", src)
	}
	return nil
}

type NullCrawlerStatus struct {
	CrawlerStatus CrawlerStatus `json:"crawlerStatus"`
	Valid         bool          `json:"valid"` // Valid is true if CrawlerStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullCrawlerStatus) Scan(value interface{}) error {
	if value == nil {
		ns.CrawlerStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.CrawlerStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullCrawlerStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.CrawlerStatus), nil
}

type App struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	SecretKey string    `json:"secretKey"`
}

type Asset struct {
	ID                string        `json:"id"`
	ChainID           int32         `json:"chainId"`
	CollectionAddress string        `json:"collectionAddress"`
	Type              AssetType     `json:"type"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
	DecimalData       sql.NullInt16 `json:"decimalData"`
	InitialBlock      sql.NullInt64 `json:"initialBlock"`
	LastUpdated       sql.NullTime  `json:"lastUpdated"`
}

type BackfillCrawler struct {
	ChainID           int32         `json:"chainId"`
	CollectionAddress string        `json:"collectionAddress"`
	CurrentBlock      int64         `json:"currentBlock"`
	Status            CrawlerStatus `json:"status"`
	CreatedAt         time.Time     `json:"createdAt"`
}

type Chain struct {
	ID          int32  `json:"id"`
	Chain       string `json:"chain"`
	Name        string `json:"name"`
	RpcUrl      string `json:"rpcUrl"`
	ChainID     int64  `json:"chainId"`
	Explorer    string `json:"explorer"`
	LatestBlock int64  `json:"latestBlock"`
	BlockTime   int32  `json:"blockTime"`
}

type Erc1155CollectionAsset struct {
	ID         uuid.UUID      `json:"id"`
	ChainID    int32          `json:"chainId"`
	AssetID    string         `json:"assetId"`
	TokenID    string         `json:"tokenId"`
	Owner      string         `json:"owner"`
	Balance    string         `json:"balance"`
	Attributes sql.NullString `json:"attributes"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type Erc1155TotalSupply struct {
	AssetID     string         `json:"assetId"`
	TokenID     string         `json:"tokenId"`
	Attributes  sql.NullString `json:"attributes"`
	TotalSupply int64          `json:"totalSupply"`
}

type Erc20CollectionAsset struct {
	ID        uuid.UUID `json:"id"`
	ChainID   int32     `json:"chainId"`
	AssetID   string    `json:"assetId"`
	Owner     string    `json:"owner"`
	Balance   string    `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Erc721CollectionAsset struct {
	ID         uuid.UUID      `json:"id"`
	ChainID    int32          `json:"chainId"`
	AssetID    string         `json:"assetId"`
	TokenID    string         `json:"tokenId"`
	Owner      string         `json:"owner"`
	Attributes sql.NullString `json:"attributes"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type OnchainHistory struct {
	ID        uuid.UUID `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	AssetID   string    `json:"assetId"`
	TokenID   string    `json:"tokenId"`
	Amount    string    `json:"amount"`
	TxHash    string    `json:"txHash"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
