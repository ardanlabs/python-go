# Go ↔ Python: Part V Using Python in Memory

### Introduction

In [a previous post](https://www.ardanlabs.com/blog/2020/06/python-go-grpc.html) we used [gRPC](https://grpc.io/) to call Python code from Go. gRPC is a great framework but there is a performance cost to it. Every function call need to marshal the arguments using [protobuf](https://developers.google.com/protocol-buffers), make a network call over [HTTP/2](https://en.wikipedia.org/wiki/HTTP/2) and then un-marshal the result using `protobuf`.

In this blog post, we'll get rid of the network layer and to some extent, the marshalling. We'll do this by using [cgo](https://golang.org/cmd/cgo/) to interact with Python as a shared library.


I'm not going to cover all of the code in order to keep this blog size down. You can find the code [on github](https://github.com/ardanlabs/python-go/tree/master/py-in-mem) and I did my best to document it. Feel free to reach out and ask me questions.

And finally, if you want to follow along you’ll need the following installed (apart from Go):
Python 3.8
numpy
A C compiler (such as gcc)

### A Crash Course in Python Internals

The Python most of us use is called `CPython`, it's written in C and is designed to be extended and embedded using C. In this section, we'll cover some topics that will help you understand the code we’ll show here better.

_Note: The Python C API is [well](https://docs.python.org/3/c-api/index.html), and there even [a book](https://realpython.com/products/cpython-internals-book/) coming up._

In Python, every value is a `PyObject *` and most of Python's API functions will return a `PyObject *` or will get a `PyObject *` as an argument. Also, errors are signaled by returning `NULL`, and you can use the `PyErr_Occurred` function to get the last exception raised.

CPython uses a [reference counting](https://en.wikipedia.org/wiki/Reference_counting) garbage collector which means that every `PyObject *` has a counter for how many variables are referencing it. Once the reference counter reaches 0, Python frees the object's memory. As a programmer, you need to take care to decrement the reference counter using `Py_DECREF`  C macro once you're done with an object.

### Code Overview

Our Go code is going to load and initialize a Python shared library so it can call a Python function that uses [numpy](https://numpy.org/) to do [outlier detection](https://en.wikipedia.org/wiki/Anomaly_detection) on a series of floating point values.

These are the steps that we will follow:
* Convert the Go `[]float64` parameter to a C `double *` (`outliers.go`)
* Create a numpy array from the C `double *` (`glue.c`)
* Call the Python function with the numpy array (`glue.c`)
* Get back a numpy array with indices of outliers (`glue.c`)
* Extract C `long *` from the numpy array (`glue.c`)
* Convert the C `long *` to Go `[]int` and return it from the Go function
  (`outliers.go`)

The Go code is in `outliers.go`, there's some C code in `glue.c` and finally the outlier detection Python function is in `outliers.py`. I’m not going to show the C code,  
If you're curious about it, have a look at [glue.c](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/glue.c).

**Listing 1: Example Usage**
```
15 	o, err := NewOutliers("outliers", "detect")
16 	if err != nil {
17 		return err
18 	}
19 	defer o.Close()
20 	indices, err := o.Detect(data)
21 	if err != nil {
22 		return err
23 	}
24 	fmt.Printf("outliers at: %v\n", indices)
25 	return nil
```
Listing 1 shows example usage.
On line 15 we create an `Outliers` object which uses the function `detect` from the `outliers` Python module. On line 19 we make sure to free the Python function. On line 20 we call the `Detect` method and get the indices of the outliers in the data.

### Code Highlights

**Listing 2: outliers.go [initialize](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/outliers.go#L25)**  
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

Listing 2 shows how we initialize Python for use in our Go program. On line 20, we declare a variable of type use [sync.Once](https://golang.org/pkg/sync/#Once) to make sure we initialize Python only once. On line 26, we call the `Do` method to call the initialization code and on line 28 we set the `initErr` variable to the last Python error.

**Listing 3: outliers.go [NewOutliers](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/outliers.go#L38)**  
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

Listing 3 shows creation of an `Outliers` object.  On line 38, we have the code to create a new `Outliers` struct. On lines 39-42, we make sure Python is initialized and there's no error. On line 44, we get a pointer to the Python function, same as doing an `import` statement in Python. On line 49 we save this Python pointer for later use in the `Outliers` struct.

**Listing 4: outliers.go [Detect](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/outliers.go#L53)**  
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

Listing 4 shows the code for `Outliers.Detect` method. On line On line 59 we convert Go `[]float64` to a C `double *`. On line 60 we call the Python function via the C layer and get back a result. On line 62 we tell Go's garbage collector that it can't reclaim the `data` slice until this line.  On lines 63-65 we check if there was an error calling Python. On lines 68,69 we fool the Go compiler to think that the C `double *` is a Go array. On lines 71,72 we copy the data from Python to a new slice. On line 74 we decrement the Python return value reference count.

**Listing 5: outliers.go [Outliers.Close](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/outliers.go#L80) method**
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

Listing 5 shows the 'Outliers.Close` method. On line 84 we decrement the Python function object reference count and on line 85 we set the `fn` field to `nil` to signal the `Outliers` object is closed.



### Building

The glue code is using header files from Python and numpy. In order to build we need to tell [cgo](https://golang.org/cmd/cgo/) where to find these header files. 

**Listing 6: outliers.go [cgo directives](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/outliers.go#L12)**
```
11 /*
12 #cgo pkg-config: python3
13 #cgo LDFLAGS: -lpython3.8
14 
15 #include "glue.h"
16 */
17 import "C"
```

Listing 6 shows the `cgo` directives.

On line 12 we use the [pkg-config](https://www.freedesktop.org/wiki/Software/pkg-config/) to find C compiler directives for Python. On line 13 we tell `cgo` to use the Python 3.8 shared library.  On line 15 we import the C code definitions from `glue.h` and on line 17 we have the `import "C"` directive that *must* come right after the comment.

numpy headers are a bit more tricky. numpy does not come with a `pkg-config` file but has a Python function that will tell you where the headers are. For security reasons, you can't have `cgo` run arbitrary commands. I opted to ask the user to set the `CGO_CFLAGS` environment variable before building or installing the package.

**Listing 7: Build commands**
```
01 $ export CGO_CFLAGS="-I $(python -c 'import numpy; print(numpy.get_include())'"
02 $ go build
```

Listing 7 shows how to build the package. One line 01 we set `CGO_CFLAGS` a value printed from a short Python program that prints the location of the numpy header files. On line 02 we build the package.

I like to use [make](https://www.gnu.org/software/make/) to automate such tasks. Have a look at the [Makefile](https://github.com/ardanlabs/python-go/blob/master/py-in-mem/Makefile) to learn more.

### Conclusion

I'd like to start by thanking the awesome people at the (aptly named) `#darkarts` channel in [Gophers Slack](https://gophers.slack.com/) for their help and insights.

The code we wrote here is risky and error prone, you should have some tight performance goals before going down this path. Benchmarking on my machine shows this code is about 45 times faster than the equivalent [gRPC code](https://www.ardanlabs.com/blog/2020/06/python-go-grpc.html) code. And even though I'm programming in Go for 10 years and in Python close to 25 - I learned some new things.


