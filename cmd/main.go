package main

import (
	"smail/internal/app"
	"fmt"
)

func main() {
	application := app.New()
	if err := application.Run(); err != nil {
		fmt.Println(err)
	}
}