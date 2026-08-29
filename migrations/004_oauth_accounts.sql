PRAGMA foreign_keys = OFF;

CREATE TABLE users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    created_at TEXT NOT NULL
);

INSERT INTO users_new (
    id,
    email,
    username,
    password_hash,
    created_at
)
SELECT
    id,
    email,
    username,
    password_hash,
    created_at
FROM users;

DROP TABLE users;

ALTER TABLE users_new RENAME TO users;

CREATE TABLE oauth_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,

    provider TEXT NOT NULL
        CHECK (length(trim(provider)) > 0),

    provider_user_id TEXT NOT NULL
        CHECK (length(trim(provider_user_id)) > 0),

    email TEXT NOT NULL
        CHECK (length(trim(email)) > 0),

    created_at TEXT NOT NULL,

    UNIQUE(provider, provider_user_id),
    UNIQUE(user_id, provider),

    FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

PRAGMA foreign_keys = ON;