-- +migrate Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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

CREATE TYPE asset_type AS ENUM ('ERC721', 'ERC1155', 'ERC20');

CREATE TYPE collection_type AS ENUM ('ERC721', 'ERC1155', 'ERC20');

CREATE TABLE
    collections (
        id VARCHAR PRIMARY KEY,
        chain_id INT NOT NULL,
        collection_address VARCHAR(42) NOT NULL,
        type collection_type NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        decimal_data SMALLINT,
        initial_block BIGINT,
        last_updated TIMESTAMP,
        FOREIGN KEY (chain_id) REFERENCES chains (id)
    );

CREATE UNIQUE INDEX assets_collection_idx ON collections (chain_id, collection_address);

CREATE TABLE
    erc_20_collection_assets (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        chain_id INT NOT NULL,
        collection_id VARCHAR NOT NULL,
        owner VARCHAR(42) NOT NULL,
        balance DECIMAL(78, 0) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_by UUID,
        signature VARCHAR,
        FOREIGN KEY (collection_id) REFERENCES collections (id)
    );


CREATE UNIQUE INDEX erc_20_collection_id_idx ON erc_20_collection_assets (collection_id, owner);

CREATE INDEX erc_20_collection_assets_owner_idx ON erc_20_collection_assets (chain_id, owner);

CREATE TABLE
   erc_721_collection_assets (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        chain_id INT NOT NULL,
        collection_id VARCHAR NOT NULL,
        token_id DECIMAL(78, 0) NOT NULL,
        owner VARCHAR(42) NOT NULL,
        attributes VARCHAR,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_by UUID,
        signature VARCHAR,
        FOREIGN KEY (collection_id) REFERENCES collections (id)
    );

CREATE UNIQUE INDEX erc_721_collection_id_idx ON erc_721_collection_assets (collection_id, token_id);

CREATE INDEX erc_721_collection_assets_owner_idx ON erc_721_collection_assets (chain_id, owner);

CREATE TABLE
    erc_1155_collection_assets (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        chain_id INT NOT NULL,
        collection_id VARCHAR NOT NULL,
        token_id DECIMAL(78, 0) NOT NULL,
        owner VARCHAR(42) NOT NULL,
        balance DECIMAL(78, 0) NOT NULL,
        attributes VARCHAR,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_by UUID,
        signature VARCHAR,
        FOREIGN KEY (collection_id) REFERENCES collections (id)
    );

CREATE UNIQUE INDEX erc_1155_collection_id_idx ON erc_1155_collection_assets (collection_id, token_id, owner);

CREATE INDEX erc_1155_collection_assets_owner_idx ON erc_1155_collection_assets (chain_id, owner);

CREATE TABLE
    onchain_histories (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
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
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        name VARCHAR NOT NULL,
        secret_key VARCHAR NOT NULL
    );

CREATE TYPE crawler_status AS ENUM ('CRAWLING', 'CRAWLED');

CREATE TABLE backfill_crawlers (
    chain_id INT NOT NULL,
    collection_address VARCHAR NOT NULL,
    current_block BIGINT NOT NULL,
    status crawler_status DEFAULT 'CRAWLING' NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chain_id, collection_address)
);

INSERT INTO apps (id, name, secret_key)
VALUES ('f3e3bf76-62dc-42a7-ad0d-ef9033bc13a5', '', 'default');

INSERT INTO chains (id, chain, name, rpc_url, chain_id, explorer, latest_block, block_time)
VALUES (1, 'U2U', 'Nebulas Testnet', 'https://rpc-nebulas-testnet.uniultra.xyz', 2484, 'https://testnet.u2uscan.xyz/', 40984307, 500);


-- +migrate Down

DROP VIEW IF EXISTS erc_1155_total_supply CASCADE;
DROP TABLE IF EXISTS erc_1155_collection_assets CASCADE;
DROP TABLE IF EXISTS erc_721_collection_assets CASCADE;
DROP TABLE IF EXISTS erc_20_collection_assets CASCADE;
DROP TABLE IF EXISTS onchain_histories CASCADE;
DROP TABLE IF EXISTS backfill_crawlers;
DROP TABLE IF EXISTS assets CASCADE;
DROP TABLE IF EXISTS chains CASCADE;
DROP TABLE IF EXISTS apps CASCADE;
