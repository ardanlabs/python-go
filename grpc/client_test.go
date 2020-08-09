package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/ardanlabs/python-go/grpc/pb"
)

func BenchmarkClient(b *testing.B) {
	require := require.New(b)

	addr := "localhost:9999"
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	require.NoError(err, "connect")
	defer conn.Close()

	client := pb.NewOutliersClient(conn)
	req := &pb.OutliersRequest{
		Metrics: dummyData(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Detect(context.Background(), req)
		require.NoError(err, "detect")
	}

}
