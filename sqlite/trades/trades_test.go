package trades_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/ardanlabs/python-go/sqlite/trades"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func tempFile(require *require.Assertions) string {
	file, err := ioutil.TempFile("", "*.db")
	require.NoError(err)
	file.Close()
	return file.Name()
}

func TestAdd(t *testing.T) {
	require := require.New(t)

	dbFile := tempFile(require)
	t.Logf("db file: %s", dbFile)
	db, err := trades.NewDB(dbFile)
	require.NoError(err)
	defer db.Close()

	trade := trades.Trade{
		Time:   time.Now(),
		Symbol: "MSFT",
		Price:  216.39,
		IsBuy:  false,
	}

	err = db.Add(trade)
	require.NoError(err)

	// TODO: Check database
}

func BenchmarkAdd(b *testing.B) {
	require := require.New(b)
	dbFile := tempFile(require)
	b.Logf("db file: %s", dbFile)

	db, err := trades.NewDB(dbFile)
	require.NoError(err)
	trade := trades.Trade{
		Time:   time.Now(),
		Symbol: "MSFT",
		Price:  216.39,
		IsBuy:  false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := db.Add(trade)
		require.NoError(err)
	}
}

func ExampleDB() {
	dbFile := "/tmp/db-test" + time.Now().Format(time.RFC3339)
	db, err := trades.NewDB(dbFile)
	if err != nil {
		fmt.Println("ERROR: create -", err)
		return
	}
	defer db.Close()

	const count = 10_000
	for i := 0; i < count; i++ {
		trade := trades.Trade{
			Time:   time.Now(),
			Symbol: "AAPL",
			Price:  rand.Float64() * 200,
			IsBuy:  i%2 == 0,
		}
		if err := db.Add(trade); err != nil {
			fmt.Println("ERROR: insert - ", err)
			return
		}
	}

	fmt.Printf("inserted %d records\n", count)
	// Output:
	// inserted 10000 records
}
