package handlers

import (
	r "db200/repositories"
)

type VisitHandler struct {
	VisitRepository *r.VisitRepository
	LinkRepository  *r.LinkRepository
}

func NewVisitHandler(visitRepository *r.VisitRepository, linkRepository *r.LinkRepository) *VisitHandler {
	return &VisitHandler{
		LinkRepository:  linkRepository,
		VisitRepository: visitRepository,
	}
}

/*
func (h *VisitHandler) Redirect(c *gin.Context) {
	shortName := c.Param("code")
	if shortName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "short name is required"})
		return
	}

	// Get link by short name
	link, err := h.service.GetByShortName(shortName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	// Get client information
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	referer := c.GetHeader("Referer")

	// Determine redirect status (302 Found for temporary redirect)
	status := http.StatusFound // 302

	// Record visit
	_, err = h.service.Create(link.ID, ip, userAgent, referer, status)
	if err != nil {
		// Log error but still redirect
		c.Error(err)
	}

	// Perform redirect
	c.Redirect(status, link.OriginalURL)
}
*/
