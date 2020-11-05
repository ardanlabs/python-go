import pandas as pd
import sqlite3
from contextlib import closing
from datetime import datetime

select_sql = """
SELECT * FROM trades
WHERE time >= ? AND time <= ?
"""


def load_trades(db_file, start_time, end_time):
    conn = sqlite3.connect(db_file, detect_types=sqlite3.PARSE_DECLTYPES)
    with closing(conn) as db:
        return pd.read_sql(select_sql, db, params=(start_time, end_time))


def time_type(value):
    return datetime.strptime(value, "%Y-%m-%dT%H:%M:%S")


if __name__ == "__main__":
    from argparse import ArgumentParser, FileType
    from sys import stdout

    parser = ArgumentParser()
    parser.add_argument(
        "--db", help="database file", type=FileType("r"), required=True)
    parser.add_argument(
        "--start", help="start time (YYYY-MM-DDTHH:MM:SS)", type=time_type, required=True)
    parser.add_argument(
        "--end", help="end time (YYYY-MM-DDTHH:MM:SS)", type=time_type, required=True)

    args = parser.parse_args()
    df = load_trades(args.db.name, args.start, args.end)
    # Show price average per symbol/buy
    out = df.groupby(['symbol', 'buy'], as_index=False)['price'].mean()
    out.to_csv(stdout, index=False)
