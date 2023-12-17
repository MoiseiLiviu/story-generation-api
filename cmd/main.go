package main

import (
	"generate-script-lambda/application/services"
	"generate-script-lambda/config"
	"generate-script-lambda/infrastructure/adapters"
	"generate-script-lambda/infrastructure/gin_interface/controllers"
	"generate-script-lambda/middleware"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
)

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	wordsPerStory := os.Getenv("WORDS_PER_STORY")
	scriptStreamerWordsPerStory, err := strconv.Atoi(wordsPerStory)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse words per story")
	}

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

	dynamoConfig, err := config.GetDynamoConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get dynamo config")
	}

	authConfig, err := config.NewAuthorizerConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get authorizer config")
	}

	storyApiUrl := os.Getenv("STORY_API_URL")
	if storyApiUrl == "" {
		log.Fatal().Msg("STORY_API_URL environment variable not set")
	}

	jwksUrl := os.Getenv("JWKS_URL")
	if jwksUrl == "" {
		log.Fatal().Msg("JWKS_URL is not set!")
	}

	workerPool, err := ants.NewPool(120)
	defer workerPool.Release()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create worker pool")
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create aws session")
	}

	s3Client := s3.New(sess)
	dynamoClient := dynamodb.New(sess)

	contentFetcher := adapters.NewContentFetcher()

	audioGenerator := adapters.NewAudioGenerator(contentFetcher, elevenLabsConfig)
	imageGenerator := adapters.NewImageGenerator(contentFetcher, dalleConfig)

	authorizer := adapters.NewCognitoAuthorizer(authConfig)

	dynamoCache := adapters.NewDynamoCache(dynamoClient, dynamoConfig)

	s3MediaStore := adapters.NewS3SegmentMediaStore(s3Client, s3Config)

	storySaver := adapters.NewStorySaver(storyApiUrl, authorizer)

	storyScriptGenerator := adapters.NewStoryScriptGenerator(scriptStreamerWordsPerStory, gptConfig, workerPool)

	segmentMediaEnhancer := services.NewSegmentMediaEnhancer(imageGenerator, audioGenerator, workerPool)

	segmentMetadataSaver := services.NewSegmentMetadataSaver(workerPool, dynamoCache)

	segmentMediaSaver := services.NewSegmentMediaSaver(s3MediaStore, workerPool)

	segmentTextGenerator := services.NewSegmentTextGenerator(storyScriptGenerator, workerPool)

	storyCreator := services.NewSegmentPipelineOrchestrator(workerPool, segmentTextGenerator, segmentMediaEnhancer, segmentMediaSaver, segmentMetadataSaver)

	storySegmentController := controllers.NewStorySegmentsController(workerPool, storyCreator, storySaver)

	router := gin.Default()

	err = router.SetTrustedProxies(nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set trusted proxies!")
	}

	authHandler, err := middleware.NewAuthHandler(jwksUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create auth handler!")
	}

	router.Use(authHandler.AuthMiddleware())
	router.Use(middleware.SSEMiddleware(workerPool))

	storySegmentController.RegisterRoutes(router)

	err = router.Run(":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server!")
	}
}
