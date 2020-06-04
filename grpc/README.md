# Go ↔ Python: Part I - gRPC

### Introduction

Like tools, programming languages tend to solve problems they are designed to. You can use a knife to tighten a screw, but it's better to use a screwdriver, and there are less chances of you getting hurt in the process.

The Go programming language shines when writing high throughput services, and Python shines when doing data science. In this series of blog posts, we're going to explore how you can use each language to do the part it's better at. We'll see how you can communicate between and Python efficiently and explore the tradeoffs for each approach.

In this post we’ll use [gRPC](https://grpc.io/) pass messages from Go to Python and back.

### gRPC Overview

gRPC is an RPC (remote procedure call) framework from Google. It uses [Protocol Buffers](https://developers.google.com/protocol-buffers) as serialization format and uses [HTTP2](https://en.wikipedia.org/wiki/HTTP/2) as the transport medium.

By using these two well established technologies, you gain access to a lot of knowledge and tooling that's already available. Many companies I consult with use gRPC to connect internal services.

One more advantage of using protocol buffers is that you write the message definition once, and then generate bindings to every language from the same source. This means that various micro services can be written in different programming languages and agree on the format of message passed around.

Protocol buffers is also an efficient binary format, you gain both faster serialization times and less bytes over the wire, and this alone will save you money. Benchmarking on my machine, serialization time compared to JSON is about 7.5 faster and the data generated is about 4 times smaller.

### Example: Outlier Detection

[Outlier detection](https://en.wikipedia.org/wiki/Anomaly_detection), also called anomaly detection, is a way of finding suspicious values in data. Modern systems collect a lot of metrics on their services, and it's hard to come up with simple thresholds for when to wake someone at 2am.

We're going to have a Go service that collects metrics and then send them to a Python process using gRPC to do outlier detection. 

### Project Structure

There are many ways to structure a multi-service project. We're going with the simple approach of having Go as the main project and the Python project in a sub directory.

Here's how our directory structure will look like

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

The `go.mod` file defines the module which is `github.com/ardanlabs/metrics`.

### Defining Messages & Services 

In gRPC, you start by writing a `.proto` file which defines the messages being sent and the RPC methods.

Here's how outliers.proto looks like:

**Listing 2**  
```
syntax = "proto3";
import "google/protobuf/timestamp.proto";
package pb;

option go_package = "github.com/ardanlabs/metrics/pb";

message Metric {
    google.protobuf.Timestamp time = 1;
    string name = 2;
    double value = 3;
}

message OutliersRequest {
    repeated Metric metrics = 1;
}

message OutliersResponse {
    repeated int32 indices = 1;
}

service Outliers {
    rpc Detect(OutliersRequest) returns (OutliersResponse) {}
}
```

We start with a preamble that does an import of timestamp, defines a package and full Go package path.

A `Metric` is a piece of information with time stamp, name (e.g. "CPU") and a float value.

Every RPC method has an input type (or types) and an output type. Our input is `OutliersRequest` which is a list/slice of Metric. The r`OutliersResponse` type is a list/slice of indices where the outlier values are.

### Python Server

The Python server lives in the `py` directory.

To generate the Python bindings you'll need the `protoc` compiler, which you can download from [here](https://developers.google.com/protocol-buffers/docs/downloads) or install using your operating system package manager (e.g. `apt-get`, `brew` ...)

Next you'll need to install the `grpcio-tools` Python package. I highly recommend using a virtual environment for all your Python projects.

**Listing 3**  
```
$ cat requirements.txt
grpcio-tools==1.29.0
numpy==1.18.4
$ python -m pip install -r requirements.txt
```

You should always version your requirements and place them in version control (e.g. git). Here we use the `requirements.txt` file for storing requirements.

Once you have the tools installed, you can generate the Python bindings:

**Listing 4**  
```
$ python -m grpc_tools.protoc \
	    -I.. --python_out=. --grpc_python_out=. \
	    ../outliers.proto
```

Let's break this long command down:

- `python -m grpc_tools.protoc` runs the `grpc_tools.protoc` module as a script
- `-I..` tells the tool where `.proto` files can be found
- `--python_out=.` tells the tool to generate the protocol buffers serialization code in the current directory
- `--grpc_python_out=.` tells the tool to generate the gRPC code in the current directory
- `../outliers.proto` is the name of the protocol buffers + gRPC definitions file

This command will run without any output, and at the end you'll see two new files: `outliers_pb2.py` which is the protocol buffers code and `outliers_pb2_grpc.py` with the gRPC client and server code.

I usually use a `Makefile` to automate tasks in Python projects and have a `make` rule to run this command. I add the generated files to source control so that the deployment machine won't have to install the `protoc` compiler.

To write the server, you need to inherit from the `OutliersServicer` defined in `outliers_pb2_grpc.py` and override the `Detect` method. We're going to use the [numpy](https://numpy.org/) package and use a simple method of picking all the values that are more than two [standard deviations](https://en.wikipedia.org/wiki/Standard_deviation) from the [mean](https://en.wikipedia.org/wiki/Mean).

**Listing 5**  
```
import logging
from concurrent.futures import ThreadPoolExecutor

import grpc
import numpy as np

from outliers_pb2 import OutliersResponse
from outliers_pb2_grpc import OutliersServicer, add_OutliersServicer_to_server


def find_outliers(data: np.ndarray):
    """Return indices where values more than 2 standard deviations from mean"""
    out = np.where(np.abs(data - data.mean()) > 2 * data.std())
    # np.where returns a tuple for each dimension, we want the 1st element
    return out[0]


class OutliersServer(OutliersServicer):
    def Detect(self, request, context):
        logging.info('detect request size: %d', len(request.metrics))
        # Convert metrics to a numpy array of values only
        data = np.fromiter((m.value for m in request.metrics), dtype='float64')
        indices = find_outliers(data)
        logging.info('found %d outliers', len(indices))
        resp = OutliersResponse(indices=indices)
        return resp


if __name__ == '__main__':
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s',
    )
    server = grpc.server(ThreadPoolExecutor())
    add_OutliersServicer_to_server(OutliersServer(), server)
    port = 9999
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    logging.info('server ready on port %r', port)
    server.wait_for_termination()
```

Now you can run the server:
**Listing 6**  
```
$ python server.py
```

If all is well, you should see a log message like:

**Listing 7**  
```
2020-05-23 13:45:12,578 - INFO - server ready on port 9999
```

### Go Client

Now that we have our server running, we can write the Go code to communicate with it. How to run and deploy the Python server is out of the scope for this post.

We'll start by generating Go bindings for gRPC. To automate this process I usually have a file called `gen.go` with a `go:generate` rule to run the commands. Other than the `protoc` compiler, you will need to `go get github.com/golang/protobuf/protoc-gen-go` which is the gRPC plugin for Go.

**Listing 8**  
```
package main

//go:generate mkdir -p pb
//go:generate protoc --go_out=plugins=grpc:pb --go_opt=paths=source_relative outliers.proto
```

Let's break this command:
- `protoc` is the protocol buffer compiler
- `--go-out=plugins=grpc:pb` tells `protoc` to use the grpc plugin and place the files in the `pb` directory
- `--go_opt=source_relative` tells `protoc` to generate the `pb` directory relative to the current directory
- `outliers.proto` is the name of the protocol buffers + gRPC definitions file

When you run `go generate` you should see no output, but there will be a new file called `outliers.pb.go` in the `pb` directory. I add the `pb` directory to source control so `go get` of this package will work without requiring `protoc` installed on the client machine.

Now we can run the client, we'll fill an `OutliersRequest` struct with some dummy data and when calling the server we'll get back an `OutliersResponse` struct.

The only difference from plain Go struct will be the `Time` field which is a pointer to `google.golang.org/protobuf/types/known/timestamppb.Timestamp`. I usually write some conversion functions from `time.Time` to `Timestamp` and back. Future versions of protocol buffers for Go should have similar utilities.

We'll also have a `dummyData` function to generate some dummy data with outliers at indices 7, 113 and 835.

**Listing 9**  
```
package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	pbtime "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ardanlabs/metrics/pb"
)

func main() {
	addr := "localhost:9999"
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewOutliersClient(conn)
	req := &pb.OutliersRequest{
		Metrics: dummyData(),
	}

	resp, err := client.Detect(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("outliers at: %v", resp.Indices)
}

func dummyData() []*pb.Metric {
	const size = 1000
	out := make([]*pb.Metric, size)
	t := time.Date(2020, 5, 22, 14, 13, 11, 0, time.UTC)
	for i := 0; i < size; i++ {
		m := &pb.Metric{
			Time: Timestamp(t),
			Name: "CPU",
			// normally we're below 40% CPU utilization
			Value: rand.Float64() * 40,
		}
		out[i] = m
		t.Add(time.Second)
	}
	// Create some outliers
	out[7].Value = 97.3
	out[113].Value = 92.1
	out[835].Value = 93.2
	return out
}

// Timestamp converts time.Time to protobuf *Timestamp
func Timestamp(t time.Time) *pbtime.Timestamp {
	var pbt pbtime.Timestamp
	pbt.Seconds = t.Unix()
	pbt.Nanos = int32(t.Nanosecond())
	return &pbt
}
```

Assuming the Python server is running on the same machine, we can run the client:

**Listing 10**  
```
$ go run client.go
2020/05/23 14:07:18 outliers at: [7 113 835]
```


### Conclusion

gRPC makes it easy and safe to call from one service to another. You get one place where all data types and methods are defined and great tooling and best practices around the framework.

The whole code: `weather.proto`, `py/server.py` and `client.go` is less than a 100 lines. You can view the project at **FIXME**

There is much more to gRPC - timeout, load balancing, TLS, streaming and more. I highly recommend going over the [official site](https://grpc.io/) read the documentation and play with the provided examples.
