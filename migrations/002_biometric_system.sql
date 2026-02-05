-- Migration for fingerprint biometric system

-- Table for storing fingerprint templates
CREATE TABLE IF NOT EXISTS fingerprints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    fingerprint_data BLOB NOT NULL,
    finger_index TEXT NOT NULL CHECK(finger_index IN (
        'left_thumb', 'left_index', 'left_middle', 'left_ring', 'left_little',
        'right_thumb', 'right_index', 'right_middle', 'right_ring', 'right_little'
    )),
    quality INTEGER NOT NULL CHECK(quality >= 0 AND quality <= 100),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, finger_index, is_active)
);

-- Table for logging fingerprint verification attempts
CREATE TABLE IF NOT EXISTS fingerprint_verifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    fingerprint_id INTEGER NOT NULL,
    match_score INTEGER NOT NULL CHECK(match_score >= 0 AND match_score <= 100),
    is_success BOOLEAN NOT NULL,
    device_id TEXT NOT NULL,
    verified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (fingerprint_id) REFERENCES fingerprints(id) ON DELETE CASCADE
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_fingerprints_user_id ON fingerprints(user_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_is_active ON fingerprints(is_active);
CREATE INDEX IF NOT EXISTS idx_fingerprint_verifications_user_id ON fingerprint_verifications(user_id);
CREATE INDEX IF NOT EXISTS idx_fingerprint_verifications_verified_at ON fingerprint_verifications(verified_at);
