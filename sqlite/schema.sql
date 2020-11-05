CREATE TABLE stocks (
    time TIMESTAMP,
    symbol VARCHAR(32),
    price FLOAT,
    buy BOOLEAN
);

CREATE INDEX stocks_time ON stocks(time);
CREATE INDEX stocks_symbol ON stocks(symbol);
