package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tomlord.io-backend/internal/services"
)

func (s *Server) listThreadsHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	page, err := s.threadService.ListThreads(c.Request.Context(), services.ListThreadsRequest{
		Limit:  int32(limit),
		Cursor: c.Query("cursor"),
	})
	if err != nil {
		writeThreadError(c, err)
		return
	}

	c.JSON(http.StatusOK, page)
}

func (s *Server) getThreadHandler(c *gin.Context) {
	thread, err := s.threadService.GetThread(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeThreadError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"thread": thread})
}

func (s *Server) createThreadHandler(c *gin.Context) {
	var req services.CreateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	thread, err := s.threadService.CreateThread(c.Request.Context(), req)
	if err != nil {
		writeThreadError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"thread": thread})
}

func (s *Server) updateThreadHandler(c *gin.Context) {
	var req services.UpdateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	thread, err := s.threadService.UpdateThread(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		writeThreadError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"thread": thread})
}

func (s *Server) deleteThreadHandler(c *gin.Context) {
	if err := s.threadService.DeleteThread(c.Request.Context(), c.Param("id")); err != nil {
		writeThreadError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Thread deleted"})
}

func writeThreadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrThreadNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Thread not found"})
	case errors.Is(err, services.ErrThreadInvalid), errors.Is(err, services.ErrThreadCursor):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process thread"})
	}
}
