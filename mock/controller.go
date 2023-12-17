package main

import (
	"generate-script-lambda/infrastructure/gin_interface/dto"
	"github.com/gin-gonic/gin"
)

type MockSegmentController interface {
	CreateStory(c *gin.Context)
	RegisterRoutes(g *gin.Engine)
}

type mockSegmentController struct {
	segmentReader SegmentReader
}

func NewMockSegmentController(segmentReader SegmentReader) MockSegmentController {
	return &mockSegmentController{
		segmentReader: segmentReader,
	}
}

func (m *mockSegmentController) CreateStory(c *gin.Context) {
	var createStoryRequest dto.CreateStoryRequest
	if err := c.ShouldBindJSON(&createStoryRequest); err != nil {
		c.SSEvent("error", "internal server error")
	}

	segments := m.segmentReader.Read()

}

func (m *mockSegmentController) RegisterRoutes(g *gin.Engine) {
	g.POST("/mock", m.CreateStory)
}
