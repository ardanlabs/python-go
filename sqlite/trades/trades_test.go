package trades_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/ardanlabs/python-go/sqlite/trades"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	require := require.New(t)

	db, err := trades.NewDB("/tmp/trades.db")
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
	file, err := ioutil.TempFile("", "*.db")
	require.NoError(err)
	file.Close()

	b.Logf("db file: %s", file.Name())

	db, err := trades.NewDB(file.Name())
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
