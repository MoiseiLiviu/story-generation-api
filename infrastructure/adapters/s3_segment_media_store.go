package adapters

import (
	"context"
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"generate-script-lambda/domain"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"strings"
)

type s3SegmentMediaStore struct {
	logger   outbound.LoggerPort
	s3Svc    *s3.S3
	s3Config *config.S3Config
}

func NewS3SegmentMediaStore(s3Svc *s3.S3, s3Config *config.S3Config, logger outbound.LoggerPort) outbound.SegmentMediaStorePort {
	return &s3SegmentMediaStore{
		logger:   logger,
		s3Svc:    s3Svc,
		s3Config: s3Config,
	}
}

func (s *s3SegmentMediaStore) Save(ctx context.Context, segment domain.SegmentWithMedia, userID string) (string, error) {
	itemPath := s.getS3ItemPath(segment, userID)

	putInput := &s3.PutObjectInput{
		Bucket:        aws.String(s.s3Config.BucketName),
		Key:           aws.String(itemPath),
		Body:          strings.NewReader(string(segment.MediaContent)),
		ContentLength: aws.Int64(int64(len(segment.MediaContent))),
	}

	_, err := s.s3Svc.PutObjectWithContext(ctx, putInput)
	if err != nil {
		s.logger.Error(err, "Failed to upload object to S3")
		return "", err
	}

	s3Url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.s3Config.BucketName, segment.ID)

	return s3Url, nil
}

func (s *s3SegmentMediaStore) getS3ItemPath(segment domain.SegmentWithMedia, userID string) string {
	return fmt.Sprintf("user/%s/story/%s/%s/%s", userID, segment.StoryID, segment.Type, segment.ID)
}
