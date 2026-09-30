package storage

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Uploader struct {
	client *s3.Client
}

func NewS3Uploader(region, accessKeyID, secretKey string) *S3Uploader {
	config := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretKey, ""),
	}
	return &S3Uploader{client: s3.NewFromConfig(config)}
}

func (uploader *S3Uploader) Upload(requestContext context.Context, bucket, key string, body []byte, contentType string) error {
	_, err := uploader.client.PutObject(requestContext, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	return err
}
