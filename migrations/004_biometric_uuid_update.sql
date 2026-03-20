-- Migration: Update fingerprints tables to use TEXT user_id (UUID)
-- This allows proper foreign key relationship with the users table which uses UUID IDs

-- Since SQLite doesn't support ALTER COLUMN, we need to recreate the tables

-- Step 1: Rename old tables
ALTER TABLE fingerprints RENAME TO fingerprints_old;
ALTER TABLE fingerprint_verifications RENAME TO fingerprint_verifications_old;

-- Step 2: Create new fingerprints table with TEXT user_id
CREATE TABLE fingerprints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    fingerprint_data BLOB NOT NULL,
    finger_index TEXT NOT NULL CHECK(finger_index IN (
        'left_thumb', 'left_index', 'left_middle', 'left_ring', 'left_little',
        'right_thumb', 'right_index', 'right_middle', 'right_ring', 'right_little'
    )),
    quality INTEGER NOT NULL CHECK(quality >= 0 AND quality <= 100),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    UNIQUE(user_id, finger_index, is_active)
);

-- Step 3: Create new fingerprint_verifications table with TEXT user_id
CREATE TABLE fingerprint_verifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    fingerprint_id INTEGER NOT NULL,
    match_score INTEGER NOT NULL CHECK(match_score >= 0 AND match_score <= 100),
    is_success BOOLEAN NOT NULL,
    device_id TEXT NOT NULL,
    verified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (fingerprint_id) REFERENCES fingerprints(id) ON DELETE CASCADE
);

-- Step 4: Copy existing data (if any) - cast integer user_id to text
INSERT INTO fingerprints (id, user_id, fingerprint_data, finger_index, quality, created_at, updated_at, is_active)
SELECT id, CAST(user_id AS TEXT), fingerprint_data, finger_index, quality, created_at, updated_at, is_active
FROM fingerprints_old;

INSERT INTO fingerprint_verifications (id, user_id, fingerprint_id, match_score, is_success, device_id, verified_at)
SELECT id, CAST(user_id AS TEXT), fingerprint_id, match_score, is_success, device_id, verified_at
FROM fingerprint_verifications_old;

-- Step 5: Drop old tables
DROP TABLE fingerprints_old;
DROP TABLE fingerprint_verifications_old;

-- Step 6: Recreate indexes
CREATE INDEX IF NOT EXISTS idx_fingerprints_user_id ON fingerprints(user_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_is_active ON fingerprints(is_active);
CREATE INDEX IF NOT EXISTS idx_fingerprint_verifications_user_id ON fingerprint_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_fingerprint_verifications_verified_at ON fingerprint_verifications(verified_at);
