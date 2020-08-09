#ifndef GLUE_H
#define GLUE_H

#include <Python.h>


void init_python();
PyObject *load_func(const char *module_name, char *func_name);
int *detect(PyObject *func, double *values, long size);

#endif // GLUE_H
