package main

import (
	"flag"

	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app"
)

// @title           Smail API
// @version         1.0
// @host            localhost:8087
// @BasePath        /
func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	if *configPath == "" {
		*configPath = "config.yaml"
	}
	application := app.New()
	application.Run(*configPath)
}
