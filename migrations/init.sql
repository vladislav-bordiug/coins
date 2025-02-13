CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    coins INTEGER NOT NULL DEFAULT 1000
);

CREATE TABLE IF NOT EXISTS coin_transactions (
    id SERIAL PRIMARY KEY,
    from_user_id INTEGER,
    to_user_id INTEGER,
    amount INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_from FOREIGN KEY (from_user_id) REFERENCES users(id),
    CONSTRAINT fk_to FOREIGN KEY (to_user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS user_inventory (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    item VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    UNIQUE(user_id, item),
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
);
