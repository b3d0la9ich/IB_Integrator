package main

import (
	"fmt"
	"log"

	"ib-integrator/internal/config"
	"ib-integrator/internal/database"
	"ib-integrator/internal/server"
)

func main() {
	cfg := config.Load()
	database.Init(cfg.DBDSN)

	r := server.NewRouter(cfg)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
