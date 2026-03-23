// Package dto provides data transfer objects for HTTP requests and responses.
package dto

import (
	"database/sql"
	linksdb "db200/internal/db/links"
)

// CreateLinkRequest - структура для JSON запроса
type CreateLinkRequest struct {
	OriginalUrl string `json:"original_url" binding:"required,url"`
	ShortName   string `json:"short_name"`
	ShortUrl    string `json:"short_url"`
}

// ResponseLink- структура для JSON ответ
type LinkResponse struct {
	ID          int32  `json:"id"`
	OriginalUrl string `json:"original_url"`
	ShortName   string `json:"short_name"`
	ShortUrl    string `json:"short_url"`
}

// ToCreateLinkParams конвертирует DTO в sqlc структуру
func (r *CreateLinkRequest) ToCreateLinkParams() linksdb.CreateLinkParams {
	params := linksdb.CreateLinkParams{
		OriginalUrl: r.OriginalUrl,
		ShortName:   r.ShortName,
		ShortUrl:    r.ShortUrl,
	}

	return params
}

// UpdateLinkRequest - структура для JSON запроса
type UpdateLinkRequest struct {
	OriginalUrl *string `json:"original_url,omitempty" binding:"omitempty,url"` // валидация только если поле присутствует
	ShortName   *string `json:"short_name,omitempty"`
}

// ToUpdateLinkParams конвертирует DTO в sqlc структуру
func (r *UpdateLinkRequest) ToUpdateLinkParams(linkID int32) linksdb.UpdateLinkParams {
	params := linksdb.UpdateLinkParams{
		ID:          linkID,
		OriginalUrl: sql.NullString{String: "", Valid: false},
		ShortName:   sql.NullString{String: "", Valid: false},
	}

	// Проверяем, что поле присутствует и не пустое
	if r.OriginalUrl != nil {
		params.OriginalUrl = sql.NullString{
			String: *r.OriginalUrl,
			Valid:  true,
		}
	}

	if r.ShortName != nil {
		params.ShortName = sql.NullString{
			String: *r.ShortName,
			Valid:  true,
		}
	}

	return params

}
