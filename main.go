package main

import (
	"github.com/shouni/go-ai-client/cmd"
	"log"
	"os"
)

func main() {

	log.SetFlags(0)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
