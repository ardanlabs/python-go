# Go ↔ Python: Part IV Using Python in Memory

### Introduction

In [a previous post](https://www.ardanlabs.com/blog/2020/06/python-go-grpc.html) we used [gRPC](https://grpc.io/) to call Python code from Go. gRPC is a great framework but there is a performance cost to it - every function call need to marshal the arguments using [protobuf](https://developers.google.com/protocol-buffers), make a network call over [HTTP/2](https://en.wikipedia.org/wiki/HTTP/2) and then un-marshal the result using `protobuf`.

In this blog post we'll get rid of the network layer and, to some extent, the marshalling. We'll do this by using [cgo](https://golang.org/cmd/cgo/) to interact with Python as a shared library.

_Note: This blog is not for the weak of heart - we'll pass Go slices directly to Python, use Python allocated memory in Go and fool the Go compiler with ugly tricks. You better have some harsh performance requirements to follow this path. So buckle up Buttercup, it's going to be a wild ride._

I'd like to start by thanking the awesome people at the (aptly named) `#darkarts` channel in [Gophers Slack](https://gophers.slack.com/) for their help and insights.

I'm not going to cover all of the code, this blog post is long as is. You can find the code [on github](https://github.com/ardanlabs/python-go/tree/master/py-in-mem) and I did my best to document it. Feel free to reach out and ask me questions.

### A Crash Course in Python Internals

The Python most of use use is call `CPython`, it's written in C and is designed to be extended and embedded using C. In this section we'll cover some topics that will help you understand the code better.

_Note: The API is [well](https://docs.python.org/3/extending/index.html) [documented](https://docs.python.org/3/c-api/index.html), and there even [a book](https://realpython.com/products/cpython-internals-book/) coming up._

Every Python value is a `PyObject *`, most Python's API function will return a `PyObject *` or get a `PyObject *` as an argument.

Errors are signaled by returning `NULL`, and then you can use the `PyErr_Occurred` function to get the last exception raised.

CPython uses a [reference counting](https://en.wikipedia.org/wiki/Reference_counting) garbage collector.  It means that every `PyObject *` has a counter for how many variables are referencing it. Once the reference counter reaches 0, Python frees the objects memory. As a programmer, you need to take care to decrease the reference counter using `Py_DECREF` once you're done with an object.

### Code Overview

We'll have a Python function that uses [numpy](https://numpy.org/) to do [outlier detection](https://en.wikipedia.org/wiki/Anomaly_detection) on a series of floating point values.

Our Go code is going to load and initialize the Python shared library.

When we want to call the Python function, we'll follow these steps:
* Convert the Go `[]float64` parameter to a C `double *` (`outliers.go`)
* Create a numpy array from the C `double *` (`glue.c`)
* Call the Python function with the numpy array (`glue.c`)
* Get back a numpy array with indices of outliers (`glue.c`)
* Extract C `long *` from the numpy array (`glue.c`)
* Convert the C `long *` to Go `[]int` and return it from the Go function
  (`outliers.go`)

The Go code is in `outliers.go`, there's some C code in `glue.c` and finally an outlier detection Python function is in `outliers.py`.

You can see example usage [here](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/doc.go).

### Code Highlights

**Listing 1: outliers.go `initialize`**

```
19 var (
20 	initOnce sync.Once
21 	initErr  error
22 )
23 
24 // initialize Python & numpy, idempotent
25 func initialize() {
26 	initOnce.Do(func() {
27 		C.init_python()
28 		initErr = pyLastError()
29 	})
30 }
```

Listing 1 shows how we initialize Python. On line 20 we use a [sync.Once](https://golang.org/pkg/sync/#Once) to make sure we initialize only ince. On line 26 we use `initOnce` to call the initialization code and on line 28 we set the `initErr` from the last Python error.

**Listing 2: outliers.go `NewOutliers`**
```
37 // NewOutliers returns an new Outliers using moduleName.funcName Python function
38 func NewOutliers(moduleName, funcName string) (*Outliers, error) {
39 	initialize()
40 	if initErr != nil {
41 		return nil, initErr
42 	}
43 
44 	fn, err := loadPyFunc(moduleName, funcName)
45 	if err != nil {
46 		return nil, err
47 	}
48 
49 	return &Outliers{fn}, nil
50 }
```

Listing 2 shows creation of an `Outliers` object.  On line 38 we have the code to create a new `Outliers` struct. On lines 39-42 we make sure Python is initialized and there's no error. On line 44 we get a pointer to the Python function, same as doing an `import` statement in Python.

**Listing 3: outliers.go `Detect`**
```
52 // Detect returns slice of outliers indices
53 func (o *Outliers) Detect(data []float64) ([]int, error) {
54 	if o.fn == nil {
55 		return nil, fmt.Errorf("closed")
56 	}
57 
58 	// Convert []float64 to C double*
59 	carr := (*C.double)(&(data[0]))
60 	res := C.detect(o.fn, carr, (C.long)(len(data)))
61 	// Tell Go's GC to keep data alive until here
62 	runtime.KeepAlive(data)
63 	if res.err != 0 {
64 		return nil, pyLastError()
65 	}
66 
67 	// Ugly hack to convert C.long* to []int
68 	ptr := unsafe.Pointer(res.indices)
69 	arr := (*[1 << 20]int)(ptr)
70 	// Create a copy managed by Go
71 	indices := make([]int, res.size)
72 	copy(indices, arr[:res.size])
73 	// Free Python object
74 	C.py_decref(res.obj)
75 	return indices, nil
76 }
```

Listing 3 shows the code for `Outliers.Detect` method. On line On line 59 we convert Go `[]float64` to a C `double *`. On line 60 we call the Python function via the C layer and get back a result. On line 62 we tell Go's garabge collector that it can't reclaim the `data` slice until this line.  On lines 63-65 we check if there was an error calling Python. On lines 68,69 we fool the Go compiler to think that the C `double *` is a Go array. On lines 71,72 we copy the data from Python to a new slice. On line 74 we decrement the Python return value reference count.

**Listing 4: outliers.go `Outliers.Close` method**

```
78 // Close frees the underlying Python function
79 // You can't use the object after closing it
80 func (o *Outliers) Close() {
81 	if o.fn == nil {
82 		return
83 	}
84 	C.py_decref(o.fn)
85 	o.fn = nil
86 }
```

Listing 4 shows `Outliers.Close` method. On line 84 we decrement the Python function object reference count and on line 85 we set the `fn` field to `nil` to signal the `Outliers` object is closed.

If you're curious about the C code, have a look at [glue.c](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/glue.c).


### Building

The glue code is using header files from Python and numpy. In order to build we need to tell [cgo](https://golang.org/cmd/cgo/) where to find these header files. 

**Listing 5: outliers.go `cgo` directives**
```
11 /*
12 #cgo pkg-config: python3
13 #cgo LDFLAGS: -lpython3.8
14 
15 #include "glue.h"
16 */
17 import "C"
```

Listing 5 shows the `cgo` directives.

On line 12 we use the [pkg-config](https://www.freedesktop.org/wiki/Software/pkg-config/) to find C compiler directives for Python. On line 13 we tell `cgo` to use the Python 3.8 shared library.  On line 15 we import the C code definitions from `glue.h` and on line 17 we have the `import "C"` directive that *must* come right after the comment.

numpy headers are a bit more tricky. numpy does not come with a `pkg-config` file but has a Python function that will tell you where the headers are. For security reasons, you can't have `cgo` run arbitrary commands. I opted to asking the user to set the `CGO_CFLAGS` environment variable before building or installing the package.

**Listing 6: Build commands**
```
01 $ export CGO_CFLAGS="-I $(python -c 'import numpy; print(numpy.get_include())'"
02 $ go build
```

Listing 6 shows how to build the package. One line 01 we set `CGO_CFLAGS` a value printed from a short Python program that prints the location of the numpy header files. On line 02 we build the package.

I like to use [make](https://www.gnu.org/software/make/) to automate such tasks. Have a look at the [Makefile](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/Makefile) to learn more.

### Conclusion

This code is risky and error prone. There we moments when developing it that I've considered a career change to goat herding at some remote location. On the plus side, benchmarking on my machine shows this code is about 45 times faster than the equivalent [gRPC code](https://www.ardanlabs.com/blog/2020/06/python-go-grpc.html) code. And even though I'm programming in Go for 10 years and in Python close to 25 - I learned some new things.
