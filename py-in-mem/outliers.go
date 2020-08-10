package outliers

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

/*
#cgo pkg-config: python3
#cgo LDFLAGS: -lpython3.8

#include "glue.h"

*/
import "C"

var (
	initOnce sync.Once
	initErr  error
)

func initialize() {
	initOnce.Do(func() {
		C.init_python()
		initErr = pyLastError()
	})
}

type Outliers struct {
	pyFunc *C.PyObject
}

func NewOutliers(moduleName, funcName string) (*Outliers, error) {
	initialize()
	if initErr != nil {
		return nil, initErr
	}
	pyFunc, err := loadPyFunc(moduleName, funcName)
	if err != nil {
		return nil, err
	}

	out := &Outliers{pyFunc}
	runtime.SetFinalizer(out, func(o *Outliers) {
		C.py_decref(out.pyFunc)
	})

	return out, nil
}

func (o *Outliers) Detect(data []float64) ([]int, error) {
	carr := (*C.double)(&(data[0]))
	res := C.detect(o.pyFunc, carr, (C.long)(len(data)))
	runtime.KeepAlive(data)
	if res.err != 0 {
		return nil, pyLastError()
	}

	// Convert C int* to []int
	indices := make([]int, res.size)
	ptr := unsafe.Pointer(res.indices)
	// Ugly hack to convert C.long* to []C.long
	cArr := (*[1 << 20]C.long)(ptr)
	for i := 0; i < len(indices); i++ {
		indices[i] = (int)(cArr[i])
	}
	C.free(ptr)
	return indices, nil
}

func loadPyFunc(moduleName, funcName string) (*C.PyObject, error) {
	cMod := C.CString(moduleName)
	cFunc := C.CString(funcName)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cFunc))
	}()

	pyFunc := C.load_func(cMod, cFunc)
	if pyFunc == nil {
		return nil, pyLastError()
	}

	return pyFunc, nil
}

func pyLastError() error {
	cp := C.py_last_error()
	if cp == nil {
		return nil
	}

	err := C.GoString(cp)
	// C.free(unsafe.Pointer(cp))
	return fmt.Errorf("%s", err)
}
