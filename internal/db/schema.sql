CREATE TABLE postings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id TEXT,
    date DATETIME,
    payee TEXT,
    account TEXT,
    commodity TEXT,
    quantity TEXT,
    amount TEXT,
    original_amount TEXT,
    status TEXT,
    tag_recurring TEXT,
    tag_period TEXT,
    transaction_begin_line INTEGER,
    transaction_end_line INTEGER,
    file_name TEXT,
    forecast BOOLEAN,
    note TEXT,
    transaction_note TEXT,
    transaction_hash TEXT
);

CREATE INDEX idx_postings_txn_hash ON postings(transaction_id, transaction_hash);
CREATE INDEX idx_postings_forecast_date ON postings(forecast, date);
CREATE INDEX idx_postings_forecast_account_date ON postings(forecast, account, date);

CREATE TABLE prices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATETIME,
    commodity_type TEXT,
    commodity_id TEXT,
    commodity_name TEXT,
    quote_commodity TEXT,
    value TEXT,
    source TEXT
);

CREATE INDEX idx_prices_commodity_name ON prices(commodity_name);
CREATE INDEX idx_prices_quote_commodity ON prices(quote_commodity);
CREATE UNIQUE INDEX idx_prices_type_date_base_quote ON prices(commodity_type, date, commodity_name, quote_commodity);

CREATE TABLE portfolios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    commodity_type TEXT,
    parent_commodity_id TEXT,
    security_id TEXT,
    security_name TEXT,
    security_type TEXT,
    security_rating TEXT,
    security_industry TEXT,
    percentage TEXT
);
