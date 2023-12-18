package config

import (
	"fmt"
	"os"
	"strconv"
)

type DynamoConfig struct {
	TableName  string
	TtlMinutes int
}

func GetDynamoConfig() (*DynamoConfig, error) {
	bucketName := os.Getenv("DYNAMO_TABLE_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("DYNAMO_TABLE_NAME must be set")
	}

	ttlMinutes := os.Getenv("DYNAMO_TTL_MINUTES")
	if ttlMinutes == "" {
		return nil, fmt.Errorf("DYNAMO_TTL_MINUTES must be set")
	}

	ttlNumber, err := strconv.Atoi(ttlMinutes)
	if err != nil {
		return nil, fmt.Errorf("DYNAMO_TTL_MINUTES must be a number")
	}

	return &DynamoConfig{
		TableName:  bucketName,
		TtlMinutes: ttlNumber,
	}, nil
}
