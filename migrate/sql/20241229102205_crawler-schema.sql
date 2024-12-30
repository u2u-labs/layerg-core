-- +migrate Up

CREATE TABLE
    chains (
        id INT PRIMARY KEY,
        chain VARCHAR NOT NULL,
        name VARCHAR NOT NULL,
        rpc_url VARCHAR NOT NULL,
        chain_id BIGINT NOT NULL,
        explorer VARCHAR NOT NULL,
        latest_block BIGINT NOT NULL,
        block_time INT NOT NULL
    );

CREATE TYPE IF NOT EXISTS asset_type AS ENUM ('ERC721', 'ERC1155', 'ERC20');

CREATE TABLE
    assets (
        id VARCHAR PRIMARY KEY,
        chain_id INT NOT NULL,
        collection_address VARCHAR(42) NOT NULL,
        type asset_type NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        decimal_data SMALLINT,
        initial_block BIGINT,
        last_updated TIMESTAMP,
        FOREIGN KEY (chain_id) REFERENCES chains (id),
        CONSTRAINT UC_ASSET_COLLECTION UNIQUE (chain_id, collection_address)
    );

CREATE INDEX assets_chain_id_collection_address_idx ON assets (chain_id);

CREATE TABLE
    erc_721_collection_assets (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        chain_id INT NOT NULL,
        asset_id VARCHAR NOT NULL,
        token_id DECIMAL(78, 0) NOT NULL,
        owner VARCHAR(42) NOT NULL,
        attributes VARCHAR,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (asset_id) REFERENCES assets (id),
        CONSTRAINT UC_ERC721 UNIQUE (asset_id, chain_id, token_id)
    );

CREATE INDEX erc_721_collection_assets_chain_id_idx ON erc_721_collection_assets (asset_id, token_id);
CREATE INDEX erc_721_collection_assets_owner_idx ON erc_721_collection_assets (chain_id, owner);

CREATE TABLE
    erc_1155_collection_assets (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        chain_id INT NOT NULL,
        asset_id VARCHAR NOT NULL,
        token_id DECIMAL(78, 0) NOT NULL,
        owner VARCHAR(42) NOT NULL,
        balance DECIMAL(78, 0) NOT NULL,
        attributes VARCHAR,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (asset_id) REFERENCES assets (id)
    );

CREATE INDEX erc_1155_collection_assets_chain_id_idx ON erc_1155_collection_assets (asset_id, token_id);
CREATE INDEX erc_1155_collection_assets_owner_idx ON erc_1155_collection_assets (chain_id, owner);

CREATE TABLE
    erc_20_collection_assets (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        chain_id INT NOT NULL,
        asset_id VARCHAR NOT NULL,
        owner VARCHAR(42) NOT NULL UNIQUE,
        balance DECIMAL(78, 0) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (asset_id) REFERENCES assets (id)
    );

CREATE INDEX erc_20_collection_assets_asset_id_idx ON erc_20_collection_assets (asset_id, owner);
CREATE INDEX erc_20_collection_assets_owner_idx ON erc_20_collection_assets (chain_id, owner);

CREATE TABLE
    onchain_histories (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        "from" VARCHAR(42) NOT NULL,
        "to" VARCHAR(42) NOT NULL,
        asset_id VARCHAR NOT NULL,
        token_id DECIMAL(78, 0) NOT NULL,
        amount DECIMAL(60,18) NOT NULL,
        tx_hash VARCHAR(66) NOT NULL,
        timestamp TIMESTAMP NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX onchain_history_tx_hash_idx ON onchain_histories (tx_hash);

CREATE TABLE
    apps (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        name VARCHAR NOT NULL,
        secret_key VARCHAR NOT NULL
    );

INSERT INTO apps (id, name, secret_key)
VALUES ('f3e3bf76-62dc-42a7-ad0d-ef9033bc13a5', '', 'default');

INSERT INTO chains (id, chain, name, rpc_url, chain_id, explorer, latest_block, block_time)
VALUES (1, 'U2U', 'Nebulas Testnet', 'https://rpc-nebulas-testnet.uniultra.xyz', 2484, 'https://testnet.u2uscan.xyz/', 40984307, 500);

INSERT INTO assets (id, chain_id, collection_address, type, created_at, updated_at, decimal_data, initial_block, last_updated)
VALUES ('1:0xdFAe88F8610a038AFcDF47A5BC77C0963C65087c', 1, '0xdFAe88F8610a038AFcDF47A5BC77C0963C65087c', 'ERC20', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 18, 0, CURRENT_TIMESTAMP);

INSERT INTO assets (id, chain_id, collection_address, type, created_at, updated_at, decimal_data, initial_block, last_updated)
VALUES ('2:0xC5f15624b4256C1206e4BB93f2CCc9163A75b703', 1, '0xC5f15624b4256C1206e4BB93f2CCc9163A75b703', 'ERC20', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 18, 0, CURRENT_TIMESTAMP);

INSERT INTO assets (id, chain_id, collection_address, type, created_at, updated_at, decimal_data, initial_block, last_updated)
VALUES ('1:0x0091BD12166d29539Db6bb37FB79670779aBf266', 1, '0x0091BD12166d29539Db6bb37FB79670779aBf266', 'ERC721', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0, 0, CURRENT_TIMESTAMP);

INSERT INTO assets (id, chain_id, collection_address, type, created_at, updated_at, decimal_data, initial_block, last_updated)
VALUES ('1:0x9E87754dAB31dAD057DCDF233000F71fF55fA37f', 1, '0x9E87754dAB31dAD057DCDF233000F71fF55fA37f', 'ERC1155', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 0, 0, CURRENT_TIMESTAMP);

ALTER TABLE erc_1155_collection_assets 
ADD CONSTRAINT UC_ERC1155_OWNER UNIQUE (asset_id, chain_id, token_id, owner);

CREATE VIEW erc_1155_total_supply AS (
  SELECT 
    asset_id,
    token_id,
    attributes,
    SUM(balance) AS total_supply
  FROM 
    erc_1155_collection_assets
  GROUP BY 
    asset_id, token_id, attributes
);

-- +migrate Down

DROP VIEW IF EXISTS erc_1155_total_supply CASCADE;
DROP TABLE IF EXISTS erc_1155_collection_assets CASCADE;
DROP TABLE IF EXISTS erc_721_collection_assets CASCADE;
DROP TABLE IF EXISTS erc_20_collection_assets CASCADE;
DROP TABLE IF EXISTS onchain_histories CASCADE;
DROP TABLE IF EXISTS assets CASCADE;
DROP TABLE IF EXISTS chains CASCADE;
DROP TABLE IF EXISTS apps CASCADE;
