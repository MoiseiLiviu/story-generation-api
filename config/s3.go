package config

import (
	"fmt"
	"os"
)

type S3Config struct {
	BucketName string
	Region     string
}

func GetS3Config() (*S3Config, error) {
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME must be set")
	}

	region := os.Getenv("REGION")
	if region == "" {
		return nil, fmt.Errorf("REGION must be set")
	}

	return &S3Config{
		BucketName: bucketName,
		Region:     region,
	}, nil
}
