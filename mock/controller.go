package mock_generator

import (
	"context"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/infrastructure/gin_interface/dto"
	"generate-script-lambda/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"sync"
)

type MockSegmentController interface {
	CreateStory(c *gin.Context)
	RegisterRoutes(g *gin.Engine)
}

type mockSegmentController struct {
	logger     outbound.LoggerPort
	workerPool outbound.TaskDispatcher
	runner     *Runner
	storySaver outbound.StorySaverPort
}

func NewMockSegmentController(logger outbound.LoggerPort, workerPool outbound.TaskDispatcher, runner *Runner,
	storySaver outbound.StorySaverPort) MockSegmentController {
	return &mockSegmentController{
		logger:     logger,
		workerPool: workerPool,
		runner:     runner,
		storySaver: storySaver,
	}
}

func (m *mockSegmentController) CreateStory(c *gin.Context) {
	var createStoryRequest dto.CreateStoryRequest
	newCtx, cancel := context.WithCancel(c)
	defer cancel()
	defer c.Abort()
	if err := c.ShouldBindJSON(&createStoryRequest); err != nil {
		err = c.AbortWithError(400, err)
		if err != nil {
			m.logger.Error(err, "failed to abort with error")
		}
		return
	}

	userID := c.GetString(middleware.ContextUserIDKey)

	storyID := uuid.NewString()

	segmentEvents, errCh := m.runner.Run(newCtx)

	err := m.workerPool.Submit(func() {
		var sendErrOnce sync.Once
		for err := range errCh {
			cancel()
			m.logger.Error(err, "error in pipeline")
			sendErrOnce.Do(func() {
				c.SSEvent("error", "internal server error")
				c.Abort()
			})
		}
	})
	if err != nil {
		m.logger.Error(err, "failed to submit worker")
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

		if err != nil {
			c.SSEvent("error", "internal server error")
			return
		}
	}

	m.logger.InfoWithFields("segments generation complete", map[string]interface{}{
		"story_id": storyID,
	})

	err = m.storySaver.Save(newCtx, outbound.SaveStoryParams{
		ID:     storyID,
		UserID: userID,
		Input:  createStoryRequest.Input,
	})
	if err != nil {
		m.logger.Error(err, "failed to save story")
		c.SSEvent("error", "internal server error")
		return
	} else {
		m.logger.InfoWithFields("story saved", map[string]interface{}{
			"story_id": storyID,
		})
	}

	c.SSEvent("generation_complete", nil)
}

func (m *mockSegmentController) RegisterRoutes(g *gin.Engine) {
	g.POST("generate/mock", m.CreateStory)
}
