#ifndef GLUE_H
#define GLUE_H

#include <Python.h>

typedef struct {
	long *indices;
	long size;
	int err;
} result_t;

void *init_python();
PyObject *load_func(const char *module_name, char *func_name);
result_t detect(PyObject *func, double *values, long size);
const char *py_last_error();
void py_decref(PyObject *obj);

#endif // GLUE_H
