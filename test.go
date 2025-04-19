package main

import "github.com/notjuliet/grove/tid"

func main() {
	var n int64 = 1234567890
	s, err := tid.Create(n, 0)
	if err != nil {
		panic(err)
	}
	println(n, "->", s)

	n, _, err = tid.Parse(s)
	if err != nil {
		panic(err)
	}
	println(s, "->", n)

	c := tid.NewClock(0)
	for range 10 {
		println(c.Now())
	}
}
