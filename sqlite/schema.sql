CREATE TABLE IF NOT EXISTS stocks (
    time TIMESTAMP,
    symbol VARCHAR(32),
    price FLOAT,
    buy BOOLEAN
);

CREATE INDEX IF NOT EXISTS stocks_time ON stocks(time);
CREATE INDEX IF NOT EXISTS stocks_symbol ON stocks(symbol);
