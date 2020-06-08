# Go ↔ Python: Part II Writing Python Extension Library in Go

### Introduction

In [the previous post][FIXME] we saw how a Go process can call a Python process using [gRPC](https://grpc.io/). In this post we're going to flip the roles, Python is going to call a Go function.

Python has a long tradition of writing extension modules, mostly in C, to speed up some parts of the code. There are several way of writing extension modules to Python, from the built-in [API](https://docs.python.org/3/extending/index.html) to frameworks such as [SWIG](http://www.swig.org/), [pybind11](https://pybind11.readthedocs.io/) and others.

Instead of writing a Python extension, that involves a lot of boilerplate code, we're going to take a simpler approach. We'll build a shared library from the Go code and then use Python's ability to call functions from shared libraries using the [ctypes](https://docs.python.org/3/library/ctypes.html) module.

_Note: ctypes uses [libffi](https://github.com/libffi/libffi) under the hood. If you want to read some really scary C code - head over to the repo and start reading. :)_

### Example: Parallel Check of Files Digital Signature

Say you have a process that downloads a directory with data files. The directory also contains a `sha1sum.txt` file with a [sha1](https://en.wikipedia.org/wiki/SHA-1) digital signature for every file.

Here's an example file

**Listing 1: sha1sum.txt**
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

You'd like to verify that all the files in the library are valid by matching their digital signature with the one in the file. Files can be damaged while downloaded or changed by malicious code. To speed this process, we'll calculate the digital signature of each file in a separate goroutine, spreading the work on all of the CPUs on our machine.

### Go Code

I'm not going to show the Go code here, if you're curious - see [here](https://github.com/ardanlabs/python-go/blob/master/pyext/checksig.go).

The main point is that you write normal Go code, and in a different file we'll have the code exposing the code in `checksig.go` to a shared library. Writing regular Go code allows you to test it as usual, use linters (such as [golangci-lint](https://github.com/golangci/golangci-lint)) ...

The function we're going to expose is `CheckSignature`, here's how it's defined:

**Listing 2** CheckSignatures function definition
```
func CheckSignatures(rootDir string) error {
```

### Exporting To Shared Library

To expose Go code to shared library, you need to:
- import the `C` package (aka [cgo](https://golang.org/cmd/cgo/))
- Use `//export` directives on every function you want to expose
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
- In line 03 we import the "C" library.
- In line 05 we mark `verify` as exported using the `//export` comment. *Don't*
  add a space between the `//` to the `export`
- `verify` get a `*C.char` as parameter and returns `*C.char`. This is C's string
- In line 07 we use `C.GoString` to convert the C string to a Go string
- In line 13 we use `C.CString` to convert the error value to a C string. This
  is going to allocate memory for the string and will be cleaned by Python
- In line 17 we have an empty `main`, this is required

For now, let's build the extension manually.

**Listing 3: Building the Shared Library**
```
$ go build -buildmode=c-shared -o _checksig.so
```

Listing 3 shows the command to generate the shred library `_checksig.so`.

_Note: The reason for using `_` is to avoid name collision with `checksig.py` Python module that we'll show later. When you run `import checksig`, Python will first try to load the shared library and not the Python file._

And now we can try calling `verify` from Python. In the code repository you'll find [logs.tar.bz2](https://github.com/ardanlabs/python-go/blob/master/pyext/logs.tar.bz2) which contains some log files and a `sha1sum.txt` file. **The signature for `http08.log` is intentionally wrong.**

I've extracted the archive to `/tmp/logs`, if you want to do the same - run the command:

```
$ tar xjf logs.tar.bz2 -C /tmp
```

I love the interactive shell in Python, it lets me play around with code and when I have a working version I write the code in a file.

**Listing 4: Python Session**
```
01 >>> import ctypes
02 >>> so = ctypes.cdll.LoadLibrary('./_checksig.so')
03 >>> verify = so.verify
04 >>> verify.argtypes = [ctypes.c_char_p]
05 >>> verify.restype = ctypes.c_char_p
06 >>> out = verify('/tmp/logs'.encode('utf-8'))
07 >>> print(out.decode('utf-8'))
08 "/tmp/logs/httpd-08.log" - mismatch
```

The `>>> ` prefix comes from the Python interactive shell prompt.

Let's go over line by line:
- 01: Import the `ctypes` module
- 02: Load the shared library
- 03: Assign a variable to the `verify` function
- 04, 05: Tell ctypes the parameter types and return type of `verify`. Shared
  libraries function are just a name, you should know beforehand what are the
  parameters - usually by including a C header file
- 06: Call verify. Since we're dealing with C we convert the `str` (Python's
  string) parameter to `bytes` which is a C `char *`
- 07: Print the output. The return type from C is `char *`, or `bytes` in
  Python, we use `decode` to convert it back to `str`

Very nice, with very little effort we're able to call Go code from C.

### Sharing Memory

Both Python & Go have a garbage collector, meaning they will automatically free any memory that is unused. However having a garbage collector doesn't mean you can't leak memory.

_Note: You should read Bill's [Garbage Collection In Go](https://www.ardanlabs.com/blog/2018/12/garbage-collection-in-go-part1-semantics.html) blog posts. They will give you a good understanding on garbage collectors in general and on the Go garbage collector specifically._

You need to be extra careful when sharing memory between Go and Python (or C).  Sometimes it's not clear when a memory allocation happens. For example, in `export.go` line 13 we have the following code:

```
str := C.CString(err.Error())
```

If you read `C.String` [documentation](https://golang.org/cmd/cgo/) you'll see the following (my emphasis):

> // Go string to C string
> // The C string is allocated in the C heap using malloc.
> // **It is the caller's responsibility to arrange for it to be
> // freed**, such as by calling C.free (be sure to include stdlib.h
> // if C.free is needed).
> func C.CString(string) *C.char

In Listing 4 we have a memory leak! Line 05 `verify.restype = ctypes.c_char_p` make the return value of `verify` a `bytes`, however under the hood Python will allocate memory for this `bytes` object and copy over the data allocated by Go. We don't have a reference to the memory allocated by Go and can't free it.

Let's get back to our Python session and fix this memory leak.

**Listing 5: Python Session**

```
01 >>> verify.restype = ctypes.c_void_p
02 >>> libc = ctypes.cdll.LoadLibrary('libc.so.6')
03 >>> free = libc.free
04 >>> free.argtypes = [ctypes.c_void_p]
05 >>> ptr = verify('/tmp/logs'.encode('utf-8'))
06 >>> out = ctypes.string_at(ptr)
07 >>> free(ptr)
08 >>> print(out.decode('utf-8'))
09 "/tmp/logs/httpd-08.log" - mismatch
```

In line 01 we change the result type of `verify` to C's `void *`. In lines 03 and 04 we load the `free` function from C's standard library (on windows change line 03 to `libc = cdll.msvcrt`). In line 05 we get the pointer to the allocated memory and in line 06 we convert it to bytes. Finally, in line 07 we release the memory allocated by Go and then continue as before.

### Python Module

Once we saw how the code works in the Python interactive prompt (aka [REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop)), we can write a module to wrap the shared library.

**Listing 6: checksig.py**
```
01 """Parallel check of files digital signature"""
02 
03 import ctypes
04 from pathlib import Path
05 from distutils.sysconfig import get_config_var
06 
07 # Find out where the shared library is at
08 ext_suffix = get_config_var('EXT_SUFFIX')
09 here = Path(__file__).absolute().parent
10 so_file = here / ('_checksig' + ext_suffix)
11 
12 
13 # Load function and set its signature
14 so = ctypes.cdll.LoadLibrary(so_file)
15 verify = so.verify
16 verify.argtypes = [ctypes.c_char_p]
17 verify.restype = ctypes.c_void_p
18 free = so.free
19 free.argtypes = [ctypes.c_void_p]
20 
21 
22 def check_signature(root_dir):
23     """Check (in parallel) digital signature of all files in root_dir.
24     We assume there's a sha1sum.txt file under root_dir
25     """
26     res = verify(root_dir.encode('utf-8'))
27     if res is not None:
28         msg = ctypes.string_at(res).decode('utf-8')
29         free(res)
30         raise ValueError(msg)
```

The code in lines 14-19 is very much what we did in the interactive prompt.

Lines 7-10 deal with the shared library name - we'll get to why we do that when we'll talk about packaging below. In lines 22-30 we define the API of out module - a single function called `check_signature`. ctypes will convert C's `NULL` to Python's `None`, hence the `if` statement in line 27.

_Note: Python's naming conventions differ from Go. Most Python code is following the standard defined in [PEP-8](https://www.python.org/dev/peps/pep-0008/)._

### Packaging

One of the great things about Go is the packaging. Once you've built the executable - you can copy it over and it'll work™. In the Python world things are different, you need a Python interpreter on the machine running your code and install external dependencies there as well.

What happens in Python when you have an extension library which is compiled to platform specific code? There are two options:
1. You can ship a platform specific package called a `wheel`
2. You can have the installer compile the Go code. Meaning the user will need a Go compiler on the machine running the application

Since this problem is common to any extension library, there's a standard way in Python that addresses both problems above. You write a `setup.py` which defines the Python project, and then generate a `wheel` distribution and a source (called `sdist`) distribution. If the user installs your package on a machine that matches a wheel architecture - they'll get the binary package, otherwise Python will download the source distribution and will try to compile it.

Python has built-in support for extensions written in C, C++ and SWIG. We'll have to write our own command for building the Go extension.

**Listing 7: setup.py**
```
01 """Setup for checksig package"""
02 from distutils.errors import CompileError
03 from subprocess import call
04 
05 from setuptools import Extension, setup
06 from setuptools.command.build_ext import build_ext
07 
08 
09 class build_go_ext(build_ext):
10     """Custom command to build extension from Go source files"""
11     def build_extension(self, ext):
12         ext_path = self.get_ext_fullpath(ext.name)
13         cmd = ['go', 'build', '-buildmode=c-shared', '-o', ext_path]
14         cmd += ext.sources
15         out = call(cmd)
16         if out != 0:
17             raise CompileError('Go build failed')
18 
19 
20 setup(
21     name='checksig',
22     version='0.1.0',
23     py_modules=['checksig'],
24     ext_modules=[
25         Extension('_checksig', ['checksig.go', 'export.go'])
26     ],
27     cmdclass={'build_ext': build_go_ext},
28     zip_safe=False,
29 )
```

In line 09 we define our command to build an extension that uses the Go compiler.  Lines 12-14 defines the command to run and line 15 runs this command as an external command (Python's `subprocess` is like Go’s `os/exec`).

In line 20 we call the setup command, giving the package name in line 21 and a version in line 22. In line 23 we define the Python modules (without the `.py` extension) and in lines 24-26 we define the extension module (the Go code).  In line 27 we override the built-in 'build_ext` command with our `build_ext` command that builds Go code. In line 28 we specify the package is not zip safe since it contains shared libraries.

Another file we need to create is `MANIFEST.in`, it's a file that defines all the extra files that need to be packaged in a source distribution. 

**Listing 8: MANIFEST.in**
```
01 include README.md
02 include *.go go.mod go.sum
```

We include the README and all the Go related files.

Now we can build

**Listing 9: Building the Packages**
```
$ python setup.py bdist_wheel
$ python setup.py sdist
```

The package are built in the `dist` directory

**Listing 10: Content of `dist` Directory**
```
$ ls dist
checksig-0.1.0-cp38-cp38-linux_x86_64.whl
checksig-0.1.0.tar.gz
```

The wheel binary package (with `.whl` extension) has the platform information in its name: `cp38` means CPython version 3.8, `linux_x86_64` is the operation system and the architecture - like Go's `GOOS` and `GOARCH`.

Now you can use Python's package manager, [pip](https://packaging.python.org/tutorials/installing-packages/) to install these packages. You can also upload these packages to Python package index [PyPI](https://pypi.org/) using tools such as [twine](https://github.com/pypa/twine), then people will be all to run `pip install checksig` and use your package.

### Conclusion

With very little code, you can use Go from Python. Unlike the previous installment, there's no RPC step - meaning you don't need to marshal and unmarshal parameters on every function call and there's no network involved as well. On the other hand you need to be more careful with memory management and the packaging process is more complex.

If you'd like to return Python types from Go (say a `list` or a `dict`), you can use Python's [extensive C API](https://docs.python.org/3/extending/index.html) using cgo. You can have a look at the [go-python](https://github.com/sbinet/go-python) that can ease a lot of the pain of writing Python extensions in Go.

In the next installment we're going to flip the roles again, we'll call Python from Go - but this time, without RPC.
