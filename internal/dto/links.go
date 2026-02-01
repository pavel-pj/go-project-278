package dto

import (
	"database/sql"
	linksdb "db200/internal/db/links"
)

// CreateLinkRequest - структура для JSON запроса
type CreateLinkRequest struct {
	OriginalUrl string `json:"original_url" binding:"required,url"`
	ShortName   string `json:"short_name,omitempty"`
	ShortUrl    string `json:"short_url,omitempty"`
}

// ToCreateLinkParams конвертирует DTO в sqlc структуру
func (r *CreateLinkRequest) ToCreateLinkParams() linksdb.CreateLinkParams {
	params := linksdb.CreateLinkParams{
		OriginalUrl: r.OriginalUrl,
		ShortUrl:    r.ShortUrl,
	}

	// Конвертируем string в sql.NullString
	if r.ShortName != "" {
		params.ShortName = sql.NullString{
			String: r.ShortName,
			Valid:  true,
		}
	}

	return params
}

// LinkResponse - структура для JSON ответа
type LinkResponse struct {
	ID          int32  `json:"id"`
	OriginalUrl string `json:"original_url"`
	ShortName   string `json:"short_name,omitempty"`
	ShortUrl    string `json:"short_url"`
}

// FromLink конвертирует sqlc Link в DTO
func FromLink(link linksdb.Link) LinkResponse {
	resp := LinkResponse{
		ID:          link.ID,
		OriginalUrl: link.OriginalUrl,
		ShortUrl:    link.ShortUrl,
	}

	if link.ShortName.Valid {
		resp.ShortName = link.ShortName.String
	}

	return resp
}
