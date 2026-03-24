package main

import (
	"flag"
	"log"

	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app"
)

// @title           Smail API
// @version         1.0
// @host            localhost:8087
// @BasePath        /
func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/config.yaml", "path to config file")
	flag.Parse()

	application := app.New(configPath)
	if application == nil {
		log.Fatal("invalid config")
	}
	application.Run(configPath)
}
