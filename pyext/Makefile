_checksig.so: *.go
	echo go build -buildmode=c-shared -o $@

clean:
	-rm *.h *.so
	-rm -r build/ checksig.egg-info/ dist/

test-dist:
	docker build -f Dockerfile.test-dist .
