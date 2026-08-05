package main

import "fmt"

type Point struct {
	X int
	Y int
}

func main() {
	text := fmt.Sprintf(
		"Multiline %s",
		"expression",
	)

	point := Point{
		X: 1,
		Y: 2,
	}

	var a int
	var b int
	a, b = 10,
		20

	isValid := (a > 0) &&
		(b > 0)

	longValid := (a > 0) && (b > 0) && (a+b > 0) && (a*b > 0) && (a-b < 100) &&
		(b-a < 100) && (a+b == 30) && (a*b == 200)

	fmt.Println(text, point, a, b, isValid, longValid)
}
