package tid

import "strings"

const b32Sorted = "234567abcdefghijklmnopqrstuvwxyz"

func b32Encode(v uint64) string {
	v = (0x7FFF_FFFF_FFFF_FFFF & v)
	s := ""
	for range 13 {
		s = string(b32Sorted[v&0x1F]) + s
		v = v >> 5
	}
	return s
}

func b32Decode(s string) uint {
	var v uint = 0
	for n := range s {
		c := strings.IndexByte(b32Sorted, s[n])
		if c < 0 {
			return 0
		}
		v = (v << 5) | uint(c&0x1F)
	}
	return v
}
