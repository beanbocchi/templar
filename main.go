package main

import (
	"log"

	"github.com/beanbocchi/templar/internal"
)

func main() {
	if err := internal.Start(); err != nil {
		log.Panicf("failed to start server: %v", err)
	}

	select {}
}
