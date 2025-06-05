package main

import (
	"fmt"

	"github.com/phanitejak/kptgolib/sqlitemock"
)

func main() {
	_, err := sqlitemock.New(false)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Good!!!")
}
