package main

import "C"

//export verify
func verify(root *C.char) *C.char {
	rootDir := C.GoString(root)
	if err := CheckSignatures(rootDir); err != nil {
		return C.CString(err.Error())
	}

	return nil
}

func main() {}
