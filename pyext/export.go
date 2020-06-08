package main

import "C"

//export verify
func verify(root *C.char) *C.char {
	rootDir := C.GoString(root)
	err := CheckSignatures(rootDir)
	if err == nil {
		return nil
	}

	str := C.CString(err.Error())
	return str
}

func main() {}
