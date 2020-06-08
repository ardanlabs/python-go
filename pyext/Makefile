all:
	$(error please pick a target)


_checksig.so: *.go
	echo go build -buildmode=c-shared -o $@

so: _checksig.so

clean:
	-rm *.h *.so
	-rm -r build/ checksig.egg-info/ dist/

dists:
	-rm -r dist
	python setup.py sdist
	python setup.py bdist_wheel

test-dist: dists
	docker build -f Dockerfile.test-dist .
