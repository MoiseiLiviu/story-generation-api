package config

import (
	"fmt"
	"os"
)

type S3Config struct {
	BucketName string
}

func GetS3Config() (*S3Config, error) {
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME must be set")
	}

	return &S3Config{
		BucketName: bucketName,
	}, nil
}
