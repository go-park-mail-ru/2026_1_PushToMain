package main

import (
	"flag"
	"log"

	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/app"
)

// @title           Smail API
// @version         1.0
// @host            localhost:8083
// @BasePath        /
func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/email/config.yaml", "path to config file")
	flag.Parse()

	application := app.New(configPath)
	if application == nil {
		log.Fatal("invalid config")
	}
	application.Run(configPath)
}
