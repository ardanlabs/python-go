package main

import (
	"math/rand"
	"os"
	"unsafe"
)

/*
#cgo pkg-config: python3
#cgo LDFLAGS: -lpython3.8
#cgo CFLAGS: -I/home/miki/.venv/lib/python3.8/site-packages/numpy/core/include

#include "glue.h"

*/
import "C"

func main() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	os.Setenv("PYTHONPATH", pwd)
	C.init_python()

	pyFunc := loadPyFunc("outliers", "detect")
	const size = 1000
	arr := make([]float64, size)
	for i := 0; i < size; i++ {
		arr[i] = rand.Float64()
	}

	arr[7] = 97.3
	arr[113] = 92.1
	arr[835] = 93.2

	ca := (*C.double)(&(arr[0]))
	C.detect(pyFunc, ca, (C.long)(len(arr)))
}

func loadPyFunc(moduleName, funcName string) *C.PyObject {
	cMod := C.CString(moduleName)
	cFunc := C.CString(funcName)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cFunc))
	}()

	return C.load_func(cMod, cFunc)
}
