all:
	$(error please pick a target)


_checksig.so: *.go
	go build -buildmode=c-shared -o $@

so: _checksig.so

clean:
	-rm *.h *.so

test-dist:
	docker build -f Dockerfile.test .
