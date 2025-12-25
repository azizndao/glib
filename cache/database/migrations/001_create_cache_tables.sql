-- Migration for cache_entries table

-- +migrate Up
CREATE TABLE IF NOT EXISTS cache_entries (
    key VARCHAR(255) NOT NULL PRIMARY KEY,
    value BLOB NOT NULL,
    expiration DATETIME NULL,
    created_at DATETIME NOT NULL,
    INDEX idx_cache_expiration (expiration)
);

CREATE TABLE IF NOT EXISTS cache_locks (
    name VARCHAR(255) NOT NULL PRIMARY KEY,
    owner VARCHAR(255) NOT NULL,
    expiration DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    INDEX idx_lock_owner (owner),
    INDEX idx_lock_expiration (expiration)
);

-- +migrate Down
DROP TABLE IF EXISTS cache_locks;
DROP TABLE IF EXISTS cache_entries;
