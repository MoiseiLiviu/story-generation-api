package controllers

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/infrastructure/gin_interface/dto"
	"generate-script-lambda/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

type StorySegmentsController interface {
	CreateStory(c *gin.Context)
	RegisterRoutes(g *gin.Engine)
}

type storySegmentsController struct {
	workerPool           *ants.Pool
	pipelineOrchestrator inbound.SegmentPipelineOrchestrator
	storySaver           outbound.StorySaverPort
}

func NewStorySegmentsController(workerPool *ants.Pool, storyCreator inbound.SegmentPipelineOrchestrator, storySaver outbound.StorySaverPort) StorySegmentsController {
	return &storySegmentsController{
		workerPool:           workerPool,
		pipelineOrchestrator: storyCreator,
		storySaver:           storySaver,
	}
}

func (s *storySegmentsController) CreateStory(c *gin.Context) {
	var createStoryRequest dto.CreateStoryRequest
	newCtx, cancel := context.WithCancel(c)
	defer cancel()
	if err := c.ShouldBindJSON(&createStoryRequest); err != nil {
		c.Error(err)
		return
	}

	userID := c.GetString(middleware.ContextUserIDKey)

	storyID := uuid.NewString()

	segmentEvents, errCh := s.pipelineOrchestrator.StartPipeline(newCtx, inbound.StartPipelineParams{
		Input:   createStoryRequest.Input,
		StoryID: storyID,
		VoiceID: createStoryRequest.VoiceID,
	})

	err := s.workerPool.Submit(func() {
		for err := range errCh {
			cancel()
			log.Err(err).Msg("error in pipeline")
		}
		c.SSEvent("error", "internal server error")
	})
	if err != nil {
		log.Err(err).Msg("failed to submit error handler")
		c.SSEvent("error", "internal server error")
		return
	}

	for event := range segmentEvents {
		err := s.workerPool.Submit(func() {
			select {
			case <-newCtx.Done():
				return
			default:
				c.SSEvent("segment", event)
			}
		})
		if err != nil {
			c.SSEvent("error", "internal server error")
			return
		}

	}

	err = s.storySaver.Save(newCtx, outbound.SaveStoryParams{
		ID:     storyID,
		UserID: userID,
		Input:  createStoryRequest.Input,
	})
	if err != nil {
		log.Err(err).Msg("failed to save story")
		c.SSEvent("error", "internal server error")
		return
	}

	c.SSEvent("generation_complete", nil)
}

func (s *storySegmentsController) RegisterRoutes(g *gin.Engine) {
	g.POST("/stories", s.CreateStory)
}
