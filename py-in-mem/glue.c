#include "glue.h"
#define NPY_NO_DEPRECATED_API NPY_1_19_API_VERSION 
#include <numpy/arrayobject.h>

// Return void * since import_array is a macro returning NULL
void *init_python() {
  Py_Initialize();
  import_array();
}

// Load function, same as "import module_name.func_name as obj" in Python
// Returns the function object or NULL if not found
PyObject *load_func(const char *module_name, char *func_name) {
  // Import the module
  PyObject *py_mod_name = PyUnicode_FromString(module_name);
  if (py_mod_name == NULL) {
    return NULL;
  }

  PyObject *module = PyImport_Import(py_mod_name);
  Py_DECREF(py_mod_name);
  if (module == NULL) {
    return NULL;
  }

  // Get function, same as "getattr(module, func_name)" in Python
  PyObject *func = PyObject_GetAttrString(module, func_name);
  Py_DECREF(module);
  return func;
}

// Call a function with array of values
result_t detect(PyObject *func, double *values, long size) {
  result_t res = {NULL, 0};

  // Create numpy array from values
  npy_intp dim[] = {size};
  PyObject *arr = PyArray_SimpleNewFromData(1, dim, NPY_DOUBLE, values);
  if (arr == NULL) {
    res.err = 1;
    return res;
  }

  // Construct function arguments
  PyObject *args = PyTuple_New(1);
  PyTuple_SetItem(args, 0, arr);

  PyArrayObject *out = (PyArrayObject *)PyObject_CallObject(func, args);
  if (out == NULL) {
    res.err = 1;
    return res;
  }

  res.obj = (PyObject *)out;
  res.size = PyArray_SIZE(out);
  res.indices = (long *)PyArray_GETPTR1(out, 0);
  return res;
}

// Return last error as char *, NULL if there was no error
const char *py_last_error() {
  PyObject *err = PyErr_Occurred();
  if (err == NULL) {
    return NULL;
  }

  PyObject *str = PyObject_Str(err);
  const char *utf8 = PyUnicode_AsUTF8(str);
  Py_DECREF(str);
  return utf8;
}

// Decrement reference counter for object. We can't use Py_DECREF directly from
// Go since it's a macro
void py_decref(PyObject *obj) { Py_DECREF(obj); }
