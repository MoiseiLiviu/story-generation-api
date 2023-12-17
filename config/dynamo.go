package config

import (
	"fmt"
	"os"
)

type DynamoConfig struct {
	TableName string
}

func GetDynamoConfig() (*DynamoConfig, error) {
	bucketName := os.Getenv("DYNAMO_TABLE_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("DYNAMO_TABLE_NAME must be set")
	}

	return &DynamoConfig{
		TableName: bucketName,
	}, nil
}
