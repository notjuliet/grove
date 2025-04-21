package tid

import "strings"

const b32Sorted = "234567abcdefghijklmnopqrstuvwxyz"

func b32Encode(i int64) string {
	s := ""

	for i != 0 {
		c := i % 32
		i = i / 32
		s = b32Sorted[c:c+1] + s
	}

	return s
}

func b32Decode(s string) int64 {
	var i int64 = 0

	for _, c := range s {
		i = i*32 + int64(strings.IndexRune(b32Sorted, c))
	}

	return i
}
