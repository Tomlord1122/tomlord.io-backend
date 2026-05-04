package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) previewHandler(c *gin.Context) {
	urlStr := c.Query("url")
	if urlStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing url query parameter"})
		return
	}

	preview, err := s.previewService.FetchPreview(c.Request.Context(), urlStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preview)
}
