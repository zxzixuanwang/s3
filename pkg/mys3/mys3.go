package mys3

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func New(endpoint, region string, https bool) Mys3 {
	able := false
	if https {
		able = true
	}
	sess, _ := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		DisableSSL:       aws.Bool(able),
		S3ForcePathStyle: aws.Bool(true),
	})

	return &s3Service{sess: sess, svc: s3.New(sess)}
}

type Mys3 interface {
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	ListObject(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
	Upload(input *s3manager.UploadInput) (*s3manager.UploadOutput, error)
}

type s3Service struct {
	sess *session.Session
	svc  *s3.S3
}

func (s *s3Service) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	out, err := s.svc.GetObject(input)
	if err != nil {
		log.Println("getObject:", err)
		return nil, err
	}
	return out, err
}

func (s *s3Service) Upload(input *s3manager.UploadInput) (*s3manager.UploadOutput, error) {
	uploader := s3manager.NewUploader(s.sess)

	up, err := uploader.Upload(input)
	if err != nil {
		log.Println("upload:", err)
		return nil, err
	}
	return up, nil
}

func (s *s3Service) ListObject(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	out, err := s.svc.ListObjects(input)
	if err != nil {
		log.Println("list err", err)
		return nil, err
	}
	return out, nil
}
