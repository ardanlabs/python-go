#include "glue.h"
#define NPY_NO_DEPRECATED_API NPY_1_7_API_VERSION
#include <numpy/arrayobject.h>


// Return void * since import_array is a macro returning void *
void *init_python() {
	Py_Initialize();
	import_array();
}


PyObject *load_func(const char *module_name, char *func_name) {
	PyObject *py_mod_name, *module;

	py_mod_name = PyUnicode_FromString(module_name);
	if (py_mod_name == NULL) {
		return NULL;
	}

	module = PyImport_Import(py_mod_name);
  Py_DECREF(py_mod_name);
	if (module == NULL) {
		return NULL;
	}

	PyObject *func = PyObject_GetAttrString(module, func_name);
  Py_DECREF(module);
	return func;
}

result_t detect(PyObject *func, double *values, long size) {
	result_t res = {NULL, 0};
	npy_intp dim[] = {size};
	PyObject *arr = PyArray_SimpleNewFromData(1, dim, NPY_DOUBLE, values);
	if (arr == NULL) {
		res.err = 1;
		return res;
	}
	PyObject *args = PyTuple_New(1);
	PyTuple_SetItem(args, 0, arr);
	PyArrayObject *out = (PyArrayObject *)PyObject_CallObject(func, args);
	if (out == NULL) {
		res.err = 1;
		return res;
	}


	res.size = PyArray_SIZE(out);
	res.indices = (long *)PyArray_GETPTR1(out, 0);
	return res;
}

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

void py_decref(PyObject *obj) {
	Py_DECREF(obj);
}
