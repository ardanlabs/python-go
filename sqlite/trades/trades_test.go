package trades_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ardanlabs/python-go/sqlite/trades"
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
