package main

import (
	"smail/internal/app"
	"fmt"
	_ "smail/docs"
)

// @title           Smail API
// @version         1.0
// @host            localhost:8087
// @BasePath        /
func main() {
	application := app.New()
	application.Run()
}