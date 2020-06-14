# Go ↔ Python: Part III Packaging Python Code

### Introduction

In [the previous post][FIXME] compiled Go code to a shared library and used it from Python. In this post we're going to finish the development process by writing a Python module that hides the low level details of working with a shared library and then package this code as a Python package. 

### Python Module

After we verified that out code works in the Python interactive prompt (aka [REPL](https://en.wikipedia.org/wiki/Read%E2%80%93eval%E2%80%93print_loop)), we can write a module. This module will have a Pythonic API and will hide the low level details of working with the shared library.

**Listing 1: checksig.py**
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

Listing 1 has the code of our Python module. The code in lines 14-19 is very much what we did in the interactive prompt.

Lines 7-10 deal with the shared library file name - we'll get to why we need that when we'll talk about packaging below. In lines 22-30 we define the API of our module - a single function called `check_signature`. ctypes will convert C's `NULL` to Python's `None`, hence the `if` statement in line 27. In line 30 we signal an error by raising a `ValueError` exception.

_Note: Python's naming conventions differ from Go. Most Python code is following the standard defined in [PEP-8](https://www.python.org/dev/peps/pep-0008/)._

### Packaging

One of the great things about Go is the packaging. Once you've built the executable - you can copy it over and it'll work™. In the Python world things are different, you need a Python interpreter on the machine running your code and install any external dependencies there as well.

What happens in Python when you have an extension library which is compiled to platform specific code? There are two options:
1. You can ship a platform specific package called a `wheel`
2. You can have the installer compile the Go code. Meaning the user will need a Go compiler on the machine running the application

Since this problem is common to any extension library, there's a standard way in Python that addresses both problems above. You write a `setup.py` which defines the Python project, and then generate a `wheel` distribution and a source (called `sdist`) distribution. If the user installs your package on a machine that matches a wheel architecture - Python's installer will install the binary package, otherwise the installer will download the source distribution and will try to compile it.

Python has built-in support for extensions written in C, C++ and SWIG, but not for Go. We will write our own command for building the Go extension.

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

In line 09 we define our command to build an extension that uses the Go compiler.  Lines 12-14 define the command to run and line 15 runs this command as an external command (Python's `subprocess` is like Go’s `os/exec`).

In line 20 we call the setup command, giving the package name in line 21 and a version in line 22. In line 23 we define the Python modules (without the `.py` extension) and in lines 24-26 we define the extension module (the Go code).  In line 27 we override the built-in 'build_ext` command with our `build_ext` command that builds Go code. In line 28 we specify the package is not zip safe since it contains shared libraries.

Another file we need to create is `MANIFEST.in`, it's a file that defines all the extra files that need to be packaged in a source distribution. 

**Listing 3: MANIFEST.in**
```
01 include README.md
02 include *.go go.mod go.sum
```

Listing 3 shows the extra files that should be packaged in source distribution (`sdist`). 

Now we can build

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

The wheel binary package (with `.whl` extension) has the platform information in its name: `cp38` means CPython version 3.8, `linux_x86_64` is the operation system and the architecture - same as Go's `GOOS` and `GOARCH`.

Now you can use Python's package manager, [pip](https://packaging.python.org/tutorials/installing-packages/) to install these packages. If you want to publish your package externally, you can upload these packages to the Python Package Index [PyPI](https://pypi.org/) using tools such as [twine](https://github.com/pypa/twine).

### Conclusion

With little effort, you hide the fact that you're using Go to extend Python and expose a Python module that has a Pythonic API. Packaging is what makes your code deployable and valuable, don't skip this step.

If you'd like to return Python types from Go (say a `list` or a `dict`), you can use Python's [extensive C API](https://docs.python.org/3/extending/index.html) using cgo. You can have a look at the [go-python](https://github.com/sbinet/go-python) that can ease a lot of the pain of writing Python extensions in Go.

In the next installment we're going to flip the roles again, we'll call Python from Go - in the same memory space and with almost zero serialization.
