package id

import (
	"fmt"
	"hash/crc32"
)

// GenID generates an ID according to the raw material.
func GenID(raw string) string {
	if raw == "" {
		return ""
	}
	res := crc32.ChecksumIEEE([]byte(raw))
	return fmt.Sprintf("%x", res)
}
