// main содержит точку входа в приложение и настройку роутинга.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingHandler(t *testing.T) {

	router := setupRouter()
	router = pingHandler(router)

	w := httptest.NewRecorder()
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", "/ping", nil)
	router.ServeHTTP(w, req)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", response["message"])

}
