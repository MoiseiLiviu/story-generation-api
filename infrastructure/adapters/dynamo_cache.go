package adapters

import (
	"context"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"generate-script-lambda/domain"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"time"
)

type dynamoSegmentItem struct {
	StoryId        string  `dynamodbav:"story_id"`
	SegmentId      string  `dynamodbav:"segment_id"`
	Text           string  `dynamodbav:"text"`
	SegmentOrdinal int     `dynamodbav:"segment_ordinal"`
	TTL            int64   `dynamodbav:"ttl"`
	Duration       float64 `dynamodbav:"duration"`
}

type dynamoCache struct {
	logger       outbound.LoggerPort
	dynamoSvc    *dynamodb.DynamoDB
	dynamoConfig *config.DynamoConfig
}

func NewDynamoCache(logger outbound.LoggerPort, dynamoSvc *dynamodb.DynamoDB, dynamoConfig *config.DynamoConfig) outbound.SegmentCachePort {
	return &dynamoCache{
		logger:       logger,
		dynamoSvc:    dynamoSvc,
		dynamoConfig: dynamoConfig,
	}
}

func (c *dynamoCache) Save(ctx context.Context, segment domain.VideoSegment, storyID string) error {
	item := dynamoSegmentItem{
		StoryId:        storyID,
		SegmentId:      segment.ID,
		Text:           segment.Text,
		Duration:       segment.Duration,
		SegmentOrdinal: segment.Ordinal,
		TTL:            time.Now().Add(time.Duration(c.dynamoConfig.TtlMinutes) * time.Minute).Unix(),
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		c.logger.ErrorWithFields(err, "Failed to marshal segment item", map[string]interface{}{
			"item": item,
		})
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(c.dynamoConfig.TableName),
	}

	_, err = c.dynamoSvc.PutItemWithContext(ctx, input)
	if err != nil {
		c.logger.ErrorWithFields(err, "Failed to save segment item", map[string]interface{}{
			"item": item,
		})
		return err
	}

	return err
}
