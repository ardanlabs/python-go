# Using SQLite For Fun & Profit

[SQLite](https://www.sqlite.org/index.html) is an embedded SQL database. SQLite is small, fast and the [most used](https://www.sqlite.org/mostdeployed.html) database. It's also by far my favorite way to exchange large amounts of data.

I prefer to use relational (SQL) databases in general since they provide several features that are very helpful when working with data:

[Transactions](https://en.wikipedia.org/wiki/Database_transaction)
: You insert data into an SQL database inside a transaction. This means that either all of the data gets in, or non of it. Transactions simplify retry logic in data pipelines by order of magnitude.

[Schema](https://en.wikipedia.org/wiki/Database_schema)
: Data in relational database has a schema, which means it's easier to check the validity of your data.

[SQL](https://en.wikipedia.org/wiki/SQL)
: SQL (Structured Query Language) is a language for selecting and changing data. You don't need to invent yet another way to select interesting parts of data. SQL is an established format and there's a lot of knowledge and tooling around it.

I like SQLite since the database is a single file, which makes it easier to share data. Even though it's a single file, SQLite can handle up to 281 terabytes of data. SQLite also comes with a command line client called `sqlite3` which is great for quick prototyping.


## The Project

We'll write an HTTP server in Go what will get notifications on trades and will store them in an SQLite database. Then we'll write a Python program that will process this data.

In Go, we'll be using [github.com/mattn/go-sqlite3](github.com/mattn/go-sqlite3) which is a wrapper around the SQLite C library.

_Note: Since `go-sqlite` uses `cgo`, the initial build time will be longer than usual. Using `cgo` means that the resulting executable depends on some shared libraries, making distribution slightly more complicated._

In Python, we'll use the build-in `sqlite3` modules and Pandas `read_sql` function to load the data.

## The Go HTTP Server

**Listing 1: Trade struct**
```
38 // Trade is a buy/sell trade for symbol
39 type Trade struct {
40 	Time   time.Time
41 	Symbol string
42 	Price  float64
43 	IsBuy  bool `json:"buy"`
44 }
```

Listing one shows the `Trade` data structure. It has a `Time` field for the trade time, a `Symbol` field for the stock symbol (e.g. `AAPL`) the `Price` and a boolean flag that tells if it's a buy or a sell trade.
On line 43 we use a field tag to tell the JSON decoder to fill the `IsBuy` field from the `buy` field in the incoming JSON object.

**Listing 2: Database Schema**
```
25 	schemaSQL = `
26 CREATE TABLE IF NOT EXISTS trades (
27     time TIMESTAMP,
28     symbol VARCHAR(32),
29     price FLOAT,
30     buy BOOLEAN
31 );
32 
33 CREATE INDEX IF NOT EXISTS trades_time ON trades(time);
34 CREATE INDEX IF NOT EXISTS trades_symbol ON trades(symbol);
`
```
Listing 2 describes the database schema. On line 26 we create a table called `trades`. On lines 27-30 we define the table columns that correspond to the `Trade` struct fields. On lines 33-34 we create indices on the table to allow fast selection of rows by `time` and `symbol`.

Inserting records one-by-one is a slow process. We're going to store records in a buffer and once it's full insert all the records in the buffer to the database. This has the advantage of being fast, on my machine about 60,000 records/sec, but carries the risk that we'll loose data on server crash.

**Listing 3: Insert Record SQL***
```
17 	insertSQL = `
18 INSERT INTO trades (
19 	time, symbol, price, buy
20 ) VALUES (
21 	?, ?, ?, ?
22 )
23 `
```

Listing 3 defines the SQL to insert a record to the database. On line 21 we use `?` as place holders for the parameters to this query. *Never* use `fmt.Sprintf` to craft an SQL query - you're risking an [SQL injection](https://xkcd.com/327/).

**Listing 4: TradesDB**
```
46 // TradesDB is a database of stocks
47 type TradesDB struct {
48 	db     *sql.DB
49 	stmt   *sql.Stmt
50 	buffer []Trade
51 }
```

Listing 4 describes the `TradesDB` struct. On line 48 we hold the connection to the database. On line 49 we store a prepared (pre-compiled) statement for inserting and on line 50 we have the in-memory buffer of pending transactions.

**Listing 5: NewTradesDB**
```
53 // NewTradesDB connect to SQLite database in dbFile
54 // Tables will be created if they don't exist
55 func NewTradesDB(dbFile string) (*TradesDB, error) {
56 	db, err := sql.Open("sqlite3", dbFile)
57 	if err != nil {
58 		return nil, err
59 	}
60 
61 	_, err = db.Exec(schemaSQL)
62 	if err != nil {
63 		return nil, err
64 	}
65 
66 	stmt, err := db.Prepare(insertSQL)
67 	if err != nil {
68 		return nil, err
69 	}
70 
71 	buffer := make([]Trade, 0, 1024)
72 	return &TradesDB{db, stmt, buffer}, nil
73 }
```

Listing 5 show the creation of a `TradesDB`. On line 56 we connect to the database. On line 61 we execute the schema SQL to create the `trades` table if it doesn't already exist. On line 66 we pre-compile the insert SQL statement. On line 71 we create the internal buffer with 0 length and a capacity of 1024.

**Listing 6: AddTrade**
```
104 // AddTrade adds a new trade.
105 // The new trade is only added to the internal buffer and will be inserted
106 // to the database later on Flush
107 func (db *TradesDB) AddTrade(t Trade) error {
108 	// FIXME: We might grow buffer indefinitely on persistent Flush errors
109 	db.buffer = append(db.buffer, t)
110 	if len(db.buffer) == cap(db.buffer) {
111 		if err := db.Flush(); err != nil {
112 			return err
113 		}
114 	}
115 	return nil
116 }
```

Listing 6 show the `AddTrade` method. On line 109 we append the trade to the in-memory buffer. On line 110 we check to see if the buffer is full and if it is we call `Flush` on line 111 that will insert the records from the buffer into the database.

**Listing 7: Flush**
```
83 // Flush inserts trades from internal buffer to the database
84 func (db *TradesDB) Flush() error {
85 	tx, err := db.db.Begin()
86 	if err != nil {
87 		return err
88 	}
89 
90 	for _, t := range db.buffer {
91 		_, err := tx.Stmt(db.stmt).Exec(t.Time, t.Symbol, t.Price, t.IsBuy)
92 		if err != nil {
93 			tx.Rollback()
94 			return err
95 		}
96 	}
97 	err = tx.Commit()
98 	if err == nil {
99 		db.buffer = db.buffer[:0]
100 	}
101 	return err
102 }
```

Listing 7 shows the `Flush` method. On line 85 we start a transaction. On line 90 we iterate over the internal buffer and on line 91 insert each trade. On line 93 we issue a [rollback](https://en.wikipedia.org/wiki/Rollback_(data_management)). On line 97 we issue a [commit](https://en.wikipedia.org/wiki/Commit_(data_management)). On line 99, if there are no errors, we reset the in-memory trades buffer.

**Listing 8: Close**
```
75 // Close closes all database related resources
76 func (db *TradesDB) Close() error {
77 	// TODO: Check errors from Flush & stmt.Close
78 	db.Flush()
79 	db.stmt.Close()
80 	return db.db.Close()
81 }
```

Listing 8 shows the `Close` methods. On line 78 we call `Flush` to insert any remaining trades into the database. On line 79 and 80 we close the statement and the database.
