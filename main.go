package main

import (
	"db200/internal/app"
	d "db200/internal/db"
	r "db200/router"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {

	db, err := d.Connect()
	if err != nil {
		log.Fatal("Database Error: ", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Warning: failed to close database connection: %v", err)
		}
	}()
	app := app.NewApp(db)
	router := r.NewRouter(app)

	// Run server (blocks until stopped)
	err = router.Run(":8080")
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}

}
