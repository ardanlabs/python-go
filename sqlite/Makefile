all:
	$(error please pick a target)

run-httpd:
	go run .

web-bench:
	wrk -s wrk.lua http://localhost:8080/trade

gen-data:
	rm -f trades.db
	go run db.go create_data.go

report:
	python analyze_trades.py \
	    --db trades.db \
	    --start 2020-07-02T01:00:00 \
	    --end 2020-07-02T02:00:00

count:
	echo 'SELECT COUNT(time) FROM trades' | sqlite3 trades.db

times:
	echo 'SELECT MIN(time), MAX(time) FROM trades' | sqlite3 trades.db
