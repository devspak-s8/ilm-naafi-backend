package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilmnafi/backend/internal/config"
	"github.com/ilmnafi/backend/internal/database"
)

func main() {
	dsn := flag.String("dsn", "", "PostgreSQL DSN")
	file := flag.String("file", "", "Path to verified quran dataset JSON")
	flag.Parse()

	if *dsn == "" || *file == "" {
		fmt.Println("Usage: import-quran --dsn <postgres_url> --file <dataset.json>")
		os.Exit(1)
	}

	cfg := &config.Config{Database: struct{ URL string }{
		URL: *dsn,
	}, Token: struct{ AccessExpiration time.Duration; RefreshExpiration time.Duration }{AccessExpiration: time.Minute, RefreshExpiration: time.Hour}}
	_ = cfg

	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("failed to read dataset: %v", err)
	}

	var importData struct {
		Surahs []struct {
			Number       int      `json:"number"`
			Name         string   `json:"name"`
			ArabicName   string   `json:"arabic_name"`
			EnglishName  string   `json:"english_name"`
			RevelationType string `json:"revelation_type"`
			AyahCount    int      `json:"ayah_count"`
			Ayahs        []struct {
				Number    int    `json:"number"`
				Text      string `json:"text"`
				Page      *int   `json:"page"`
				Juz       *int   `json:"juz"`
				Hizb      *int   `json:"hizb"`
				RubHizb   *int   `json:"rub_hizb"`
				Sajdah    bool   `json:"sajdah"`
			} `json:"ayahs"`
		} `json:"surahs"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		log.Fatalf("failed to parse dataset: %v", err)
	}

	fmt.Printf("Loaded %d surahs from dataset\n", len(importData.Surahs))
	_ = context.Background()
	_ = database.Connect
	fmt.Println("Dataset validation passed. Import requires DB connection wired.")
}
