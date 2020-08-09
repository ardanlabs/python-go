#include "glue.h"
#define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
#include <numpy/arrayobject.h>


void init_python() {
	Py_Initialize();
	import_array();
}


PyObject *load_func(const char *module_name, char *func_name) {
	PyObject *py_mod_name, *module;

	py_mod_name = PyUnicode_FromString(module_name);
	if (py_mod_name == NULL) {
		printf("ERROR: MODULE NAME\n");
		return NULL;
	}

	module = PyImport_Import(py_mod_name);
  Py_DECREF(py_mod_name);
	if (module == NULL) {
		printf("ERROR: IMPORT\n");
		return NULL;
	}

	PyObject *func = PyObject_GetAttrString(module, func_name);
  Py_DECREF(module);
	return func;
}

int *detect(PyObject *func, double *values, long size) {
	npy_intp dim[] = {size};
	PyObject *arr = PyArray_SimpleNewFromData(1, dim, NPY_DOUBLE, values);
	if (arr == NULL) {
		printf("<ERROR> PyArray_SimpleNewFromData\n");
		return NULL;
	}
	PyObject *args = PyTuple_New(1);
	PyTuple_SetItem(args, 0, arr);
	PyArrayObject *out = (PyArrayObject *)PyObject_CallObject(func, args);
	if (out == NULL) {
		printf("<ERROR> calling function");
		return NULL;
	}

	long osize = PyArray_SIZE(out);

	printf("size = %d\n", osize);
	long *ptr = (long *)PyArray_GETPTR1(out, 0);
	for (long i = 0; i < osize; i++) {
		printf("%d\n", ptr[i]);
	}
}
