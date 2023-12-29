package controllers

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/infrastructure/gin_interface/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StorySegmentsController interface {
	CreateStory(c *gin.Context)
	RegisterRoutes(g *gin.Engine)
}

type storySegmentsController struct {
	logger               outbound.LoggerPort
	videoCreatorPipeline inbound.VideoCreatorPipelinePort
}

func NewStorySegmentsController(
	logger outbound.LoggerPort,
	pipelineOrchestrator inbound.VideoCreatorPipelinePort,
) StorySegmentsController {
	return &storySegmentsController{
		logger:               logger,
		videoCreatorPipeline: pipelineOrchestrator,
	}
}

func (s *storySegmentsController) CreateStory(c *gin.Context) {
	var createStoryRequest dto.CreateStoryRequest
	newCtx, cancel := context.WithCancel(c)
	defer cancel()
	if err := c.ShouldBindJSON(&createStoryRequest); err != nil {
		err = c.AbortWithError(400, err)
		if err != nil {
			s.logger.Error(err, "failed to abort with error")
		}
		return
	}

	storyID := uuid.NewString()

	res, err := s.videoCreatorPipeline.StartPipeline(newCtx, inbound.StartPipelineParams{
		Input:         createStoryRequest.Input,
		StoryID:       storyID,
		UserID:        createStoryRequest.UserID,
		VoiceID:       createStoryRequest.VoiceID,
		WordsPerStory: createStoryRequest.WordsPerStory,
	})
	if err != nil {
		err = c.AbortWithError(500, err)
		if err != nil {
			s.logger.Error(err, "failed to abort with error")
		}
		return
	}

	c.JSON(200, dto.CreateStoryResponse{
		StoryID:     storyID,
		VideoKey:    res.VideoKey,
		VideoRegion: res.VideoRegion,
	})
}

func (s *storySegmentsController) RegisterRoutes(g *gin.Engine) {
	g.POST("/generate", s.CreateStory)
}
