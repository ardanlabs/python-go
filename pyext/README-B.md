# Go ↔ Python: Part III Packaging Python Code

### Introduction

In [the previous post](https://www.ardanlabs.com/blog/2020/07/extending-python-with-go.html) we compiled Go code to a shared library and used it from the Python interactive shell. In this post we're going to finish the development process by writing a Python module that hides the low level details of working with a shared library and then package this code as a Python package. 

### Recap - Architecture & Workflow Overview

**Figure  1**  
![](func-call.png)

Figure 1 shows the flow of data from Python to Go and back.

The workflow we're following is:

* Write Go code (`CheckSignature`), 
* Exporting to the shared library (`verify`)
* Use ctypes in the Python interactive prompt to call the Go code
* Write and package the Python code (`check_signatures`)

In the previous blog post we've done the first three parts and in this blog post we're going to implement the Python module and package it. We'll do this is the following steps:

* Write the Python module (`checksig.py`)
* Write the project definition file (`setup.py`)
* Build the extension

### Python Module

Let's start with writing a Python module. This module will have a Pythonic API and will hide the low level details of working with the shared library.

**Listing 1: checksig.py**
```
01 """Parallel check of files digital signature"""
02 
03 import ctypes
04 from distutils.sysconfig import get_config_var
05 from pathlib import Path
06 
07 # Location of shared library
08 here = Path(__file__).absolute().parent
09 ext_suffix = get_config_var('EXT_SUFFIX')
10 so_file = here / ('_checksig' + ext_suffix)
11 
12 # Load functions from shared library set their signatures
13 so = ctypes.cdll.LoadLibrary(so_file)
14 verify = so.verify
15 verify.argtypes = [ctypes.c_char_p]
16 verify.restype = ctypes.c_void_p
17 free = so.free
18 free.argtypes = [ctypes.c_void_p]
19 
20 
21 def check_signatures(root_dir):
22     """Check (in parallel) digital signature of all files in root_dir.
23     We assume there's a sha1sum.txt file under root_dir
24     """
25     res = verify(root_dir.encode('utf-8'))
26     if res is not None:
27         msg = ctypes.string_at(res).decode('utf-8')
28         free(res)
29         raise ValueError(msg)
```

Listing 1 has the code of our Python module. The code in lines 12-18 is very much what we did in the interactive prompt.

Lines 7-10 deal with the shared library file name - we'll get to why we need that when we'll talk about packaging below. In lines 21-29 we define the API of our module - a single function called `check_signatures`. ctypes will convert C's `NULL` to Python's `None`, hence the `if` statement in line 26. In line 29 we signal an error by raising a `ValueError` exception.

_Note: Python's naming conventions differ from Go. Most Python code is following the standard defined in [PEP-8](https://www.python.org/dev/peps/pep-0008/)._

### Packaging

One of the great things about Go is the packaging. Once you've built the executable - you can copy it over and it'll work™. In the Python world things are different: You need a Python interpreter on the machine running your code, and also install any external dependencies.

What happens in Python when you have an extension library which is compiled to platform specific code? There are two options:

1. You can ship a platform specific package called a `wheel`
2. You can have the installer compile the Go code. Meaning the user will need a Go compiler on the machine running the application

_Note: In Go you also need to compile your code to match the target machine operating system and architecture. This is done by setting the `GOOS` and `GOARCH` environment variables._

In Python, the standard way to solve the above issues is to write a `setup.py` file (the equivalent of Go's `go.mod` file) which defines the Python package, and then generate a `wheel` distribution and a source (called `sdist`) distribution. If the user installs your package on a machine that matches a wheel architecture - Python's installer will use the binary package, otherwise the installer will download the source distribution and will try to compile it.

Python has built-in support for extensions written in C, C++ and [SWIG](http://www.swig.org/), but not for Go. We will write our own command for building the Go shared library.

**Listing 2: setup.py**
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

Listing 2 shows the `setup.py` file for our project. In line 09 we define a command to build an extension that uses the Go compiler.  Lines 12-14 define the command to run and line 15 runs this command as an external command (Python's `subprocess` is like Go’s `os/exec`).

In line 20 we call the setup command, specifying the package name in line 21 and a version in line 22. In line 23 we define the Python module name and in lines 24-26 we define the extension module (the Go code). In line 27 we override the built-in 'build_ext` command with our `build_ext` command that builds Go code. In line 28 we specify the package is not zip safe since it contains a shared library.

Another file we need to create is `MANIFEST.in`, it's a file that defines all the extra files that need to be packaged in a source distribution. 

**Listing 3: MANIFEST.in**
```
01 include README.md
02 include *.go go.mod go.sum
```

Listing 3 shows the extra files that should be packaged in source distribution (`sdist`). 

Now we can build the packages.

**Listing 4: Building the Packages**
```
$ python setup.py bdist_wheel
$ python setup.py sdist
```

Listing 4 shows the command to build a `wheel` and `sdist` package files.

The package are built in the `dist` directory

**Listing 5: Content of `dist` Directory**
```
$ ls dist
checksig-0.1.0-cp38-cp38-linux_x86_64.whl
checksig-0.1.0.tar.gz
```

In Listing 5 we use the `ls` command to show the content of the `dist` directory.

The wheel binary package (with `.whl` extension) has the platform information in its name: `cp38` means CPython version 3.8, `linux_x86_64` is the operation system and the architecture - same as Go's `GOOS` and `GOARCH`. Since the wheel file name changes depending on the architecture it’s built on, we had to write some logic in Listing 1 lines 08-10.

Now you can use Python's package manager, [pip](https://packaging.python.org/tutorials/installing-packages/) to install these packages. If you want to publish your package, you can upload it to the Python Package Index [PyPI](https://pypi.org/) using tools such as [twine](https://github.com/pypa/twine).

See `example.py and `Dockerfile.test-b` in the [source repository](https://github.com/ardanlabs/python-go/tree/master/pyext) for a full build, install & use flow.

### Conclusion

With little effort, you can extend Python using Go and expose a Python module that has a Pythonic API. Packaging is what makes your code deployable and valuable, don't skip this step.

If you'd like to return Python types from Go (say a `list` or a `dict`), you can use Python's [extensive C API](https://docs.python.org/3/extending/index.html) with cgo. You can also have a look at the [go-python](https://github.com/sbinet/go-python) that can ease a lot of the pain of writing Python extensions in Go.

In the next installment we're going to flip the roles again, we'll call Python from Go - in the same memory space and with almost zero serialization.
