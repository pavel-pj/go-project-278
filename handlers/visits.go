package handlers

import (
	s "db200/services"
)

type VisitHandler struct {
	service *s.VisitService
}

func NewVisitHandler(service *s.VisitService) *VisitHandler {
	return &VisitHandler{
		service: service,
	}
}

/*
func (h *VisitHandler) Create(c *gin.Context) {
	var req dto.CreateLinkRequest

	// Валидируем JSON
	if err := c.ShouldBindJSON(&req); err != nil {

*/
