package main

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"time"
	"context"
	"fmt"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	pClient := s3.NewPresignClient(s3Client)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key: aws.String(key),
	}
	presignedURL, err := pClient.PresignGetObject(context.TODO(), input, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}
	
	return presignedURL.URL, nil
}
