package main

import (
	"fmt"

	"github.com/marcosarl1/Circuito-API/internal/config"
)

func main() {
	cfg := config.Load()
	fmt.Printf("port=%s db=%s coll=%s\n", cfg.Port, cfg.MongoDB, cfg.MongoCollection)
}
