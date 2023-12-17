package adapters

import (
	"context"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"generate-script-lambda/domain"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/rs/zerolog/log"
)

type dynamoSegmentItem struct {
	StoryId        string             `dynamodbav:"story_id"`
	SegmentId      string             `dynamodbav:"segment_id"`
	Text           string             `dynamodbav:"text"`
	S3Url          string             `dynamodbav:"s3_url"`
	Type           domain.SegmentType `dynamodbav:"type"`
	SegmentOrdinal int                `dynamodbav:"segment_ordinal"`
}

type dynamoCache struct {
	dynamoSvc    *dynamodb.DynamoDB
	dynamoConfig *config.DynamoConfig
}

func NewDynamoCache(dynamoSvc *dynamodb.DynamoDB, dynamoConfig *config.DynamoConfig) outbound.SegmentCachePort {
	return &dynamoCache{
		dynamoSvc:    dynamoSvc,
		dynamoConfig: dynamoConfig,
	}
}

func (p *dynamoCache) Save(ctx context.Context, segment domain.SegmentWithMediaUrl) error {
	item := dynamoSegmentItem{
		StoryId:        segment.StoryID,
		SegmentId:      segment.ID,
		Text:           segment.Text,
		S3Url:          segment.MediaURL,
		Type:           segment.Type,
		SegmentOrdinal: segment.Ordinal,
	}
	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		log.Error().
			Err(err).
			Interface("item", item).
			Msg("Failed to marshal item for DynamoDB")
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(p.dynamoConfig.TableName),
	}

	_, err = p.dynamoSvc.PutItemWithContext(ctx, input)
	if err != nil {
		log.Error().
			Err(err).
			Interface("input", input).
			Msg("Failed to publish item to DynamoDB")
	}

	return err
}
