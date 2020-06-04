# Go ↔ Python: Part I - gRPC

### Introduction

Like tools, programming languages tend to solve problems they are designed to. You _can_ use a knife to tighten a screw, but it's better to use a screwdriver., Plus there is less chance of you getting hurt in the process.

The Go programming language shines when writing high throughput services, and Python shines when used for data science. In this series of blog posts, we're going to explore how you can use each language to do the part it's better and explore various methods of communication between Go & Python.

In this post, we’ll learn how Go and Python programs can communicate between each other using [gRPC](https://grpc.io/). This post assumes you know how to program both in Go and Python. You don’t need to be an expert in either of the languages, but the post assumes you know the basics of each language.

### gRPC Overview

gRPC is an RPC (remote procedure call) framework from Google. It uses [Protocol Buffers](https://developers.google.com/protocol-buffers) as serialization format and uses [HTTP2](https://en.wikipedia.org/wiki/HTTP/2) as the transport medium.

By using these two well established technologies, you gain access to a lot of knowledge and tooling that's already available. Many companies I consult with use gRPC to connect internal services.

Another advantage of using protocol buffers is that you write the message definition once, and then generate bindings to every language from the same source. This means that various services can be written in different programming languages and all the applications agree on the format of messages.

Protocol buffers is also an efficient binary format, you gain both faster serialization times and less bytes over the wire, and this alone will save you money. Benchmarking on my machine, serialization time compared to JSON is about 7.5 faster and the data generated is about 4 times smaller.

### Example: Outlier Detection

[Outlier detection](https://en.wikipedia.org/wiki/Anomaly_detection), also called anomaly detection, is a way of finding suspicious values in data. Modern systems collect a lot of metrics from their services, and it's hard to come up with simple thresholds for when to wake someone at 2am.

We're going to have a Go service that collects metrics and then send them to a Python service using gRPC to perform outlier detection. 

### Project Structure

There are many ways to structure a multi-service project. We're going with a simple approach of having Go as the main project and Python as a sub-project in the source tree.

**Listing 1**  
```
.
├── client.go
├── gen.go
├── go.mod
├── go.sum
├── outliers.proto
├── pb
│   └── outliers.pb.go
└── py
    ├── Makefile
    ├── outliers_pb2_grpc.py
    ├── outliers_pb2.py
    ├── requirements.txt
    └── server.py
```

Listing1 shows the directory structure of our project. The project is using [Go modules](https://blog.golang.org/using-go-modules). The module name for the project is defined in the `go.mod` file (see listing 2). I mention this since we’re going to reference the module name ( `github.com/ardanlabs/metrics`) in several places.

**Listing 2**
```
01 module github.com/ardanlabs/metrics
02
03 go 1.14
04
05 require (
06     github.com/golang/protobuf v1.4.2
07     google.golang.org/grpc v1.29.1
08     google.golang.org/protobuf v1.23.0
09 )
```

Listing 2 shows what the `go.mod` file looks like for our project. You can see on line 01 where the name of the module is defined.

### Defining Messages & Services 

In gRPC, you start by writing a `.proto` file which defines the messages being sent and the RPC methods.

**Listing 3**  
```
01 syntax = "proto3";
02 import "google/protobuf/timestamp.proto";
03 package pb;
04
05 option go_package = "github.com/ardanlabs/metrics/pb";
06
07 message Metric {
08    google.protobuf.Timestamp time = 1;
09    string name = 2;
10    double value = 3;
11 }
12
13 message OutliersRequest {
14    repeated Metric metrics = 1;
15 }
16
17 message OutliersResponse {
18    repeated int32 indices = 1;
19 }
20
21 service Outliers {
22    rpc Detect(OutliersRequest) returns (OutliersResponse) {}
23 }
```

Listing 3 shows what the `outliers.proto` looks like. Important things to mention are on line 02 in listing 3, this is where we import the protocol buffers definition of `timestamp`. Then on line 05 we define the full Go package name - github.com/ardanlabs/metrics/pb

A metric is a measurement of resource usage, it is used for monitoring and diagnostics of your system. We define a `Metric` on line 07, it has a timestamp, name (e.g. "CPU") and float value. For example we can say that at `2020-03-14T12:30:14` we measured 41.2% CPU utilization.

Every RPC method has an input type (or types) and an output type. Our method `Detect` (line 22) uses an `OutliersRequest` message type (line 13) as the input and an `OutliersResponse` message type (line 17) for the output. The `OutliersRequest` message type is a list/slice of Metric and the `OutliersResponse` message type is a list/slice of indices where the outlier values were found. For example if we have the values of `[1, 2, 100, 1, 3, 200, 1]`, the result will be `[2]` which is the index of the outlier 100.

### Python Service

In this section, we’ll go over the Python service code. 

**Listing 4**  
```
.
├── client.go
├── gen.go
├── go.mod
├── go.sum
├── outliers.proto
├── pb
│   └── outliers.pb.go
└── py
    ├── Makefile
    ├── outliers_pb2_grpc.py
    ├── outliers_pb2.py
    ├── requirements.txt
    └── server.py
```

Listing 4 shows how the code for the Python service is in the `py` directory off the root of the project.

To generate the Python bindings you'll need to install the `protoc` compiler, which you can download from [here](https://developers.google.com/protocol-buffers/docs/downloads). You can also install the compiler using your operating system package manager (e.g. `apt-get`, `brew` ...)

Once you have the compiler installed, you'll need to install the Python `grpcio-tools` package.

_Note: I highly recommend using a virtual environment for all your Python projects. Read [this](https://docs.python.org/3/tutorial/venv.html) to learn more._

**Listing 5**  
```
$ cat requirements.txt

OUTPUT:
grpcio-tools==1.29.0
numpy==1.18.4

$ python -m pip install -r requirements.txt
```

Listing 5 shows how to inspect and install the external dependencies for our Python project. The `requirements.txt` specifies the external dependencies for the project, very much like how `go.mod` specifies dependencies for Go projects.

As you can see from the output of the `cat` command, we need two external dependencies `grpcio-tools` and [numpy](https://numpy.org/) . A good practice is to have this file in source control and always version your dependencies (e.g. `numpy==1.18.4`), similar to what you would do with your `go.mod` for your Go projects.

Once you have the dependencies installed, you can generate the Python bindings.

**Listing 6**  
```
$ python -m grpc_tools.protoc \
    -I.. --python_out=. --grpc_python_out=. \
    ../outliers.proto
```

Listing 6 shows how to generate the Python binding for the gRPC support. Let's break this long command in listing 5 down:

* `python -m grpc_tools.protoc` runs the `grpc_tools.protoc` module as a script.
* `-I..` tells the tool where `.proto` files can be found.
* `--python_out=.` tells the tool to generate the protocol buffers serialization code in the current directory.
* `--grpc_python_out=.` tells the tool to generate the gRPC code in the current directory.
* `../outliers.proto` is the name of the protocol buffers + gRPC definitions file.

This Python command will run without any output, and at the end you'll see two new files: `outliers_pb2.py` which is the protocol buffers code and `outliers_pb2_grpc.py` which is the gRPC client and server code.

_Note: I usually use a `Makefile` to automate tasks in Python projects and create a `make` rule to run this command. I add the generated files to source control so that the deployment machine won't have to install the `protoc` compiler.

To write the Python service, you need to inherit from the `OutliersServicer` defined in `outliers_pb2_grpc.py` and override the `Detect` method. We're going to use the numpy package and use a simple method of picking all the values that are more than two [standard deviations](https://en.wikipedia.org/wiki/Standard_deviation) from the [mean](https://en.wikipedia.org/wiki/Mean).

**Listing 7**   
```
01 import logging
02 from concurrent.futures import ThreadPoolExecutor
03
04 import grpc
05 import numpy as np
06
07 from outliers_pb2 import OutliersResponse
08 from outliers_pb2_grpc import OutliersServicer, add_OutliersServicer_to_server
09
10
11 def find_outliers(data: np.ndarray):
12     """Return indices where values more than 2 standard deviations from mean"""
13     out = np.where(np.abs(data - data.mean()) > 2 * data.std())
14     # np.where returns a tuple for each dimension, we want the 1st element
15     return out[0]
16
17
18 class OutliersServer(OutliersServicer):
19     def Detect(self, request, context):
20          logging.info('detect request size: %d', len(request.metrics))
21          # Convert metrics to numpy array of values only
22          data = np.fromiter((m.value for m in request.metrics), dtype='float64')
23          indices = find_outliers(data)
24          logging.info('found %d outliers', len(indices))
25          resp = OutliersResponse(indices=indices)
26          return resp
27
28
29 if __name__ == '__main__':
30     logging.basicConfig(
31          level=logging.INFO,
32          format='%(asctime)s - %(levelname)s - %(message)s',
33	)
34     server = grpc.server(ThreadPoolExecutor())
35     add_OutliersServicer_to_server(OutliersServer(), server)
36     port = 9999
37     server.add_insecure_port(f'[::]:{port}')
38     server.start()
39     logging.info('server ready on port %r', port)
40     server.wait_for_termination()
```

Listing 7 shows the code in the `server.py` file. This is all the code we needed to write the Python service.

**Listing 8**  
```
$ python server.py

OUTPUT:
2020-05-23 13:45:12,578 - INFO - server ready on port 9999
```
Listing 8 shows how to run the service.

### Go Client

Now that we have our Python service running, we can write the Go client that will communicate with it.

We'll start by generating Go bindings for gRPC. To automate this process, I usually have a file called `gen.go` with a `go:generate`  command to generate the bindings. You will need to download the `github.com/golang/protobuf/protoc-gen-go` module which is the gRPC plugin for Go.

**Listing 9**  
```
01 package main
02
03 //go:generate mkdir -p pb
04 //go:generate protoc --go_out=plugins=grpc:pb --go_opt=paths=source_relative outliers.proto
```

Listing 9 shows the `gen.go` file and how `go:generate` is used to execute the gRPC plugin to generate the bindings.

Let's break down the command on line 04 which generates the bindings:

* `protoc` is the protocol buffer compiler.
* `--go-out=plugins=grpc:pb` tells `protoc` to use the gRPC plugin and place the files in the `pb` directory.
* `--go_opt=source_relative` tells `protoc` to generate the code in the `pb` directory relative to the current directory.
* `outliers.proto` is the name of the protocol buffers + gRPC definitions file.

When you run `go generate` in the shell, you should see no output, but there will be a new file called `outliers.pb.go` in the `pb` directory.

**Listing 10**  
```
.
├── client.go
├── gen.go
├── go.mod
├── go.sum
├── outliers.proto
├── pb
│   └── outliers.pb.go
└── py
    ├── Makefile
    ├── outliers_pb2_grpc.py
    ├── outliers_pb2.py
    ├── requirements.txt
    └── server.py
```

Listing 10 shows the `pb` directory and the new file `outliers.pb.go` that was generated by the `go generate` call. I add the `pb` directory to source control so if the project is cloned to a new machine the project will work without requiring `protoc` to be installed on that machine.

Now we can build and run the Go client.

**Listing 11**  
```
01 package main
02
03 import (
04     "context"
05     "log"
06     "math/rand"
07     "time"
08
09     "google.golang.org/grpc"
10     pbtime "google.golang.org/protobuf/types/known/timestamppb"
11
12     "github.com/ardanlabs/metrics/pb"
13 )
14
15 func main() {
16     addr := "localhost:9999"
17     conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
18     if err != nil {
19          log.Fatal(err)
20     }
21     defer conn.Close()
22
23     client := pb.NewOutliersClient(conn)
24     req := pb.OutliersRequest{
25          Metrics: dummyData(),
26     }
27
28     resp, err := client.Detect(context.Background(), &req)
29     if err != nil {
30          log.Fatal(err)
31     }
32     log.Printf("outliers at: %v", resp.Indices)
33 }
34
35 func dummyData() []*pb.Metric {
36     const size = 1000
37     out := make([]*pb.Metric, size)
38     t := time.Date(2020, 5, 22, 14, 13, 11, 0, time.UTC)
39     for i := 0; i < size; i++ {
40          m := &pb.Metric{
41               Time: Timestamp(t),
42               Name: "CPU",
43               // Normally we're below 40% CPU utilization
44               Value: rand.Float64() * 40,
45          }
46          out[i] = m
47          t.Add(time.Second)
48     }
49     // Create some outliers
50     out[7].Value = 97.3
51     out[113].Value = 92.1
52     out[835].Value = 93.2
53     return out
54 }
55
56 // Timestamp converts time.Time to protobuf *Timestamp
57 func Timestamp(t time.Time) *pbtime.Timestamp {
58     return &pbtime.Timestamp {
59         Seconds: t.Unix(),
60         Nanos: int32(t.Nanosecond()),
61     }

62 }
```

Listing 11 shows the code from `client.go`. The code  fills an `OutliersRequest` value on line 24  with some dummy data (generated by the `dummyData` function on line 35) and then on line 28 calls the Python service. The call to the Python service returns an `OutliersResponse` value.

Let's break down the code a little more:

* On line 17, we connect to the Python server, using the `WithInsecure` option since the Python server we wrote doesn’t support HTTPS.
* On line 23, we create a new `OutliersClient` with the connection we made on line 17.
* On line 24, we create the gPRC request.
* On line 28, we perform the actual gRPC call. Every gRPC call has a `context.Context` as first parameter, allowing you to control timeouts and cancellations.
* gRPC has it’s own implementation of a `Timestamp` struct. On line 57, we have a utility function to convert from Go’s `time.Time` value to gRPC `Timestamp` value.

**Listing 12**  
```
$ go run client.go

OUTPUT:
2020/05/23 14:07:18 outliers at: [7 113 835]
```

Listing 12 shows you how to run the Go client. This assumes the Python server is running on the same machine.

### Conclusion

gRPC makes it easy and safe to pass messages from one service to another. You can maintain one place where all data types and methods are defined, and there is great tooling and best practices for the gRPC framework.

The whole code: `weather.proto`, `py/server.py` and `client.go` is less than a 100 lines. You can view the project code [here](https://github.com/ardanlabs/python-go/tree/master/grpc).

There is much more to gRPC like timeout, load balancing, TLS, and streaming. I highly recommend going over the [official site](https://grpc.io/) read the documentation and play with the provided examples.

In the next post in this series, we’ll flip the roles and have Python call a Go service.
