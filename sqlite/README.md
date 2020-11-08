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

## The Go HTTP Server - [trades.go](https://github.com/ardanlabs/python-go/blob/master/sqlite/trades.go)

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

Listing 5 show the creation of a `TradesDB`. On line 56 we connect to the database using the "sqlite3" driver. On line 61 we execute the schema SQL to create the `trades` table if it doesn't already exist. On line 66 we pre-compile the insert SQL statement. On line 71 we create the internal buffer with 0 length and a capacity of 1024.

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

Listing 8 shows the `Close` methods. On line 78 we call `Flush` to insert any remaining trades into the database. On line 79 and 80 we close the statement and the database. Functions creating a `TradesDB` should have a `defer db.Close()` to make sure the database connection is freed. In our case the database is global and the connection is alive for the life of the program - so we don't call `Close`.

**Listing 9: HTTP Handler**
```
118 // tradeHandler handles a new trade notification
119 func tradeHandler(w http.ResponseWriter, r *http.Request) {
120 	if r.Method != "POST" {
121 		http.Error(w, "only POST", http.StatusMethodNotAllowed)
122 		return
123 	}
124 
125 	if db == nil {
126 		log.Printf("DB uninitialized")
127 		http.Error(w, "Database not initialized", http.StatusInternalServerError)
128 		return
129 	}
130 
131 	defer r.Body.Close()
132 
133 	var tr Trade
134 	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
135 		log.Printf("json decode error: %s", err)
136 		http.Error(w, err.Error(), http.StatusBadRequest)
137 		return
138 	}
139 
140 	if err := db.AddTrade(tr); err != nil {
141 		log.Printf("add error: %s", err)
142 		http.Error(w, err.Error(), http.StatusInternalServerError)
143 		return
144 	}
145 }
```

Listing 9 shows the HTTP handler for new trades. On line 134 we decode the JSON input into a `Trade` struct. On line 140 we add the trade struct to the global `TradesDB` called `db`

**Listing 10: main**
```
147 func main() {
148 	dbFile := os.Getenv("DB_FILE")
149 	if dbFile == "" {
150 		dbFile = "trades.db"
151 	}
152 
153 	var err error
154 	db, err = NewTradesDB(dbFile)
155 	if err != nil {
156 		log.Fatal(err)
157 	}
158 	log.Printf("conneted to %s", dbFile)
159 
160 	http.HandleFunc("/trade", tradeHandler)
161 
162 	addr := os.Getenv("HTTPD_ADDR")
163 	if addr == "" {
164 		addr = ":8080"
165 	}
166 
167 	log.Printf("server listening on %s", addr)
168 	if err := http.ListenAndServe(addr, nil); err != nil {
169 		log.Fatal(err)
170 	}
171 }
```

Listing 10 shows how we run the server. On line 148 we use the `DB_FILE` environment variable to know the location of the database file. If it doesn't exist, SQLite will create it. On 154 we create the global database `db`. On lines 160 to 170 we set the HTTP server routing and start the HTTP server.

**Listing 11: imports**
```
03 import (
04 	"database/sql"
05 	"encoding/json"
06 	"log"
07 	"net/http"
08 	"os"
09 	"time"
10 
11 	_ "github.com/mattn/go-sqlite3"
12 )
```

Listing 11 shows the imports for the file. On line 04 we import `database/sql` that defines the API for working with SQL databases. `database/sql` does not contain any specific database driver. On line 11 we import the `github.com/mattn/go-sqlite3` package. Since we import `github.com/mattn/go-sqlite3` only for the side effect of registering the "sqlite3" protocol. Since unused imports are a compilation error, we use `_` in front of the import - telling the Go compiler it's OK we don't use this package in the code.


## The Python Code - [analyze_trades.py](https://github.com/ardanlabs/python-go/blob/master/sqlite/analyze_trades.py)

**Listing 12: imports**
```
02 import sqlite3
03 from contextlib import closing
04 from datetime import datetime
05 
06 import pandas as pd
```

Listing 12 shows the libraries we're using in the Python code. On line 02 we import the built-in `sqlite3` module and on line 06 we import the pandas library.

**Listing 13: Select SQL**
```
08 select_sql = """
09 SELECT * FROM trades
10 WHERE time >= ? AND time <= ?
11 """
```

Listing 13 show the SQL for selecting data. On line 10 we select all the columns from the `trades` table. On line 10 we add a `WHERE` clause for selecting in time range. As in the Go code, we use `?` as placeholders for arguments and *not* consturct the SQL manually.

**Listing 14: Loading Trades**
```
14 def load_trades(db_file, start_time, end_time):
15     """Load trades from db_file in given time range."""
16     conn = sqlite3.connect(db_file)
17     with closing(conn) as db:
18         df = pd.read_sql(select_sql, db, params=(start_time, end_time))
19 
20     # We can't use detect_types=sqlite3.PARSE_DECLTYPES here since Go is
21     # inserting time zone and Python's sqlite3 doesn't handle it.
22     # See https://bugs.python.org/issue29099
23     df["time"] = pd.to_datetime(df["time"])
24     return df
```

Listing 14 shows the code for loading trades at a given time range. On line 16 we connect to the database. On lines 17 we use a [context manager](https://www.python.org/dev/peps/pep-0343/), somewhat like Go's `defer` to make sure the database is closed. On line 18 we use pandas `read_sql` function to load data from an SQL query to a `DataFrame`. Python has [an API](https://www.python.org/dev/peps/pep-0249/) for connection to databases (line `database/sql`) and Pandas can use any compatible driver. On line 23 we convert the `time` column to pandas `Timestamp`. This is specific to SQLite that doesn't have built-in support for `TIMESTAMP` types.


**Listing 15: Average Price**
```
27 def average_price(df):
28     """Return average price in df grouped by (stock, buy)"""
29     return df.groupby(["symbol", "buy"], as_index=False)["price"].mean()
```

Listing 15 shows how to calculate the average price per `symbol` and `buy`. On line 29 we use the DataFrame `groupby` to group by `symbol` and `buy`. We use `as_index=False` to get the `symbol` and `buy` as columns in the resulting data frame. Then we take the `price` column and calculate the mean per group.

**Listing 16: Output**
```
symbol,buy,price
AAPL,0,250.82925665004535
AAPL,1,248.28277375538832
GOOG,0,250.11537993385295
GOOG,1,252.4726772487683
MSFT,0,250.9214212695317
MSFT,1,248.60187022941685
NVDA,0,250.3844763417279
NVDA,1,249.3578146208962
```

Listing 16 shows the output of running the Python code on dummy data.

## Conclusion

I highly recommend you consider using SQLite in your next project. It's a mature and stable project that can handle huge amounts of data. Many programming language have drivers to SQLite database, which makes it a good storage option.

I've simplified the code as much as I could to show the more interesting points. There are several places where you can try an improve it:
- Add a retry logic to `Flush`
- Do more error checking in `Close`
- Have the Go HTTP sever invoke the Python code every hour
- Run more analysis on the Python side

Have fun with the code, [let me know](mailto:miki@353solutions.com) what crazy things you did.  
