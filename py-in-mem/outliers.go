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
	fn *C.PyObject // Outlier detection Python function object
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

	return &Outliers{fn}, nil
}

// Detect returns slice of outliers indices
func (o *Outliers) Detect(data []float64) ([]int, error) {
	if o.fn == nil {
		return nil, fmt.Errorf("closed")
	}

	if len(data) == 0 { // Short path
		return nil, nil
	}

	// Convert []float64 to C double*
	carr := (*C.double)(&(data[0]))
	res := C.detect(o.fn, carr, (C.long)(len(data)))

	// Tell Go's GC to keep data alive until here
	runtime.KeepAlive(data)
	if res.err != 0 {
		return nil, pyLastError()
	}

	// Create a Go slice from C long*
	indices, err := cArrToSlice(res.indices, res.size)
	if err != nil {
		return nil, err
	}

	// Free Python array object
	C.py_decref(res.obj)
	return indices, nil
}

// Close frees the underlying Python function
// You can't use the object after closing it
func (o *Outliers) Close() {
	if o.fn == nil {
		return
	}
	C.py_decref(o.fn)
	o.fn = nil
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

// Create a new []int from a *C.long
func cArrToSlice(cArr *C.long, size C.long) ([]int, error) {
	const maxSize = 1 << 20
	if size > maxSize {
		return nil, fmt.Errorf("C array to large (%d > %d)", size, maxSize)
	}

	// Ugly hack to convert C.long* to []int - make the compiler think there's
	// a Go array at the C array memory location
	ptr := unsafe.Pointer(cArr)
	arr := (*[maxSize]int)(ptr)

	// Create a slice with copy of data managed by Go
	s := make([]int, size)
	copy(s, arr[:size])

	return s, nil
}
