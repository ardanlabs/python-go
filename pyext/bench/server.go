package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/ardanlabs/python-go/pyext/bench/pb"
)

type BenchServer struct {
	pb.UnimplementedBenchServer
}

func (b *BenchServer) Bench(context.Context, *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

func main() {
	addr := "localhost:8888"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("server listening on %s", addr)

	srv := grpc.NewServer()
	pb.RegisterBenchServer(srv, &BenchServer{})
	srv.Serve(lis)
}
