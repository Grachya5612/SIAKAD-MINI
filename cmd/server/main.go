package main

import (
	"fmt"
	"log"
	"os"

	"siakad-mini/internal/config"
	"siakad-mini/internal/database"
	"siakad-mini/internal/router"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg)

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "migrate":
		if err := database.Migrate(db); err != nil {
			log.Fatalf("migrate gagal: %v", err)
		}
		fmt.Println("migrate selesai")
	case "seed":
		if err := database.Seed(db); err != nil {
			log.Fatalf("seed gagal: %v", err)
		}
		fmt.Println("seed selesai")
	case "serve":
		app := router.New(cfg, db)
		fmt.Printf("Server berjalan di http://localhost:%s\n", cfg.AppPort)
		log.Fatal(app.Listen(":" + cfg.AppPort))
	default:
		fmt.Println("perintah: migrate | seed | serve")
	}
}
