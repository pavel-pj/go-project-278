package main

import (
	"context"
	"db200/internal/app"
	d "db200/internal/db"
	r "db200/router"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingHandler(t *testing.T) {

	db, err := d.Connect()
	if err != nil {
		log.Fatal("Database Error: ", err)
	}
	app := app.NewApp(db)
	router := r.NewRouter(app)

	w := httptest.NewRecorder()
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/ping", nil)
	router.ServeHTTP(w, req)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", response["message"])

}
