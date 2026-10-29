package main

import "fmt"

func main() {
	fmt.Println("Testing My First Code in GoLang")

	var name string = "Adrián"
	var lastname string = "del Valle Matos"
	var age string = "43"
	var gender string = "M"

	fmt.Println("Name: " + name + "\n" +
		"Lastname " + lastname + "\n" +
		"Age: " + age + "\n" +
		"Gender: " + gender + "\n")
}

func method() {
	fmt.Println("This is the line writted in Go code.")

	var name string = "George"
	var lastname string = "Washington"
	var age string = "58860"
	var gender string = "M"

	fmt.Println("Name: " + name + "\n" +
		"Lastname " + lastname + "\n" +
		"Age: " + age + "\n" +
		"Gender: " + gender + "\n")
}
