package id

import (
	"fmt"
	"hash/crc32"
	"reflect"
	"unsafe"
)

// GenID generates an ID according to the raw material.
func GenID(raw string) string {
	if raw == "" {
		return ""
	}
	sh := &reflect.SliceHeader{
		Data: (*reflect.StringHeader)(unsafe.Pointer(&raw)).Data,
		Len:  len(raw),
		Cap:  len(raw),
	}
	p := *(*[]byte)(unsafe.Pointer(sh))

	res := crc32.ChecksumIEEE(p)
	return fmt.Sprintf("%x", res)
}
