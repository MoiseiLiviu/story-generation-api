package controllers

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/infrastructure/gin_interface/dto"
	"generate-script-lambda/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sync"
)

type StorySegmentsController interface {
	CreateStory(c *gin.Context)
	RegisterRoutes(g *gin.Engine)
}

type storySegmentsController struct {
	logger               outbound.LoggerPort
	workerPool           outbound.TaskDispatcher
	pipelineOrchestrator inbound.SegmentPipelineOrchestrator
	storySaver           outbound.StorySaverPort
}

func NewStorySegmentsController(logger outbound.LoggerPort, workerPool outbound.TaskDispatcher,
	pipelineOrchestrator inbound.SegmentPipelineOrchestrator, storySaver outbound.StorySaverPort) StorySegmentsController {
	return &storySegmentsController{
		logger:               logger,
		workerPool:           workerPool,
		pipelineOrchestrator: pipelineOrchestrator,
		storySaver:           storySaver,
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

	userID := c.GetString(middleware.ContextUserIDKey)

	storyID := uuid.NewString()

	segmentEvents, errCh := s.pipelineOrchestrator.StartPipeline(newCtx, inbound.StartPipelineParams{
		Input:   createStoryRequest.Input,
		StoryID: storyID,
		VoiceID: createStoryRequest.VoiceID,
		UserID:  userID,
	})

	err := s.workerPool.Submit(func() {
		var sendErrOnce sync.Once
		for err := range errCh {
			cancel()
			s.logger.Error(err, "error in pipeline")
			sendErrOnce.Do(func() {
				c.SSEvent("error", "internal server error")
				c.Abort()
			})
		}
	})
	if err != nil {
		s.logger.Error(err, "failed to submit worker")
		c.SSEvent("error", "internal server error")
		return
	}

	for event := range segmentEvents {
		select {
		case <-newCtx.Done():
			return
		default:
			c.SSEvent("segment", event)
		}
	}

	s.logger.InfoWithFields("segments generation complete", map[string]interface{}{
		"story_id": storyID,
	})

	err = s.storySaver.Save(newCtx, outbound.SaveStoryParams{
		ID:     storyID,
		UserID: userID,
		Input:  createStoryRequest.Input,
	})
	if err != nil {
		s.logger.Error(err, "failed to save story")
		c.SSEvent("error", "internal server error")
		return
	} else {
		s.logger.InfoWithFields("story saved", map[string]interface{}{
			"story_id": storyID,
		})
	}

	c.SSEvent("generation_complete", nil)
}

func (s *storySegmentsController) RegisterRoutes(g *gin.Engine) {
	g.POST("/generate", s.CreateStory)
}
