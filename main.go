package main

import (
	"log"
	"net/http"
)

func main() {
	cfg, err := loadConfig(configPathFromEnv())
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("listening on %s", cfg.Server.Listen)
	if err := http.ListenAndServe(cfg.Server.Listen, newHandler(cfg)); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
