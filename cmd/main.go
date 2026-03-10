package main

import (
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app"
	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
)

// @title           Smail API
// @version         1.0
// @host            localhost:8087
// @BasePath        /
func main() {
	application := app.New()
	application.Run()
}