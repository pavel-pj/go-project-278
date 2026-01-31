// main go
package main

import (
	r "db200/router"
	"log"
)

func main() {
	router := r.NewRouter()

	// Run server (blocks until stopped)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}

}
