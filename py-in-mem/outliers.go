// outliers provides outlier detection via Python
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

// initialize Python & numpy, idempotent
func initialize() {
	initOnce.Do(func() {
		C.init_python()
		initErr = pyLastError()
	})
}

// Outliers does outlier detection
type Outliers struct {
	fn *C.PyObject
}

// NewOutliers returns an new Outliers using moduleName.funcName Python function
func NewOutliers(moduleName, funcName string) (*Outliers, error) {
	initialize()
	if initErr != nil {
		return nil, initErr
	}
	fn, err := loadPyFunc(moduleName, funcName)
	if err != nil {
		return nil, err
	}

	out := &Outliers{fn}
	runtime.SetFinalizer(out, func(o *Outliers) {
		C.py_decref(out.fn)
	})

	return out, nil
}

// Detect returns slice of outliers indices
func (o *Outliers) Detect(data []float64) ([]int, error) {
	// Convert []float64 to C double*
	carr := (*C.double)(&(data[0]))
	res := C.detect(o.fn, carr, (C.long)(len(data)))
	// Tell Go's GC to keep data alive until here
	runtime.KeepAlive(data)
	if res.err != 0 {
		return nil, pyLastError()
	}

	// Ugly hack to convert C.long* to []int
	ptr := unsafe.Pointer(res.indices)
	arr := (*[1 << 20]int)(ptr)
	// Create a copy managed by Go
	indices := make([]int, res.size)
	copy(indices, arr[:res.size])
	// Free Python object
	C.py_decref(res.obj)
	return indices, nil
}

// loadPyFunc loads a Python function by module and function name
func loadPyFunc(moduleName, funcName string) (*C.PyObject, error) {
	// Convert names to C char*
	cMod := C.CString(moduleName)
	cFunc := C.CString(funcName)

	// Free memory allocated by C.CString
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cFunc))
	}()

	fn := C.load_func(cMod, cFunc)
	if fn == nil {
		return nil, pyLastError()
	}

	return fn, nil
}

// Python last error
func pyLastError() error {
	cp := C.py_last_error()
	if cp == nil {
		return nil
	}

	err := C.GoString(cp)
	// We don't need to free cp, see
	// https://docs.python.org/3/c-api/unicode.html#c.PyUnicode_AsUTF8AndSize
	// which says: "The caller is not responsible for deallocating the buffer."
	return fmt.Errorf("%s", err)
}
