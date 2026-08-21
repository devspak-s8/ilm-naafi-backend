package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ilmnafi/backend/internal/config"
	"github.com/ilmnafi/backend/internal/database"
)

func main() {
	dsn := flag.String("dsn", "", "PostgreSQL DSN")
	file := flag.String("file", "", "Path to verified adhkar dataset JSON")
	flag.Parse()

	if *dsn == "" || *file == "" {
		fmt.Println("Usage: import-adhkar --dsn <postgres_url> --file <dataset.json>")
		os.Exit(1)
	}

	_ = config.Config{}
	_ = database.Connect

	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("failed to read dataset: %v", err)
	}

	var importData []struct {
		Category string `json:"category"`
		Arabic   string `json:"arabic"`
		Count    int    `json:"count"`
	}
	if err := json.Unmarshal(data, &importData); err != nil {
		log.Fatalf("failed to parse dataset: %v", err)
	}

	fmt.Printf("Loaded %d adhkar items from dataset\n", len(importData))
	fmt.Println("Import tool ready. Wire to DB when needed.")
}
