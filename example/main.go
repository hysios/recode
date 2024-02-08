package main

import (
	"fmt"

	"github.com/hysios/recode/example/model"
)

//go:generate recode -label RECODE-MODEL -input ../input.txt -row "&{{ . }}"
func main() {
	// Print all models
	fmt.Printf("%#v\n", []interface{}{
		// RECODE-MODEL-BEGIN
		&model.User{},
		&model.Friend{},
		// RECODE-MODEL-END
	})
	fmt.Println("Hello World!")
}
