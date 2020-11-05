all:
	$(error please pick a target)

run-httpd:
	go run .

web-bench:
	wrk -s wrk.lua http://localhost:8080/trade
