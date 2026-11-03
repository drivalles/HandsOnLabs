package main

import (
	"fmt"
)

func main() {
	Welcome()
	SerieFibonacci()
}
func Welcome() {
	fmt.Println("Hello World")

	var name string = "Adrian"
	var lastname string = "del Valle"
	var age string = "43"
	var gender string = "M"

	fmt.Println("Name: " + name + "\n" +
		"Lastname " + lastname + "\n" +
		"Age: " + age + "\n" +
		"Gender: " + gender + "\n")
}

func SerieFibonacci() {

	var sum []int

	for i := 0; i < 15; i++ {
		if i == 0 || i == 1 {
			sum = append(sum, i)
		} else {
			sum = append(sum, sum[i-1]+sum[i-2])
		}
	}

	// for i := 0; i < len(sum); i++ {
	// 	fmt.Println(strconv.Itoa(sum[i]) + " ")
	// }
	// fmt.Printf("sum: %v\n", sum)

	fmt.Printf("SerieFibonacci %v\n", sum)
}
