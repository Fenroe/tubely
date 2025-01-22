package main

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3client)
	input := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	req, err := presignClient.PresignGetObject(context.TODO(), &input, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
