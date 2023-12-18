package mock_generator

import (
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"github.com/gin-gonic/gin"
)

func Init(g *gin.Engine, workerPool outbound.TaskDispatcher, metadataSaver inbound.SegmentMetadataSaverPort, storySaver outbound.StorySaverPort,
	logger outbound.LoggerPort) {
	segmentReader := NewFileSegmentReader(logger)
	runner := NewRunner(workerPool, segmentReader, metadataSaver, logger)
	mockController := NewMockSegmentController(logger, workerPool, runner, storySaver)

	mockController.RegisterRoutes(g)
}
