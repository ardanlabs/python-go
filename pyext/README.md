# checksig - Calling Go from Python

**WIP: Ignore for now**

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
