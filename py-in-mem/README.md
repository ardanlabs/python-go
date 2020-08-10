# Go ↔ Python: Part IV Using Python in Memory

### Introduction

In [a previous
post](https://www.ardanlabs.com/blog/2020/06/python-go-grpc.html) we used
[gRPC](https://grpc.io/) to call Python code from Go. gRPC is a great framework
but there is a performance cost to it - every function call need to marshal the
arguments ([protobuf](https://developers.google.com/protocol-buffers), make a
network call ([HTTP/2](https://en.wikipedia.org/wiki/HTTP/2)) and then
unmarshal the result (`protobuf` again).

In this blog post we'll get rid of the network layer and, to some extent, the
marshalling. We'll do this by using [cgo](https://golang.org/cmd/cgo/) to
interact with Python as a shared library.

_Note: This blog is not for the weak of heart - we'll pass Go slices directly
to Python, use Python allocated memory directly in Go and fool the Go compiler
with ugly tricks. You better have some harsh performance requirements to follow
this path._

### Python Code

The Python code uses [numpy](https://numpy.org/) to do [outlier
detection](https://en.wikipedia.org/wiki/Anomaly_detection) on a series of
floating point values.

**Listing 1: outliers.py**
```
01 import numpy as np
02 
03 
04 def detect(data):
05     """Return indices where values more than 2 standard deviations from mean"""
06     out = np.where(np.abs(data - data.mean()) > 2 * data.std())
07     # np.where returns a tuple for each dimension, we want the 1st element
08     return out[0]
```

On line 04 we define a function that accepts and numpy `array`. On line 06 we use [boolean indexing](https://numpy.org/devdocs/user/basics.indexing.html#boolean-or-mask-index-arrays) to find all the values that are more than 2 [standard deviations](https://en.wikipedia.org/wiki/Standard_deviation) from the mean. On line 08 we return the list of indices of the outliers.

### Embedding Python Overview

Most of the time you'll use Python via the `python` interpreter, but Python can also be used as a [shared library](https://en.wikipedia.org/wiki/Library_(computing)#Shared_libraries). Python has an [extensive C API](https://docs.python.org/3/c-api/index.html) and [a lot of documentation](https://docs.python.org/3/extending/index.html) on how to extend and embed Python.

The Python C-API is for, well ..., C. We are going to use `cgo` to glue Go and Python together. Here are the steps we'll follow:

1. Initialize the Python interpreter
2. Load the Python function (`detect` in our case) and store it in a variable
3. Call the Python function with a `[]float64` values and get back `[]int` of indices

The embedding code involves some C code which is written in it's own file - `glue.c`.


**Figure  1**  
![](data-flow.png)

Figure 1 shows the flow of data from Go to Python and back.

### Handling Python Errors

### Initialization

To initialize Python & numpy, we need to call `Py_Initialize` from the Python
API and `import_array()` from numpy.


**Listing 2: gule.c `init_python`**
```
01 #include "glue.h"
02 #define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
03 #include <numpy/arrayobject.h>
04 
05 
06 // Return void * since import_array is a macro returning void *
07 void *init_python() {
08 	Py_Initialize();
09 	import_array();
10 }
```

Listing 2 show the C part of initialization. On line 01 we include `glue.h` that has function definitions and includes the `Python.h` header file. On line 03 we include the numpy header file and on lines 06-10 we initialize the code.


**Listing 3: outliers.go `initialize`**
```
01 package outliers
02 
03 import (
04 	"fmt"
05 	"runtime"
06 	"sync"
07 	"unsafe"
08 )
09 
10 /*
11 #cgo pkg-config: python3
12 #cgo LDFLAGS: -lpython3.8
13 
14 #include "glue.h"
15 
16 */
17 import "C"
18 
19 var (
20 	initOnce sync.Once
21 	initErr  error
22 )
23 
24 func initialize() {
25 	initOnce.Do(func() {
26 		C.init_python()
27 		initErr = pyLastError()
28 	})
29 }
```

Listing 3 shows the Go initialization code. On lines 10-17 we have `cgo` code.
On line 11 we use the [pkg-config](https://www.freedesktop.org/wiki/Software/pkg-config/) to find C compiler directives for Python. On line 12 we tell `cgo` to use the Python shared library.
In order to make sure initialization code runs ones, on line 20 we have a [sync.Once](https://golang.org/pkg/sync/#example_Once) variable. On line 24 we use `initOnce` to call the `init_python` function and on line 27 we record the initi
