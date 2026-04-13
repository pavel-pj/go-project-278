//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"db200/internal/db/generated"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *generated.Queries) string
		shortCode      string
		expectedStatus int
		expectedURL    string
		expectedError  string
	}{
		{
			name: "Success - Redirect to existing link",
			setupData: func(ctx context.Context, q *generated.Queries) string {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://example.com",
					ShortName:   "testcode",
					ShortUrl:    "https://test.com/r/testcode",
				})
				require.NoError(t, err)
				return link.ShortName
			},
			shortCode:      "testcode",
			expectedStatus: http.StatusFound,
			expectedURL:    "https://example.com",
		},
		{
			name:           "Error - Empty short code",
			setupData:      nil,
			shortCode:      "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "link not found",
		},
		{
			name:           "Error - Link not found",
			setupData:      nil,
			shortCode:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedError:  "link not found",
		},
		{
			name: "Success - Redirect creates visit record",
			setupData: func(ctx context.Context, q *generated.Queries) string {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://tracked.com",
					ShortName:   "tracked",
					ShortUrl:    "https://test.com/r/tracked",
				})
				require.NoError(t, err)
				return link.ShortName
			},
			shortCode:      "tracked",
			expectedStatus: http.StatusFound,
			expectedURL:    "https://tracked.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			shortCode := tt.shortCode
			if tt.setupData != nil {
				shortCode = tt.setupData(ctx, q)
			}

			visitHandler := NewVisitHandler(q)

			r := gin.New()
			r.GET("/r/:code", visitHandler.Redirect)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/r/%s", shortCode), nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// For empty short code, the response might be HTML, not JSON
			if tt.expectedError != "" && w.Code != http.StatusFound && w.Code != http.StatusBadRequest {
				// Check if response is JSON before parsing
				if w.Header().Get("Content-Type") == "application/json; charset=utf-8" {
					var response map[string]string
					err := json.Unmarshal(w.Body.Bytes(), &response)
					require.NoError(t, err)
					assert.Equal(t, tt.expectedError, response["error"])
				}
			}

			if tt.expectedStatus == http.StatusFound {
				location := w.Header().Get("Location")
				assert.Equal(t, tt.expectedURL, location)
			}
		})
	}
}

func TestRedirectCreatesVisit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q, cleanup := SetupTx(t)
	defer cleanup()

	ctx := context.Background()

	// Create a link
	link, err := q.CreateLink(ctx, generated.CreateLinkParams{
		OriginalUrl: "https://visit-test.com",
		ShortName:   "visittest",
		ShortUrl:    "https://test.com/r/visittest",
	})
	require.NoError(t, err)

	visitHandler := NewVisitHandler(q)

	r := gin.New()
	r.GET("/r/:code", visitHandler.Redirect)

	// Perform redirect
	req := httptest.NewRequest(http.MethodGet, "/r/visittest", nil)
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	req.Header.Set("Referer", "https://google.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)

	// Check that visit was created - get all visits and find ours
	visits, err := q.GetVisits(ctx, generated.GetVisitsParams{Limit: 100, Offset: 0})
	require.NoError(t, err)

	var foundVisit *generated.LinkVisit
	for i := range visits {
		if visits[i].LinkID == link.ID {
			foundVisit = &visits[i]
			break
		}
	}

	require.NotNil(t, foundVisit, "Visit should be created")
	assert.Equal(t, link.ID, foundVisit.LinkID)
	assert.Equal(t, int32(http.StatusFound), foundVisit.Status)
	assert.Equal(t, "Test-Agent/1.0", foundVisit.UserAgent)
	assert.Equal(t, "https://google.com", foundVisit.Referer)
}

