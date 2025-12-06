package aws

import (
	"auction/pkg/config"
	"time"

	"github.com/gofiber/storage/s3/v2"
)

var appConfig = config.Read()

type S3 struct {
	bucket *s3.Storage
}

func NewS3Bucket() *S3 {
	s3 := s3.New(s3.Config{
		Endpoint: appConfig.AWSEndpoint,
		Bucket:   appConfig.AWSBucket,
		Region:   appConfig.AWSDefaultRegion,
		Credentials: s3.Credentials{
			AccessKey:       appConfig.AWSAccessKey,
			SecretAccessKey: appConfig.AWSSecretKey,
		},
		MaxAttempts:    3,
		RequestTimeout: time.Second * 10,
		Reset:          false,
	})

	return &S3{
		bucket: s3,
	}
}

func (s *S3) Upload(key string, data []byte) error {
	return s.bucket.Set(key, data, time.Hour*100)
}

func (s *S3) Download(key string) ([]byte, error) {
	return s.bucket.Get(key)
}

func (s *S3) Delete(key string) error {
	return s.bucket.Delete(key)
}
