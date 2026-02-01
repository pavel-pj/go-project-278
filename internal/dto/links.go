package dto

import (
	linksdb "db200/internal/db/links"
)

// CreateLinkRequest - структура для JSON запроса
type CreateLinkRequest struct {
	OriginalUrl string `json:"original_url" binding:"required,url"`
	ShortName   string `json:"short_name"`
	ShortUrl    string `json:"short_url" binding:"required,url"`
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
