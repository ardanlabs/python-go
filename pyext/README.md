# Go ↔ Python: Part II Writing Python Extension Library in Go

**WIP: Ignore for now**

### Introduction

In [the previous post][FIXME] we saw how a Go process can call a Python process
using [gRPC][https://grpc.io/]. In this post we're going to flip the roles,
Python is going to call a Go function.

Python has a long tradition of writing extension modules, mostly in C, to speed
up some parts of the code. There are several way of writing extension modules
to Python, from the built-in
[API][https://docs.python.org/3/extending/index.html] to frameworks such as
[SWIG](http://www.swig.org/),
[pybind11](https://pybind11.readthedocs.io/en/stable/) and others.

We're going to take a simpler approach, and use Python's ability to call
function from shared libraries using the
[ctypes](https://docs.python.org/3/library/ctypes.html) module.

_Note: ctypes uses [libffi](https://github.com/libffi/libffi) under the hood.
If you want to read so really scary C code - head over and start reading._

We'll create a shared library from the Go code using the `-buildmode=c-shared`
build flag.

### Example: Parallel Check of Files Digital Signature

You have a process that downloads a directory with data files. The directory
also contains a `sha1sum.txt` file with a
[sha1](https://en.wikipedia.org/wiki/SHA-1) digital signature for every file.

Here's an example file

**Listing 1**
```
6659cb84ab403dc85962fc77b9156924bbbaab2c  httpd-00.log
5693325790ee53629d6ed3264760c4463a3615ee  httpd-01.log
fce486edf5251951c7b92a3d5098ea6400bfd63f  httpd-02.log
b5b04eb809e9c737dbb5de76576019e9db1958fd  httpd-03.log
ff0e3f644371d0fbce954dace6f678f9f77c3e08  httpd-04.log
c154b2aa27122c07da77b85165036906fb6cbc3c  httpd-05.log
28fccd72fb6fe88e1665a15df397c1d207de94ef  httpd-06.log
86ed10cd87ac6f9fb62f6c29e82365c614089ae8  httpd-07.log
feaf526473cb2887781f4904bd26f021a91ee9eb  httpd-08.log
330d03af58919dd12b32804d9742b55c7ed16038  httpd-09.log
```

You'd like to verify that all the files in the library are valid by matching
their digital signature with the one in the file. File can be damaged while
downloaded or changed by malicious code.


### Go Code

I'm not going to show the Go code here, if you're curious - see
[here](https://github.com/ardanlabs/python-go/blob/master/pyext/checksig.go).

The main point is that you write normal Go code, in a different file we'll have
the code exposing the code in `checksig.go` to a shared library. Since you
write regular Go code, you can test it as usual, use linters (such as
[golangci-lint](https://github.com/golangci/golangci-lint)) ...

The function we're going to expose is `CheckSignature`, here's how it's defined:

**Listing 2** CheckSignatures function definition
```
func CheckSignatures(rootDir string) error {
```

### Exporting To Shared Library

To expose Go code to shared library, you need 4 things:
- import the `C` package (aka [cgo](https://golang.org/cmd/cgo/))
- Use //export directives on every function you want to expose
- Have an empty `main`
- Build with `-buildmode=c-shared` flag

**Listing 3** export.go
```
01 package main
02 
03 import "C"
04 
05 //export verify
06 func verify(root *C.char) *C.char {
07 	rootDir := C.GoString(root)
08 	err := CheckSignatures(rootDir)
09 	if err == nil {
10 		return nil
11 	}
12 
13 	str := C.CString(err.Error())
14 	return str
15 }
16 
17 func main() {}
```

Highlights:
- In line 03 we importing the "C" library.
- In line 05 we mark `verify` as exported using the `//export` comment. *Don't*
  add a space between `//` to `export`
- `verify` get a `*C.char` as parameter and returns `*C.char`. This is C's string
- In line 07 we use `C.GoString` to convert the C string to a Go string
- In line 13 we use `C.CString` to convert the error value to a C string. This
  is going to allocate memory for the string and will be cleaned by Python
- In line 17 we have an empty `main`, this is required
