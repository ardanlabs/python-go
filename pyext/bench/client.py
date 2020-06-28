import grpc

import bench_pb2 as pb
import bench_pb2_grpc as gpb


def call(chan):
    msg = pb.Empty()
    stub = gpb.BenchStub(chan)
    return stub.Bench(msg)


def connect(host, port):
    return grpc.insecure_channel(f'{host}:{port}')
