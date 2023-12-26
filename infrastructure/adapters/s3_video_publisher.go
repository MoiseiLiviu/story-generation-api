package adapters

import (
	"context"
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"os"
)

type s3VideoPublsiher struct {
	logger   outbound.LoggerPort
	s3Svc    *s3.S3
	s3Config *config.S3Config
}

func NewS3VideoPublisher(logger outbound.LoggerPort, s3Config *config.S3Config) outbound.VideoPublisherPort {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(s3Config.Region)})
	if err != nil {
		logger.Error(err, "Failed to create session")
	}
	return &s3VideoPublsiher{
		logger:   logger,
		s3Svc:    s3.New(sess),
		s3Config: s3Config,
	}
}

func (s *s3VideoPublsiher) Publish(ctx context.Context, req outbound.PublishVideoRequest) (*outbound.PublishVideoResponse, error) {
	itemPath := s.getS3ItemPath(req)

	file, err := os.Open(req.VideoFileName)
	if err != nil {
		s.logger.Error(err, "Failed to open video file")
		return nil, err
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			s.logger.Error(err, "Failed to close video file")
			return
		}
		err = os.Remove(file.Name())
		if err != nil {
			s.logger.Error(err, "Failed to remove video file")
			return
		}
	}(file)

	putInput := &s3.PutObjectInput{
		Bucket: aws.String(s.s3Config.BucketName),
		Key:    aws.String(itemPath),
		Body:   file,
	}

	_, err = s.s3Svc.PutObjectWithContext(ctx, putInput)
	if err != nil {
		s.logger.Error(err, "Failed to upload object to S3")
		return nil, err
	}

	return &outbound.PublishVideoResponse{
		VideoKey:    itemPath,
		StoreRegion: s.s3Config.Region,
	}, nil
}

func (s *s3VideoPublsiher) getS3ItemPath(req outbound.PublishVideoRequest) string {
	return fmt.Sprintf("user/%s/story/%s/video/%s", req.UserID, req.StoryID, uuid.NewString())
}