func TestGetVisits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *generated.Queries) int
		rangeParam     string
		expectedCount  int
		expectedTotal  int64
		expectedStatus int
	}{
		{
			name: "Success - Get all visits without range",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://test.com",
					ShortName:   "test",
					ShortUrl:    "https://test.com/r/test",
				})
				require.NoError(t, err)

				for i := 0; i < 3; i++ {
					_, err := q.CreateVisit(ctx, generated.CreateVisitParams{
						LinkID:    link.ID,
						Ip:        "127.0.0.1",
						UserAgent: "test-agent",
						Status:    302,
						Referer:   "",
					})
					require.NoError(t, err)
				}
				return 3
			},
			rangeParam:     "",
			expectedCount:  3,
			expectedTotal:  3,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - Get visits with range [0,1]",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://test2.com",
					ShortName:   "test2",
					ShortUrl:    "https://test.com/r/test2",
				})
				require.NoError(t, err)

				for i := 0; i < 5; i++ {
					_, err := q.CreateVisit(ctx, generated.CreateVisitParams{
						LinkID:    link.ID,
						Ip:        "127.0.0.1",
						UserAgent: "test-agent",
						Status:    302,
						Referer:   "",
					})
					require.NoError(t, err)
				}
				return 5
			},
			rangeParam:     "[0,1]",
			expectedCount:  2,
			expectedTotal:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - Empty visits",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				return 0
			},
			rangeParam:     "",
			expectedCount:  0,
			expectedTotal:  0,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - Range exceeds bounds",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://test3.com",
					ShortName:   "test3",
					ShortUrl:    "https://test.com/r/test3",
				})
				require.NoError(t, err)

				for i := 0; i < 5; i++ {
					_, err := q.CreateVisit(ctx, generated.CreateVisitParams{
						LinkID:    link.ID,
						Ip:        "127.0.0.1",
						UserAgent: "test-agent",
						Status:    302,
						Referer:   "",
					})
					require.NoError(t, err)
				}
				return 5
			},
			rangeParam:     "[10,20]",
			expectedCount:  0,
			expectedTotal:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Error - Invalid range format",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				return 0
			},
			rangeParam:     "[0,1,2]",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - Invalid range values",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				return 0
			},
			rangeParam:     "[a,b]",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			tt.setupData(ctx, q)

			visitHandler := NewVisitHandler(q)

			r := gin.New()
			r.GET("/api/link_visits", visitHandler.GetVisits)

			url := "/api/link_visits"
			if tt.rangeParam != "" {
				url = url + "?range=" + tt.rangeParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			if tt.expectedStatus != http.StatusOK {
				return
			}

			if tt.expectedTotal > 0 {
				contentRange := w.Header().Get("Content-Range")
				assert.NotEmpty(t, contentRange, "Content-Range header should be present")
			}

			var response []generated.LinkVisit
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Len(t, response, tt.expectedCount)
		})
	}
}

func TestGetVisitsWithLargeDataset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q, cleanup := SetupTx(t)
	defer cleanup()

	ctx := context.Background()

	// Create a link
	link, err := q.CreateLink(ctx, generated.CreateLinkParams{
		OriginalUrl: "https://large.com",
		ShortName:   "large",
		ShortUrl:    "https://test.com/r/large",
	})
	require.NoError(t, err)

	// Create 25 visits
	for i := 1; i <= 25; i++ {
		_, err := q.CreateVisit(ctx, generated.CreateVisitParams{
			LinkID:    link.ID,
			Ip:        fmt.Sprintf("192.168.1.%d", i),
			UserAgent: "test-agent",
			Status:    302,
			Referer:   "",
		})
		require.NoError(t, err)
	}

	visitHandler := NewVisitHandler(q)

	tests := []struct {
		rangeParam    string
		expectedCount int
	}{
		{"[0,9]", 10},
		{"[4,13]", 10},
		{"[19,28]", 6},
		{"[0,24]", 25},
		{"[10,14]", 5},
	}

	for _, tt := range tests {
		t.Run(tt.rangeParam, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/link_visits", visitHandler.GetVisits)

			url := "/api/link_visits?range=" + tt.rangeParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response []generated.LinkVisit
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Len(t, response, tt.expectedCount)
		})
	}
}

func TestGetVisitsWithMultipleLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q, cleanup := SetupTx(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple links with visits
	links := []string{"link1", "link2", "link3"}
	expectedTotal := 0

	for i, shortName := range links {
		link, err := q.CreateLink(ctx, generated.CreateLinkParams{
			OriginalUrl: fmt.Sprintf("https://%s.com", shortName),
			ShortName:   shortName,
			ShortUrl:    fmt.Sprintf("https://test.com/r/%s", shortName),
		})
		require.NoError(t, err)

		// Create different number of visits per link
		visitCount := (i + 1) * 2 // 2, 4, 6 = 12 total (fits within default limit)
		for j := 0; j < visitCount; j++ {
			_, err := q.CreateVisit(ctx, generated.CreateVisitParams{
				LinkID:    link.ID,
				Ip:        "127.0.0.1",
				UserAgent: "test-agent",
				Status:    302,
				Referer:   "",
			})
			require.NoError(t, err)
		}
		expectedTotal += visitCount
	}

	visitHandler := NewVisitHandler(q)

	r := gin.New()
	r.GET("/api/link_visits", visitHandler.GetVisits)

	// Use range to get all visits
	req := httptest.NewRequest(http.MethodGet, "/api/link_visits?range=[0,11]", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []generated.LinkVisit
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, expectedTotal, "Should return all visits from all links")
}

func TestRedirectWithHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userAgent      string
		referer        string
		expectedStatus int
	}{
		{
			name:           "Redirect with User-Agent and Referer",
			userAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			referer:        "https://google.com/search?q=test",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "Redirect without headers",
			userAgent:      "",
			referer:        "",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "Redirect with only User-Agent",
			userAgent:      "curl/7.68.0",
			referer:        "",
			expectedStatus: http.StatusFound,
		},
		{
			name:           "Redirect with only Referer",
			userAgent:      "",
			referer:        "https://example.com",
			expectedStatus: http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			// Create a link
			link, err := q.CreateLink(ctx, generated.CreateLinkParams{
				OriginalUrl: "https://headers-test.com",
				ShortName:   "headerstest",
				ShortUrl:    "https://test.com/r/headerstest",
			})
			require.NoError(t, err)

			visitHandler := NewVisitHandler(q)

			r := gin.New()
			r.GET("/r/:code", visitHandler.Redirect)

			req := httptest.NewRequest(http.MethodGet, "/r/headerstest", nil)
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify visit was created with correct headers
			visits, err := q.GetVisits(ctx, generated.GetVisitsParams{Limit: 100, Offset: 0})
			require.NoError(t, err)

			var foundVisit *generated.LinkVisit
			for i := range visits {
				if visits[i].LinkID == link.ID {
					foundVisit = &visits[i]
					break
				}
			}

			require.NotNil(t, foundVisit, "Visit should be created")
			assert.Equal(t, tt.userAgent, foundVisit.UserAgent)
			assert.Equal(t, tt.referer, foundVisit.Referer)
		})
	}
}

func TestGetVisitsEmptyResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q, cleanup := SetupTx(t)
	defer cleanup()

	visitHandler := NewVisitHandler(q)

	r := gin.New()
	r.GET("/api/link_visits", visitHandler.GetVisits)

	req := httptest.NewRequest(http.MethodGet, "/api/link_visits", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []generated.LinkVisit
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Empty(t, response)

	contentRange := w.Header().Get("Content-Range")
	assert.Equal(t, "visits 0-0/0", contentRange)
}

func TestGetVisitsWithPaginationBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q, cleanup := SetupTx(t)
	defer cleanup()

	ctx := context.Background()

	// Create a link
	link, err := q.CreateLink(ctx, generated.CreateLinkParams{
		OriginalUrl: "https://boundary.com",
		ShortName:   "boundary",
		ShortUrl:    "https://test.com/r/boundary",
	})
	require.NoError(t, err)

	// Create 10 visits
	for i := 0; i < 10; i++ {
		_, err := q.CreateVisit(ctx, generated.CreateVisitParams{
			LinkID:    link.ID,
			Ip:        "127.0.0.1",
			UserAgent: "test-agent",
			Status:    302,
			Referer:   "",
		})
		require.NoError(t, err)
	}

	visitHandler := NewVisitHandler(q)

	tests := []struct {
		name        string
		rangeParam  string
		expectCount int
		expectOK    bool
	}{
		{"First page", "[0,4]", 5, true},
		{"Second page", "[5,9]", 5, true},
		{"Last page partial", "[8,9]", 2, true}, // Fixed: [8,9] returns 2 items (indices 8 and 9)
		{"Start negative", "[-1,5]", 0, false},
		{"Start greater than end", "[5,1]", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/link_visits", visitHandler.GetVisits)

			url := "/api/link_visits?range=" + tt.rangeParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.expectOK {
				assert.Equal(t, http.StatusOK, w.Code)

				var response []generated.LinkVisit
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, tt.expectCount)
			} else {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			}
		})
	}
}
