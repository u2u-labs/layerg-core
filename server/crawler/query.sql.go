package crawler

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
		assets AS a 
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
		if err := rows.Scan(&backfillCrawler.ChainID, &backfillCrawler.CollectionAddress, &backfillCrawler.CurrentBlock, &backfillCrawler.Status); err != nil {
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
			erc_20_collection_assets (asset_id, chain_id, owner, balance)
		VALUES (
			$1, $2, $3, $4
		) ON CONFLICT (owner) DO UPDATE SET
			balance = $4
		RETURNING id, chain_id, asset_id, owner, balance, created_at, updated_at
	`
	_, err := db.ExecContext(ctx, add20Asset,
		arg.AssetID,
		arg.ChainID,
		arg.Owner,
		arg.Balance,
	)
	return err
}

func Add721Asset(ctx context.Context, db *sql.DB, arg models.Add721AssetParams) error {
	add721Asset := `
		INSERT INTO
			erc_721_collection_assets (asset_id, chain_id, token_id, owner, attributes)
		VALUES (
			$1, $2, $3, $4, $5
		) ON CONFLICT ON CONSTRAINT UC_ERC721 DO UPDATE SET
			owner = $4,
			attributes = $5
		RETURNING id, chain_id, asset_id, token_id, owner, attributes, created_at, updated_at
		`
	_, err := db.ExecContext(ctx, add721Asset,
		arg.AssetID,
		arg.ChainID,
		arg.TokenID,
		arg.Owner,
		arg.Attributes,
	)
	return err
}

func Add1155Asset(ctx context.Context, db *sql.DB, arg models.Add1155AssetParams) error {
	add1155Asset := `
		INSERT INTO
			erc_1155_collection_assets (asset_id, chain_id, token_id, owner, balance, attributes)
		VALUES (
			$1, $2, $3, $4, $5, $6
		) ON CONFLICT ON CONSTRAINT UC_ERC1155_OWNER DO UPDATE SET
			balance = $5,
			attributes = $6
			
		RETURNING id, chain_id, asset_id, token_id, owner, balance, attributes, created_at, updated_at
		`
	_, err := db.ExecContext(ctx, add1155Asset,
		arg.AssetID,
		arg.ChainID,
		arg.TokenID,
		arg.Owner,
		arg.Balance,
		arg.Attributes,
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
