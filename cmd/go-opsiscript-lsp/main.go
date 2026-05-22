package main

import (
	"log"
	"os"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/lsp"
)

func main() {
	logger := log.New(os.Stderr, "go-opsiscript-lsp: ", log.LstdFlags|log.Lshortfile)
	server := lsp.NewServer(os.Stdin, os.Stdout, logger)
	if err := server.Run(); err != nil {
		logger.Fatalf("server exited: %v", err)
	}
}
