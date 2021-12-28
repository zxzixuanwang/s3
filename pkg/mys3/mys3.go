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

	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		DisableSSL:       aws.Bool(able),
		S3ForcePathStyle: aws.Bool(true),
	}))
	return &s3Service{sess: sess, svc: s3.New(sess)}
}

type Mys3 interface {
	UploadPart(input *s3.UploadPartInput) (*s3.UploadPartOutput, error)
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	ListObject(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
	Upload(input *s3manager.UploadInput) (*s3manager.UploadOutput, error)
	MultipartUploads(input *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error)
	CompleteMultipartUpload(input *s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(input *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error)
	CreateMultipartUpload(input *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error)
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
	up, err := uploader.Upload(input, func(u *s3manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024
		u.Concurrency = 2
	})
	if err != nil {
		log.Println("upload:", err)
		return nil, err
	}
	return up, nil
}

func (s *s3Service) ListObject(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	out, err := s.svc.ListObjects(input)
	if err != nil {
		log.Println("list :", err)
		return nil, err
	}
	return out, nil
}

func (s *s3Service) MultipartUploads(input *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error) {
	out, err := s.svc.CreateMultipartUpload(input)
	if err != nil {
		log.Println("multipart upload:", err)
		return nil, err
	}
	return out, nil
}

func (s *s3Service) AbortMultipartUpload(input *s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error) {
	out, err := s.svc.AbortMultipartUpload(input)
	if err != nil {
		log.Println("abort multipart upload:", err)
		return nil, err
	}
	return out, nil
}

func (s *s3Service) CompleteMultipartUpload(input *s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error) {
	out, err := s.svc.CompleteMultipartUpload(input)
	if err != nil {
		log.Println("complete multipart upload:", err)
		return nil, err
	}
	return out, nil
}

func (s *s3Service) CreateMultipartUpload(input *s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error) {
	out, err := s.svc.CreateMultipartUpload(input)
	if err != nil {
		log.Println("Create multipart upload:", err)
		return nil, err
	}
	return out, nil

}

func (s *s3Service) UploadPart(input *s3.UploadPartInput) (*s3.UploadPartOutput, error) {
	out, err := s.svc.UploadPart(input)
	if err != nil {
		log.Println(" upload part:", err)
		return nil, err
	}
	return out, nil
}
