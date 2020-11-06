"""Analyze trades"""
import pandas as pd
import sqlite3
from contextlib import closing
from datetime import datetime

select_sql = """
SELECT * FROM trades
WHERE time >= ? AND time <= ?
"""


def load_trades(db_file, start_time, end_time):
    """Load trades from db_file in given time range."""
    conn = sqlite3.connect(db_file)
    with closing(conn) as db:
        df = pd.read_sql(select_sql, db, params=(start_time, end_time))

    # We can't use detect_types=sqlite3.PARSE_DECLTYPES here since Go is
    # inserting time zone and Python's sqlite3 doesn't handle it.
    # See https://bugs.python.org/issue29099
    df["time"] = pd.to_datetime(df["time"])
    return df


def average_price(df):
    """Return average price in df grouped by (stock, buy)"""
    return df.groupby(["symbol", "buy"], as_index=False)["price"].mean()


def time_type(value):
    return datetime.strptime(value, "%Y-%m-%dT%H:%M:%S")


if __name__ == "__main__":
    from argparse import ArgumentParser, FileType
    from sys import stdout

    parser = ArgumentParser()
    parser.add_argument("--db", help="database file", type=FileType("r"), required=True)
    parser.add_argument(
        "--start",
        help="start time (YYYY-MM-DDTHH:MM:SS)",
        type=time_type,
        required=True,
    )
    parser.add_argument(
        "--end", help="end time (YYYY-MM-DDTHH:MM:SS)", type=time_type, required=True
    )

    args = parser.parse_args()
    df = load_trades(args.db.name, args.start, args.end)
    out = average_price(df)
    out.to_csv(stdout, index=False)
