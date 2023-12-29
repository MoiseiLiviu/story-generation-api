package main

import (
	"fmt"
	"generate-script-lambda/application/services"
	"generate-script-lambda/config"
	"generate-script-lambda/infrastructure/adapters"
	"generate-script-lambda/infrastructure/gin_interface/controllers"
	"github.com/gin-gonic/gin"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

func main() {
	gptConfig, err := config.GetGptConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get gpt config")
	}

	dalleConfig, err := config.GetDaLLeConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get dalle config")
	}

	elevenLabsConfig, err := config.GetElevenLabsConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get eleven labs config")
	}

	s3Config, err := config.GetS3Config()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get s3 config")
	}

	zeroLogger := adapters.NewZerologWrapper()

	panicHandler := func(p interface{}) {
		zeroLogger.Error(fmt.Errorf("%v", p), "Panic in worker pool")
	}

	workerPool, err := ants.NewPool(200, ants.WithPanicHandler(panicHandler))
	defer workerPool.Release()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create worker pool")
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create aws session")
	}

	contentFetcher := adapters.NewContentFetcher(zeroLogger)

	imageGenerator := adapters.NewImageGenerator(contentFetcher, dalleConfig, zeroLogger)

	audioGenerator := adapters.NewAudioGenerator(contentFetcher, elevenLabsConfig, zeroLogger)

	mediaFileGenerator := services.NewSegmentMediaFileGenerator(zeroLogger, imageGenerator, audioGenerator, workerPool)

	storyScriptGenerator := adapters.NewStoryScriptGenerator(gptConfig, workerPool, zeroLogger)

	segmentTextGenerator := services.NewSegmentTextGenerator(zeroLogger, storyScriptGenerator, workerPool)

	videoCreator := adapters.NewVideoCreator(zeroLogger)

	segmentMediaBinder := services.NewSegmentMediaBinder(zeroLogger, workerPool)

	videoPublisher := adapters.NewS3VideoPublisher(zeroLogger, s3Config)

	videoCreatorPipeline := services.NewVideoCreatorPipeline(segmentTextGenerator, mediaFileGenerator,
		segmentMediaBinder, zeroLogger, workerPool, videoPublisher, videoCreator)

	storySegmentController := controllers.NewStorySegmentsController(zeroLogger, videoCreatorPipeline)

	router := gin.Default()

	err = router.SetTrustedProxies(nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set trusted proxies!")
	}

	storySegmentController.RegisterRoutes(router)

	err = router.Run(":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server!")
	}
}
