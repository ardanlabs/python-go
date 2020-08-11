#ifndef GLUE_H
#define GLUE_H

#include <Python.h>

// Result of calling detect
typedef struct {
  PyObject *obj; // numpy array object, so we can free it
  long *indices; // indices of outliers
  long size;     // number of outliers
  int err;       // Flag if there was an error
} result_t;

void *init_python();
PyObject *load_func(const char *module_name, char *func_name);
result_t detect(PyObject *func, double *values, long size);
const char *py_last_error();
void py_decref(PyObject *obj);

#endif // GLUE_H
