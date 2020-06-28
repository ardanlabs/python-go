all:
	$(error please pick a target)

bench:
	ipython bench.ipy

grpc: grpc-go grpc-python

grpc-go:
	mkdir -p pb
	protoc \
	    --go_out=plugins=grpc:pb \
	    --go_opt=paths=source_relative \
	    bench.proto

grpc-python:
	python -m grpc_tools.protoc \
	    -I. \
	    --python_out=. --grpc_python_out=. \
	    bench.proto

so:
	go build -buildmode=c-shared -o bench.so so.go
