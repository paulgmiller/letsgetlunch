package main

import (
	"log"
	"net/http"
	"os"

	"letsgetlunch/internal/lets"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "lets.db"
	}

	db, err := lets.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	app, err := lets.NewApp(db)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		addr = ":" + port
	}

	log.Printf("lets listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, app.Routes()))
}
