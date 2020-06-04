package main

//go:generate mkdir -p pb
//go:generate protoc --go_out=plugins=grpc:pb --go_opt=paths=source_relative outliers.proto
