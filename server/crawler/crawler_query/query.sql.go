package crawlerQuery

import (
	"context"
	"database/sql"

	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
)

func GetCrawlingBackfillCrawler(ctx context.Context, db *sql.DB) ([]models.GetCrawlingBackfillCrawlerRow, error) {
	query := `
		SELECT 
		bc.chain_id, bc.collection_address, bc.current_block, bc.status, bc.created_at, 
		a.type, 
		a.initial_block
	FROM 
		backfill_crawlers AS bc
	JOIN 
		collections AS a 
		ON a.chain_id = bc.chain_id 
		AND a.collection_address = bc.collection_address 
	WHERE 
		bc.status = 'CRAWLING'
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backfillCrawlers []models.GetCrawlingBackfillCrawlerRow

	for rows.Next() {
		var backfillCrawler models.GetCrawlingBackfillCrawlerRow
		if err := rows.Scan(&backfillCrawler.ChainID, &backfillCrawler.CollectionAddress, &backfillCrawler.CurrentBlock, &backfillCrawler.Status, &backfillCrawler.CreatedAt, &backfillCrawler.Type, &backfillCrawler.InitialBlock); err != nil {
			return nil, err
		}
		backfillCrawlers = append(backfillCrawlers, backfillCrawler)
	}
	return backfillCrawlers, nil
}

func GetAllChain(ctx context.Context, db *sql.DB) ([]models.Chain, error) {
	query := "SELECT id, chain, name, rpc_url, chain_id, explorer, latest_block, block_time FROM chains"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []models.Chain

	for rows.Next() {
		var chain models.Chain
		err := rows.Scan(
			&chain.ID,
			&chain.Chain,
			&chain.Name,
			&chain.RpcUrl,
			&chain.ChainID,
			&chain.Explorer,
			&chain.LatestBlock,
			&chain.BlockTime,
		)
		if err != nil {
			return nil, err
		}
		chains = append(chains, chain)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chains, nil
}
func UpdateCrawlingBackfill(ctx context.Context, arg models.UpdateCrawlingBackfillParams, db *sql.DB) error {
	query := `UPDATE backfill_crawlers
		SET 
			status = COALESCE($3, status),            
			current_block = COALESCE($4, current_block)  
		WHERE chain_id = $1
		AND collection_address = $2
	`
	_, err := db.ExecContext(ctx, query, arg.ChainID, arg.CollectionAddress, arg.Status, arg.CurrentBlock)
	if err != nil {
		return err
	}
	return err
}

func Add20Asset(ctx context.Context, db *sql.DB, arg models.Add20AssetParams) error {
	add20Asset := `
		INSERT INTO
			erc_20_collection_assets (chain_id, collection_id, owner, balance, updated_by, signature)
		VALUES (
			$1, $2, $3, $4, $5, $6
		) ON CONFLICT (collection_id, owner) DO UPDATE SET
			balance = $4,
			updated_by = $5,
			signature = $6,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, chain_id, collection_id, owner, balance, created_at, updated_at, updated_by, signature
	`
	_, err := db.ExecContext(ctx, add20Asset,
		arg.ChainID,
		arg.CollectionID,
		arg.Owner,
		arg.Balance,
		arg.UpdatedBy,
		arg.Signature,
	)
	return err
}

func Add721Asset(ctx context.Context, db *sql.DB, arg models.Add721AssetParams) error {
	add721Asset := `
		INSERT INTO
			erc_721_collection_assets (chain_id, collection_id, token_id, owner, attributes, updated_by, signature)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) ON CONFLICT ON CONSTRAINT erc_721_collection_id_idx DO UPDATE SET
			owner = $4,
			attributes = $5,
			updated_by = $6,
			signature = $7,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, chain_id, collection_id, token_id, owner, attributes, created_at, updated_at, updated_by, signature
		`
	_, err := db.ExecContext(ctx, add721Asset,
		arg.ChainID,
		arg.CollectionID,
		arg.TokenID,
		arg.Owner,
		arg.Attributes,
		arg.UpdatedBy,
		arg.Signature,
	)
	return err
}

func Add1155Asset(ctx context.Context, db *sql.DB, arg models.Add1155AssetParams) error {
	add1155Asset := `
		INSERT INTO
			erc_1155_collection_assets (chain_id, collection_id, token_id, owner, balance, attributes, updated_by, signature)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) ON CONFLICT ON CONSTRAINT erc_1155_collection_id_idx DO UPDATE SET
			balance = $5,
			attributes = $6,
			updated_by = $7,
			signature = $8,
			updated_at = CURRENT_TIMESTAMP
			
		RETURNING id, chain_id, collection_id, token_id, owner, balance, attributes, created_at, updated_at, updated_by, signature
		`
	_, err := db.ExecContext(ctx, add1155Asset,
		arg.ChainID,
		arg.CollectionID,
		arg.TokenID,
		arg.Owner,
		arg.Balance,
		arg.Attributes,
		arg.UpdatedBy,
		arg.Signature,
	)
	return err
}

func AddOnchainTransaction(ctx context.Context, db *sql.DB, arg models.AddOnchainTransactionParams) (models.OnchainHistory, error) {
	const addOnchainTransaction = `
		INSERT INTO 
			onchain_histories("from","to",asset_id,token_id,amount,tx_hash,timestamp)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id, "from", "to", asset_id, token_id, amount, tx_hash, timestamp, created_at, updated_at
		`
	row := db.QueryRowContext(ctx, addOnchainTransaction,
		arg.From,
		arg.To,
		arg.AssetID,
		arg.TokenID,
		arg.Amount,
		arg.TxHash,
		arg.Timestamp,
	)
	var i models.OnchainHistory
	err := row.Scan(
		&i.ID,
		&i.From,
		&i.To,
		&i.AssetID,
		&i.TokenID,
		&i.Amount,
		&i.TxHash,
		&i.Timestamp,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

func GetChainById(ctx context.Context, db *sql.DB, id int32) (models.Chain, error) {
	getChainById := `
	SELECT id, chain, name, rpc_url, chain_id, explorer, latest_block, block_time FROM chains WHERE id = $1
	`
	row := db.QueryRowContext(ctx, getChainById, id)
	var i models.Chain
	err := row.Scan(
		&i.ID,
		&i.Chain,
		&i.Name,
		&i.RpcUrl,
		&i.ChainID,
		&i.Explorer,
		&i.LatestBlock,
		&i.BlockTime,
	)
	return i, err
}

func AddBackfillCrawler(ctx context.Context, db *sql.DB, arg models.AddBackfillCrawlerParams) error {
	addBackfillCrawler := `-- name: AddBackfillCrawler :exec
	INSERT INTO backfill_crawlers (
		chain_id, collection_address, current_block
	)
	VALUES (
		$1, $2, $3
	) ON CONFLICT ON CONSTRAINT BACKFILL_CRAWLERS_PKEY DO UPDATE SET
		current_block = EXCLUDED.current_block,
		status = 'CRAWLING'
	RETURNING chain_id, collection_address, current_block, status, created_at
	`
	_, err := db.ExecContext(ctx, addBackfillCrawler, arg.ChainID, arg.CollectionAddress, arg.CurrentBlock)
	return err
}

func AddNewAsset(ctx context.Context, db *sql.DB, arg models.AddNewAssetParams) (models.Asset, error) {
	const addNewAsset = `-- name: AddNewAsset :exec
		INSERT INTO collections (
			id, chain_id, collection_address, type, decimal_data, initial_block, last_updated
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id, chain_id, collection_address, type, created_at, updated_at, decimal_data, initial_block, last_updated
		`
	var i models.Asset
	row := db.QueryRowContext(ctx, addNewAsset,
		arg.ID,
		arg.ChainID,
		arg.CollectionAddress,
		arg.Type,
		arg.DecimalData,
		arg.InitialBlock,
		arg.LastUpdated,
	)
	err := row.Scan(
		&i.ID,
		&i.ChainID,
		&i.CollectionAddress,
		&i.Type,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DecimalData,
		&i.InitialBlock,
		&i.LastUpdated,
	)
	return i, err
}
