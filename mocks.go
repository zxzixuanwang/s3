package s3

import (
	"bytes"
	"errors"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var (
	ErrNoSuchBucket  = errors.New("NoSuchBucket: The specified bucket does not exist")
	ErrBucketExists  = errors.New("bucket already exists")
	ErrBucketHasKeys = errors.New("bucket has keys so cannot be deleted")
)

type MockBucket map[string][]byte

type MockS3 struct {
	sync.RWMutex
	// bucket: {key: value}
	data map[string]MockBucket
}

func NewMockS3() *MockS3 {
	return &MockS3{
		data: map[string]MockBucket{},
	}
}

func (ms *MockS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	ms.RLock()
	defer ms.RUnlock()
	buckets := []*s3.Bucket{}
	for name := range ms.data {
		bucket := s3.Bucket{Name: aws.String(name)}
		buckets = append(buckets, &bucket)
	}
	output := s3.ListBucketsOutput{
		Buckets: buckets,
	}
	return &output, nil
}

func (ms *MockS3) DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	ms.Lock()
	defer ms.Unlock()
	if bucket, exists := ms.data[*input.Bucket]; exists {
		if len(bucket) > 0 {
			return nil, ErrBucketHasKeys
		}
		delete(ms.data, *input.Bucket)
		return &s3.DeleteBucketOutput{}, nil
	} else {
		return nil, ErrNoSuchBucket
	}
}

func (ms *MockS3) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	ms.Lock()
	defer ms.Unlock()
	if _, exists := ms.data[*input.Bucket]; exists {
		return nil, ErrBucketExists
	}
	ms.data[*input.Bucket] = MockBucket{}
	return &s3.CreateBucketOutput{}, nil
}

func (ms *MockS3) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	ms.RLock()
	defer ms.RUnlock()
	bucket, ok := ms.data[*input.Bucket]
	if !ok {
		return nil, ErrNoSuchBucket
	}
	var keys []string
	for key := range bucket {
		if strings.HasPrefix(key, *input.Prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	contents := []*s3.Object{}
	for _, key := range keys {
		value := bucket[key]
		object := s3.Object{
			Key:  aws.String(key),
			Size: aws.Int64(int64(len(value))),
		}
		contents = append(contents, &object)
	}

	output := s3.ListObjectsOutput{
		Contents:    contents,
		IsTruncated: aws.Bool(false),
	}
	return &output, nil
}

func (ms *MockS3) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	ms.RLock()
	defer ms.RUnlock()
	bucket := ms.data[*input.Bucket]
	if object, ok := bucket[*input.Key]; ok {
		body := ioutil.NopCloser(bytes.NewReader(object))
		output := s3.GetObjectOutput{
			Body: body,
		}
		return &output, nil
	} else {
		return nil, errors.New("missing key")
	}
}

func (ms *MockS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	ms.Lock()
	defer ms.Unlock()
	content, _ := ioutil.ReadAll(input.Body)
	if bucket, ok := ms.data[*input.Bucket]; ok {
		bucket[*input.Key] = content
	} else {
		return nil, ErrNoSuchBucket
	}
	return &s3.PutObjectOutput{}, nil
}

func (ms *MockS3) PutObjectRequest(input *s3.PutObjectInput) (*request.Request, *s3.PutObjectOutput) {
	ms.Lock()
	defer ms.Unlock()
	// required for s3manager.Upload
	// TODO: should only alter bucket on Send()
	content, _ := ioutil.ReadAll(input.Body)
	req := request.New(aws.Config{}, metadata.ClientInfo{}, request.Handlers{}, nil, &request.Operation{}, nil, nil)
	if bucket, ok := ms.data[*input.Bucket]; ok {
		bucket[*input.Key] = content
	} else {
		// pre-set the error on the request
		req.Build()
		req.Error = ErrNoSuchBucket
	}
	return req, &s3.PutObjectOutput{}
}

func (ms *MockS3) DeleteObjects(input *s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	ms.Lock()
	defer ms.Unlock()
	bucket := ms.data[*input.Bucket]
	for _, id := range input.Delete.Objects {
		delete(bucket, *id.Key)
	}
	return &s3.DeleteObjectsOutput{}, nil
}

func (ms *MockS3) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	ms.Lock()
	defer ms.Unlock()
	bucket := ms.data[*input.Bucket]
	delete(bucket, *input.Key)
	return &s3.DeleteObjectOutput{}, nil
}

// unimplemented

func (ms *MockS3) AbortMultipartUploadRequest(*s3.AbortMultipartUploadInput) (*request.Request, *s3.AbortMultipartUploadOutput) {
	return nil, &s3.AbortMultipartUploadOutput{}
}
func (ms *MockS3) AbortMultipartUpload(*s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error) {
	return &s3.AbortMultipartUploadOutput{}, nil
}
func (ms *MockS3) CompleteMultipartUploadRequest(*s3.CompleteMultipartUploadInput) (*request.Request, *s3.CompleteMultipartUploadOutput) {
	return nil, &s3.CompleteMultipartUploadOutput{}
}
func (ms *MockS3) CompleteMultipartUpload(*s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error) {
	return &s3.CompleteMultipartUploadOutput{}, nil
}
func (ms *MockS3) CopyObjectRequest(*s3.CopyObjectInput) (*request.Request, *s3.CopyObjectOutput) {
	return nil, &s3.CopyObjectOutput{}
}
func (ms *MockS3) CopyObject(*s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	return &s3.CopyObjectOutput{}, nil
}
func (ms *MockS3) CreateBucketRequest(*s3.CreateBucketInput) (*request.Request, *s3.CreateBucketOutput) {
	return nil, &s3.CreateBucketOutput{}
}
func (ms *MockS3) CreateMultipartUploadRequest(*s3.CreateMultipartUploadInput) (*request.Request, *s3.CreateMultipartUploadOutput) {
	return nil, &s3.CreateMultipartUploadOutput{}
}
func (ms *MockS3) CreateMultipartUpload(*s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error) {
	return &s3.CreateMultipartUploadOutput{}, nil
}
func (ms *MockS3) DeleteBucketRequest(*s3.DeleteBucketInput) (*request.Request, *s3.DeleteBucketOutput) {
	return nil, &s3.DeleteBucketOutput{}
}
func (ms *MockS3) DeleteBucketCorsRequest(*s3.DeleteBucketCorsInput) (*request.Request, *s3.DeleteBucketCorsOutput) {
	return nil, &s3.DeleteBucketCorsOutput{}
}
func (ms *MockS3) DeleteBucketCors(*s3.DeleteBucketCorsInput) (*s3.DeleteBucketCorsOutput, error) {
	return &s3.DeleteBucketCorsOutput{}, nil
}
func (ms *MockS3) DeleteBucketLifecycleRequest(*s3.DeleteBucketLifecycleInput) (*request.Request, *s3.DeleteBucketLifecycleOutput) {
	return nil, &s3.DeleteBucketLifecycleOutput{}
}
func (ms *MockS3) DeleteBucketLifecycle(*s3.DeleteBucketLifecycleInput) (*s3.DeleteBucketLifecycleOutput, error) {
	return &s3.DeleteBucketLifecycleOutput{}, nil
}
func (ms *MockS3) DeleteBucketPolicyRequest(*s3.DeleteBucketPolicyInput) (*request.Request, *s3.DeleteBucketPolicyOutput) {
	return nil, &s3.DeleteBucketPolicyOutput{}
}
func (ms *MockS3) DeleteBucketPolicy(*s3.DeleteBucketPolicyInput) (*s3.DeleteBucketPolicyOutput, error) {
	return &s3.DeleteBucketPolicyOutput{}, nil
}
func (ms *MockS3) DeleteBucketReplicationRequest(*s3.DeleteBucketReplicationInput) (*request.Request, *s3.DeleteBucketReplicationOutput) {
	return nil, &s3.DeleteBucketReplicationOutput{}
}
func (ms *MockS3) DeleteBucketReplication(*s3.DeleteBucketReplicationInput) (*s3.DeleteBucketReplicationOutput, error) {
	return &s3.DeleteBucketReplicationOutput{}, nil
}
func (ms *MockS3) DeleteBucketTaggingRequest(*s3.DeleteBucketTaggingInput) (*request.Request, *s3.DeleteBucketTaggingOutput) {
	return nil, &s3.DeleteBucketTaggingOutput{}
}
func (ms *MockS3) DeleteBucketTagging(*s3.DeleteBucketTaggingInput) (*s3.DeleteBucketTaggingOutput, error) {
	return &s3.DeleteBucketTaggingOutput{}, nil
}
func (ms *MockS3) DeleteBucketWebsiteRequest(*s3.DeleteBucketWebsiteInput) (*request.Request, *s3.DeleteBucketWebsiteOutput) {
	return nil, &s3.DeleteBucketWebsiteOutput{}
}
func (ms *MockS3) DeleteBucketWebsite(*s3.DeleteBucketWebsiteInput) (*s3.DeleteBucketWebsiteOutput, error) {
	return &s3.DeleteBucketWebsiteOutput{}, nil
}
func (ms *MockS3) DeleteObjectRequest(*s3.DeleteObjectInput) (*request.Request, *s3.DeleteObjectOutput) {
	return nil, &s3.DeleteObjectOutput{}
}
func (ms *MockS3) DeleteObjectsRequest(*s3.DeleteObjectsInput) (*request.Request, *s3.DeleteObjectsOutput) {
	return nil, &s3.DeleteObjectsOutput{}
}
func (ms *MockS3) GetBucketAccelerateConfiguration(*s3.GetBucketAccelerateConfigurationInput) (*s3.GetBucketAccelerateConfigurationOutput, error) {
	return &s3.GetBucketAccelerateConfigurationOutput{}, nil
}
func (ms *MockS3) GetBucketAccelerateConfigurationRequest(*s3.GetBucketAccelerateConfigurationInput) (*request.Request, *s3.GetBucketAccelerateConfigurationOutput) {
	return nil, &s3.GetBucketAccelerateConfigurationOutput{}
}
func (ms *MockS3) GetBucketAclRequest(*s3.GetBucketAclInput) (*request.Request, *s3.GetBucketAclOutput) {
	return nil, &s3.GetBucketAclOutput{}
}
func (ms *MockS3) GetBucketAcl(*s3.GetBucketAclInput) (*s3.GetBucketAclOutput, error) {
	return &s3.GetBucketAclOutput{}, nil
}
func (ms *MockS3) GetBucketCorsRequest(*s3.GetBucketCorsInput) (*request.Request, *s3.GetBucketCorsOutput) {
	return nil, &s3.GetBucketCorsOutput{}
}
func (ms *MockS3) GetBucketCors(*s3.GetBucketCorsInput) (*s3.GetBucketCorsOutput, error) {
	return &s3.GetBucketCorsOutput{}, nil
}
func (ms *MockS3) GetBucketLifecycleRequest(*s3.GetBucketLifecycleInput) (*request.Request, *s3.GetBucketLifecycleOutput) {
	return nil, &s3.GetBucketLifecycleOutput{}
}
func (ms *MockS3) GetBucketLifecycle(*s3.GetBucketLifecycleInput) (*s3.GetBucketLifecycleOutput, error) {
	return &s3.GetBucketLifecycleOutput{}, nil
}
func (ms *MockS3) GetBucketLifecycleConfigurationRequest(*s3.GetBucketLifecycleConfigurationInput) (*request.Request, *s3.GetBucketLifecycleConfigurationOutput) {
	return nil, &s3.GetBucketLifecycleConfigurationOutput{}
}
func (ms *MockS3) GetBucketLifecycleConfiguration(*s3.GetBucketLifecycleConfigurationInput) (*s3.GetBucketLifecycleConfigurationOutput, error) {
	return &s3.GetBucketLifecycleConfigurationOutput{}, nil
}
func (ms *MockS3) GetBucketLocationRequest(*s3.GetBucketLocationInput) (*request.Request, *s3.GetBucketLocationOutput) {
	return nil, &s3.GetBucketLocationOutput{}
}
func (ms *MockS3) GetBucketLocation(*s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return &s3.GetBucketLocationOutput{}, nil
}
func (ms *MockS3) GetBucketLoggingRequest(*s3.GetBucketLoggingInput) (*request.Request, *s3.GetBucketLoggingOutput) {
	return nil, &s3.GetBucketLoggingOutput{}
}
func (ms *MockS3) GetBucketLogging(*s3.GetBucketLoggingInput) (*s3.GetBucketLoggingOutput, error) {
	return &s3.GetBucketLoggingOutput{}, nil
}
func (ms *MockS3) GetBucketNotificationRequest(*s3.GetBucketNotificationConfigurationRequest) (*request.Request, *s3.NotificationConfigurationDeprecated) {
	return nil, &s3.NotificationConfigurationDeprecated{}
}
func (ms *MockS3) GetBucketNotification(*s3.GetBucketNotificationConfigurationRequest) (*s3.NotificationConfigurationDeprecated, error) {
	return &s3.NotificationConfigurationDeprecated{}, nil
}
func (ms *MockS3) GetBucketNotificationConfigurationRequest(*s3.GetBucketNotificationConfigurationRequest) (*request.Request, *s3.NotificationConfiguration) {
	return nil, &s3.NotificationConfiguration{}
}
func (ms *MockS3) GetBucketNotificationConfiguration(*s3.GetBucketNotificationConfigurationRequest) (*s3.NotificationConfiguration, error) {
	return &s3.NotificationConfiguration{}, nil
}
func (ms *MockS3) GetBucketPolicyRequest(*s3.GetBucketPolicyInput) (*request.Request, *s3.GetBucketPolicyOutput) {
	return nil, &s3.GetBucketPolicyOutput{}
}
func (ms *MockS3) GetBucketPolicy(*s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
	return &s3.GetBucketPolicyOutput{}, nil
}
func (ms *MockS3) GetBucketReplicationRequest(*s3.GetBucketReplicationInput) (*request.Request, *s3.GetBucketReplicationOutput) {
	return nil, &s3.GetBucketReplicationOutput{}
}
func (ms *MockS3) GetBucketReplication(*s3.GetBucketReplicationInput) (*s3.GetBucketReplicationOutput, error) {
	return &s3.GetBucketReplicationOutput{}, nil
}
func (ms *MockS3) GetBucketRequestPaymentRequest(*s3.GetBucketRequestPaymentInput) (*request.Request, *s3.GetBucketRequestPaymentOutput) {
	return nil, &s3.GetBucketRequestPaymentOutput{}
}
func (ms *MockS3) GetBucketRequestPayment(*s3.GetBucketRequestPaymentInput) (*s3.GetBucketRequestPaymentOutput, error) {
	return &s3.GetBucketRequestPaymentOutput{}, nil
}
func (ms *MockS3) GetBucketTaggingRequest(*s3.GetBucketTaggingInput) (*request.Request, *s3.GetBucketTaggingOutput) {
	return nil, &s3.GetBucketTaggingOutput{}
}
func (ms *MockS3) GetBucketTagging(*s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	return &s3.GetBucketTaggingOutput{}, nil
}
func (ms *MockS3) GetBucketVersioningRequest(*s3.GetBucketVersioningInput) (*request.Request, *s3.GetBucketVersioningOutput) {
	return nil, &s3.GetBucketVersioningOutput{}
}
func (ms *MockS3) GetBucketVersioning(*s3.GetBucketVersioningInput) (*s3.GetBucketVersioningOutput, error) {
	return &s3.GetBucketVersioningOutput{}, nil
}
func (ms *MockS3) GetBucketWebsiteRequest(*s3.GetBucketWebsiteInput) (*request.Request, *s3.GetBucketWebsiteOutput) {
	return nil, &s3.GetBucketWebsiteOutput{}
}
func (ms *MockS3) GetBucketWebsite(*s3.GetBucketWebsiteInput) (*s3.GetBucketWebsiteOutput, error) {
	return &s3.GetBucketWebsiteOutput{}, nil
}
func (ms *MockS3) GetObjectRequest(*s3.GetObjectInput) (*request.Request, *s3.GetObjectOutput) {
	return nil, &s3.GetObjectOutput{}
}
func (ms *MockS3) GetObjectAclRequest(*s3.GetObjectAclInput) (*request.Request, *s3.GetObjectAclOutput) {
	return nil, &s3.GetObjectAclOutput{}
}
func (ms *MockS3) GetObjectAcl(*s3.GetObjectAclInput) (*s3.GetObjectAclOutput, error) {
	return &s3.GetObjectAclOutput{}, nil
}
func (ms *MockS3) GetObjectTorrentRequest(*s3.GetObjectTorrentInput) (*request.Request, *s3.GetObjectTorrentOutput) {
	return nil, &s3.GetObjectTorrentOutput{}
}
func (ms *MockS3) GetObjectTorrent(*s3.GetObjectTorrentInput) (*s3.GetObjectTorrentOutput, error) {
	return &s3.GetObjectTorrentOutput{}, nil
}
func (ms *MockS3) HeadBucketRequest(*s3.HeadBucketInput) (*request.Request, *s3.HeadBucketOutput) {
	return nil, &s3.HeadBucketOutput{}
}
func (ms *MockS3) HeadBucket(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}
func (ms *MockS3) HeadObjectRequest(*s3.HeadObjectInput) (*request.Request, *s3.HeadObjectOutput) {
	return nil, &s3.HeadObjectOutput{}
}
func (ms *MockS3) HeadObject(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{}, nil
}
func (ms *MockS3) ListBucketsRequest(*s3.ListBucketsInput) (*request.Request, *s3.ListBucketsOutput) {
	return nil, &s3.ListBucketsOutput{}
}
func (ms *MockS3) ListMultipartUploadsRequest(*s3.ListMultipartUploadsInput) (*request.Request, *s3.ListMultipartUploadsOutput) {
	return nil, &s3.ListMultipartUploadsOutput{}
}
func (ms *MockS3) ListMultipartUploads(*s3.ListMultipartUploadsInput) (*s3.ListMultipartUploadsOutput, error) {
	return &s3.ListMultipartUploadsOutput{}, nil
}
func (ms *MockS3) ListMultipartUploadsPages(*s3.ListMultipartUploadsInput, func(*s3.ListMultipartUploadsOutput, bool) bool) error {
	return nil
}
func (ms *MockS3) ListObjectVersionsRequest(*s3.ListObjectVersionsInput) (*request.Request, *s3.ListObjectVersionsOutput) {
	return nil, &s3.ListObjectVersionsOutput{}
}
func (ms *MockS3) ListObjectVersions(*s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
	return &s3.ListObjectVersionsOutput{}, nil
}
func (ms *MockS3) ListObjectVersionsPages(*s3.ListObjectVersionsInput, func(*s3.ListObjectVersionsOutput, bool) bool) error {
	return nil
}
func (ms *MockS3) ListObjectsRequest(*s3.ListObjectsInput) (*request.Request, *s3.ListObjectsOutput) {
	return nil, &s3.ListObjectsOutput{}
}
func (ms *MockS3) ListObjectsPages(*s3.ListObjectsInput, func(*s3.ListObjectsOutput, bool) bool) error {
	return nil
}
func (ms *MockS3) ListObjectsV2Pages(*s3.ListObjectsV2Input, func(*s3.ListObjectsV2Output, bool) bool) error {
	return nil
}
func (ms *MockS3) ListObjectsV2(*s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	return nil, nil
}
func (ms *MockS3) ListObjectsV2Request(*s3.ListObjectsV2Input) (*request.Request, *s3.ListObjectsV2Output) {
	return nil, nil
}
func (ms *MockS3) ListPartsRequest(*s3.ListPartsInput) (*request.Request, *s3.ListPartsOutput) {
	return nil, &s3.ListPartsOutput{}
}
func (ms *MockS3) ListParts(*s3.ListPartsInput) (*s3.ListPartsOutput, error) {
	return &s3.ListPartsOutput{}, nil
}
func (ms *MockS3) ListPartsPages(*s3.ListPartsInput, func(*s3.ListPartsOutput, bool) bool) error {
	return nil
}
func (ms *MockS3) PutBucketAclRequest(*s3.PutBucketAclInput) (*request.Request, *s3.PutBucketAclOutput) {
	return nil, &s3.PutBucketAclOutput{}
}
func (ms *MockS3) PutBucketAccelerateConfiguration(*s3.PutBucketAccelerateConfigurationInput) (*s3.PutBucketAccelerateConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketAccelerateConfigurationRequest(*s3.PutBucketAccelerateConfigurationInput) (*request.Request, *s3.PutBucketAccelerateConfigurationOutput) {
	return nil, &s3.PutBucketAccelerateConfigurationOutput{}
}
func (ms *MockS3) PutBucketAcl(*s3.PutBucketAclInput) (*s3.PutBucketAclOutput, error) {
	return &s3.PutBucketAclOutput{}, nil
}
func (ms *MockS3) PutBucketCorsRequest(*s3.PutBucketCorsInput) (*request.Request, *s3.PutBucketCorsOutput) {
	return nil, &s3.PutBucketCorsOutput{}
}
func (ms *MockS3) PutBucketCors(*s3.PutBucketCorsInput) (*s3.PutBucketCorsOutput, error) {
	return &s3.PutBucketCorsOutput{}, nil
}
func (ms *MockS3) PutBucketLifecycleRequest(*s3.PutBucketLifecycleInput) (*request.Request, *s3.PutBucketLifecycleOutput) {
	return nil, &s3.PutBucketLifecycleOutput{}
}
func (ms *MockS3) PutBucketLifecycle(*s3.PutBucketLifecycleInput) (*s3.PutBucketLifecycleOutput, error) {
	return &s3.PutBucketLifecycleOutput{}, nil
}
func (ms *MockS3) PutBucketLifecycleConfigurationRequest(*s3.PutBucketLifecycleConfigurationInput) (*request.Request, *s3.PutBucketLifecycleConfigurationOutput) {
	return nil, &s3.PutBucketLifecycleConfigurationOutput{}
}
func (ms *MockS3) PutBucketLifecycleConfiguration(*s3.PutBucketLifecycleConfigurationInput) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	return &s3.PutBucketLifecycleConfigurationOutput{}, nil
}
func (ms *MockS3) PutBucketLoggingRequest(*s3.PutBucketLoggingInput) (*request.Request, *s3.PutBucketLoggingOutput) {
	return nil, &s3.PutBucketLoggingOutput{}
}
func (ms *MockS3) PutBucketLogging(*s3.PutBucketLoggingInput) (*s3.PutBucketLoggingOutput, error) {
	return &s3.PutBucketLoggingOutput{}, nil
}
func (ms *MockS3) PutBucketNotificationRequest(*s3.PutBucketNotificationInput) (*request.Request, *s3.PutBucketNotificationOutput) {
	return nil, &s3.PutBucketNotificationOutput{}
}
func (ms *MockS3) PutBucketNotification(*s3.PutBucketNotificationInput) (*s3.PutBucketNotificationOutput, error) {
	return &s3.PutBucketNotificationOutput{}, nil
}
func (ms *MockS3) PutBucketNotificationConfigurationRequest(*s3.PutBucketNotificationConfigurationInput) (*request.Request, *s3.PutBucketNotificationConfigurationOutput) {
	return nil, &s3.PutBucketNotificationConfigurationOutput{}
}
func (ms *MockS3) PutBucketNotificationConfiguration(*s3.PutBucketNotificationConfigurationInput) (*s3.PutBucketNotificationConfigurationOutput, error) {
	return &s3.PutBucketNotificationConfigurationOutput{}, nil
}
func (ms *MockS3) PutBucketPolicyRequest(*s3.PutBucketPolicyInput) (*request.Request, *s3.PutBucketPolicyOutput) {
	return nil, &s3.PutBucketPolicyOutput{}
}
func (ms *MockS3) PutBucketPolicy(*s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
	return &s3.PutBucketPolicyOutput{}, nil
}
func (ms *MockS3) PutBucketReplicationRequest(*s3.PutBucketReplicationInput) (*request.Request, *s3.PutBucketReplicationOutput) {
	return nil, &s3.PutBucketReplicationOutput{}
}
func (ms *MockS3) PutBucketReplication(*s3.PutBucketReplicationInput) (*s3.PutBucketReplicationOutput, error) {
	return &s3.PutBucketReplicationOutput{}, nil
}
func (ms *MockS3) PutBucketRequestPaymentRequest(*s3.PutBucketRequestPaymentInput) (*request.Request, *s3.PutBucketRequestPaymentOutput) {
	return nil, &s3.PutBucketRequestPaymentOutput{}
}
func (ms *MockS3) PutBucketRequestPayment(*s3.PutBucketRequestPaymentInput) (*s3.PutBucketRequestPaymentOutput, error) {
	return &s3.PutBucketRequestPaymentOutput{}, nil
}
func (ms *MockS3) PutBucketTaggingRequest(*s3.PutBucketTaggingInput) (*request.Request, *s3.PutBucketTaggingOutput) {
	return nil, &s3.PutBucketTaggingOutput{}
}
func (ms *MockS3) PutBucketTagging(*s3.PutBucketTaggingInput) (*s3.PutBucketTaggingOutput, error) {
	return &s3.PutBucketTaggingOutput{}, nil
}
func (ms *MockS3) PutBucketVersioningRequest(*s3.PutBucketVersioningInput) (*request.Request, *s3.PutBucketVersioningOutput) {
	return nil, &s3.PutBucketVersioningOutput{}
}
func (ms *MockS3) PutBucketVersioning(*s3.PutBucketVersioningInput) (*s3.PutBucketVersioningOutput, error) {
	return &s3.PutBucketVersioningOutput{}, nil
}
func (ms *MockS3) PutBucketWebsiteRequest(*s3.PutBucketWebsiteInput) (*request.Request, *s3.PutBucketWebsiteOutput) {
	return nil, &s3.PutBucketWebsiteOutput{}
}
func (ms *MockS3) PutBucketWebsite(*s3.PutBucketWebsiteInput) (*s3.PutBucketWebsiteOutput, error) {
	return &s3.PutBucketWebsiteOutput{}, nil
}
func (ms *MockS3) PutObjectAclRequest(*s3.PutObjectAclInput) (*request.Request, *s3.PutObjectAclOutput) {
	return nil, &s3.PutObjectAclOutput{}
}
func (ms *MockS3) PutObjectAcl(*s3.PutObjectAclInput) (*s3.PutObjectAclOutput, error) {
	return &s3.PutObjectAclOutput{}, nil
}
func (ms *MockS3) RestoreObjectRequest(*s3.RestoreObjectInput) (*request.Request, *s3.RestoreObjectOutput) {
	return nil, &s3.RestoreObjectOutput{}
}
func (ms *MockS3) RestoreObject(*s3.RestoreObjectInput) (*s3.RestoreObjectOutput, error) {
	return &s3.RestoreObjectOutput{}, nil
}
func (ms *MockS3) UploadPartRequest(*s3.UploadPartInput) (*request.Request, *s3.UploadPartOutput) {
	return nil, &s3.UploadPartOutput{}
}
func (ms *MockS3) UploadPart(*s3.UploadPartInput) (*s3.UploadPartOutput, error) {
	return &s3.UploadPartOutput{}, nil
}
func (ms *MockS3) UploadPartCopyRequest(*s3.UploadPartCopyInput) (*request.Request, *s3.UploadPartCopyOutput) {
	return nil, &s3.UploadPartCopyOutput{}
}
func (ms *MockS3) UploadPartCopy(*s3.UploadPartCopyInput) (*s3.UploadPartCopyOutput, error) {
	return &s3.UploadPartCopyOutput{}, nil
}

func (ms *MockS3) AbortMultipartUploadWithContext(aws.Context, *s3.AbortMultipartUploadInput, ...request.Option) (*s3.AbortMultipartUploadOutput, error) {
	return &s3.AbortMultipartUploadOutput{}, nil
}

func (ms *MockS3) CompleteMultipartUploadWithContext(aws.Context, *s3.CompleteMultipartUploadInput, ...request.Option) (*s3.CompleteMultipartUploadOutput, error) {
	return &s3.CompleteMultipartUploadOutput{}, nil
}

func (ms *MockS3) CopyObjectWithContext(aws.Context, *s3.CopyObjectInput, ...request.Option) (*s3.CopyObjectOutput, error) {
	return &s3.CopyObjectOutput{}, nil
}

func (ms *MockS3) CreateBucketWithContext(aws.Context, *s3.CreateBucketInput, ...request.Option) (*s3.CreateBucketOutput, error) {
	return &s3.CreateBucketOutput{}, nil
}

func (ms *MockS3) CreateMultipartUploadWithContext(aws.Context, *s3.CreateMultipartUploadInput, ...request.Option) (*s3.CreateMultipartUploadOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketWithContext(aws.Context, *s3.DeleteBucketInput, ...request.Option) (*s3.DeleteBucketOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketAnalyticsConfiguration(*s3.DeleteBucketAnalyticsConfigurationInput) (*s3.DeleteBucketAnalyticsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketAnalyticsConfigurationWithContext(aws.Context, *s3.DeleteBucketAnalyticsConfigurationInput, ...request.Option) (*s3.DeleteBucketAnalyticsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketAnalyticsConfigurationRequest(*s3.DeleteBucketAnalyticsConfigurationInput) (*request.Request, *s3.DeleteBucketAnalyticsConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketCorsWithContext(aws.Context, *s3.DeleteBucketCorsInput, ...request.Option) (*s3.DeleteBucketCorsOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketEncryption(*s3.DeleteBucketEncryptionInput) (*s3.DeleteBucketEncryptionOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketEncryptionWithContext(aws.Context, *s3.DeleteBucketEncryptionInput, ...request.Option) (*s3.DeleteBucketEncryptionOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketEncryptionRequest(*s3.DeleteBucketEncryptionInput) (*request.Request, *s3.DeleteBucketEncryptionOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketIntelligentTieringConfiguration(*s3.DeleteBucketIntelligentTieringConfigurationInput) (*s3.DeleteBucketIntelligentTieringConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketIntelligentTieringConfigurationWithContext(aws.Context, *s3.DeleteBucketIntelligentTieringConfigurationInput, ...request.Option) (*s3.DeleteBucketIntelligentTieringConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketIntelligentTieringConfigurationRequest(*s3.DeleteBucketIntelligentTieringConfigurationInput) (*request.Request, *s3.DeleteBucketIntelligentTieringConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketInventoryConfiguration(*s3.DeleteBucketInventoryConfigurationInput) (*s3.DeleteBucketInventoryConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketInventoryConfigurationWithContext(aws.Context, *s3.DeleteBucketInventoryConfigurationInput, ...request.Option) (*s3.DeleteBucketInventoryConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketInventoryConfigurationRequest(*s3.DeleteBucketInventoryConfigurationInput) (*request.Request, *s3.DeleteBucketInventoryConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketLifecycleWithContext(aws.Context, *s3.DeleteBucketLifecycleInput, ...request.Option) (*s3.DeleteBucketLifecycleOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketMetricsConfiguration(*s3.DeleteBucketMetricsConfigurationInput) (*s3.DeleteBucketMetricsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketMetricsConfigurationWithContext(aws.Context, *s3.DeleteBucketMetricsConfigurationInput, ...request.Option) (*s3.DeleteBucketMetricsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketMetricsConfigurationRequest(*s3.DeleteBucketMetricsConfigurationInput) (*request.Request, *s3.DeleteBucketMetricsConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketOwnershipControls(*s3.DeleteBucketOwnershipControlsInput) (*s3.DeleteBucketOwnershipControlsOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketOwnershipControlsWithContext(aws.Context, *s3.DeleteBucketOwnershipControlsInput, ...request.Option) (*s3.DeleteBucketOwnershipControlsOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteBucketOwnershipControlsRequest(*s3.DeleteBucketOwnershipControlsInput) (*request.Request, *s3.DeleteBucketOwnershipControlsOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketPolicyWithContext(aws.Context, *s3.DeleteBucketPolicyInput, ...request.Option) (*s3.DeleteBucketPolicyOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketReplicationWithContext(aws.Context, *s3.DeleteBucketReplicationInput, ...request.Option) (*s3.DeleteBucketReplicationOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketTaggingWithContext(aws.Context, *s3.DeleteBucketTaggingInput, ...request.Option) (*s3.DeleteBucketTaggingOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteBucketWebsiteWithContext(aws.Context, *s3.DeleteBucketWebsiteInput, ...request.Option) (*s3.DeleteBucketWebsiteOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteObjectWithContext(aws.Context, *s3.DeleteObjectInput, ...request.Option) (*s3.DeleteObjectOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeleteObjectTagging(*s3.DeleteObjectTaggingInput) (*s3.DeleteObjectTaggingOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteObjectTaggingWithContext(aws.Context, *s3.DeleteObjectTaggingInput, ...request.Option) (*s3.DeleteObjectTaggingOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeleteObjectTaggingRequest(*s3.DeleteObjectTaggingInput) (*request.Request, *s3.DeleteObjectTaggingOutput) {
	return nil, nil
}

func (ms *MockS3) DeleteObjectsWithContext(aws.Context, *s3.DeleteObjectsInput, ...request.Option) (*s3.DeleteObjectsOutput, error) {
	return nil, nil
}

func (ms *MockS3) DeletePublicAccessBlock(*s3.DeletePublicAccessBlockInput) (*s3.DeletePublicAccessBlockOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeletePublicAccessBlockWithContext(aws.Context, *s3.DeletePublicAccessBlockInput, ...request.Option) (*s3.DeletePublicAccessBlockOutput, error) {
	return nil, nil
}
func (ms *MockS3) DeletePublicAccessBlockRequest(*s3.DeletePublicAccessBlockInput) (*request.Request, *s3.DeletePublicAccessBlockOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketAccelerateConfigurationWithContext(aws.Context, *s3.GetBucketAccelerateConfigurationInput, ...request.Option) (*s3.GetBucketAccelerateConfigurationOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketAclWithContext(aws.Context, *s3.GetBucketAclInput, ...request.Option) (*s3.GetBucketAclOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketAnalyticsConfiguration(*s3.GetBucketAnalyticsConfigurationInput) (*s3.GetBucketAnalyticsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketAnalyticsConfigurationWithContext(aws.Context, *s3.GetBucketAnalyticsConfigurationInput, ...request.Option) (*s3.GetBucketAnalyticsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketAnalyticsConfigurationRequest(*s3.GetBucketAnalyticsConfigurationInput) (*request.Request, *s3.GetBucketAnalyticsConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketCorsWithContext(aws.Context, *s3.GetBucketCorsInput, ...request.Option) (*s3.GetBucketCorsOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketEncryption(*s3.GetBucketEncryptionInput) (*s3.GetBucketEncryptionOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketEncryptionWithContext(aws.Context, *s3.GetBucketEncryptionInput, ...request.Option) (*s3.GetBucketEncryptionOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketEncryptionRequest(*s3.GetBucketEncryptionInput) (*request.Request, *s3.GetBucketEncryptionOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketIntelligentTieringConfiguration(*s3.GetBucketIntelligentTieringConfigurationInput) (*s3.GetBucketIntelligentTieringConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketIntelligentTieringConfigurationWithContext(aws.Context, *s3.GetBucketIntelligentTieringConfigurationInput, ...request.Option) (*s3.GetBucketIntelligentTieringConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketIntelligentTieringConfigurationRequest(*s3.GetBucketIntelligentTieringConfigurationInput) (*request.Request, *s3.GetBucketIntelligentTieringConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketInventoryConfiguration(*s3.GetBucketInventoryConfigurationInput) (*s3.GetBucketInventoryConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketInventoryConfigurationWithContext(aws.Context, *s3.GetBucketInventoryConfigurationInput, ...request.Option) (*s3.GetBucketInventoryConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketInventoryConfigurationRequest(*s3.GetBucketInventoryConfigurationInput) (*request.Request, *s3.GetBucketInventoryConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketLifecycleWithContext(aws.Context, *s3.GetBucketLifecycleInput, ...request.Option) (*s3.GetBucketLifecycleOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketLifecycleConfigurationWithContext(aws.Context, *s3.GetBucketLifecycleConfigurationInput, ...request.Option) (*s3.GetBucketLifecycleConfigurationOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketLocationWithContext(aws.Context, *s3.GetBucketLocationInput, ...request.Option) (*s3.GetBucketLocationOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketLoggingWithContext(aws.Context, *s3.GetBucketLoggingInput, ...request.Option) (*s3.GetBucketLoggingOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketMetricsConfiguration(*s3.GetBucketMetricsConfigurationInput) (*s3.GetBucketMetricsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketMetricsConfigurationWithContext(aws.Context, *s3.GetBucketMetricsConfigurationInput, ...request.Option) (*s3.GetBucketMetricsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketMetricsConfigurationRequest(*s3.GetBucketMetricsConfigurationInput) (*request.Request, *s3.GetBucketMetricsConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketNotificationWithContext(aws.Context, *s3.GetBucketNotificationConfigurationRequest, ...request.Option) (*s3.NotificationConfigurationDeprecated, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketNotificationConfigurationWithContext(aws.Context, *s3.GetBucketNotificationConfigurationRequest, ...request.Option) (*s3.NotificationConfiguration, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketOwnershipControls(*s3.GetBucketOwnershipControlsInput) (*s3.GetBucketOwnershipControlsOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketOwnershipControlsWithContext(aws.Context, *s3.GetBucketOwnershipControlsInput, ...request.Option) (*s3.GetBucketOwnershipControlsOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketOwnershipControlsRequest(*s3.GetBucketOwnershipControlsInput) (*request.Request, *s3.GetBucketOwnershipControlsOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketPolicyWithContext(aws.Context, *s3.GetBucketPolicyInput, ...request.Option) (*s3.GetBucketPolicyOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketPolicyStatus(*s3.GetBucketPolicyStatusInput) (*s3.GetBucketPolicyStatusOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketPolicyStatusWithContext(aws.Context, *s3.GetBucketPolicyStatusInput, ...request.Option) (*s3.GetBucketPolicyStatusOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetBucketPolicyStatusRequest(*s3.GetBucketPolicyStatusInput) (*request.Request, *s3.GetBucketPolicyStatusOutput) {
	return nil, nil
}

func (ms *MockS3) GetBucketReplicationWithContext(aws.Context, *s3.GetBucketReplicationInput, ...request.Option) (*s3.GetBucketReplicationOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketRequestPaymentWithContext(aws.Context, *s3.GetBucketRequestPaymentInput, ...request.Option) (*s3.GetBucketRequestPaymentOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketTaggingWithContext(aws.Context, *s3.GetBucketTaggingInput, ...request.Option) (*s3.GetBucketTaggingOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketVersioningWithContext(aws.Context, *s3.GetBucketVersioningInput, ...request.Option) (*s3.GetBucketVersioningOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetBucketWebsiteWithContext(aws.Context, *s3.GetBucketWebsiteInput, ...request.Option) (*s3.GetBucketWebsiteOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetObjectWithContext(aws.Context, *s3.GetObjectInput, ...request.Option) (*s3.GetObjectOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetObjectAclWithContext(aws.Context, *s3.GetObjectAclInput, ...request.Option) (*s3.GetObjectAclOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetObjectLegalHold(*s3.GetObjectLegalHoldInput) (*s3.GetObjectLegalHoldOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectLegalHoldWithContext(aws.Context, *s3.GetObjectLegalHoldInput, ...request.Option) (*s3.GetObjectLegalHoldOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectLegalHoldRequest(*s3.GetObjectLegalHoldInput) (*request.Request, *s3.GetObjectLegalHoldOutput) {
	return nil, nil
}

func (ms *MockS3) GetObjectLockConfiguration(*s3.GetObjectLockConfigurationInput) (*s3.GetObjectLockConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectLockConfigurationWithContext(aws.Context, *s3.GetObjectLockConfigurationInput, ...request.Option) (*s3.GetObjectLockConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectLockConfigurationRequest(*s3.GetObjectLockConfigurationInput) (*request.Request, *s3.GetObjectLockConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) GetObjectRetention(*s3.GetObjectRetentionInput) (*s3.GetObjectRetentionOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectRetentionWithContext(aws.Context, *s3.GetObjectRetentionInput, ...request.Option) (*s3.GetObjectRetentionOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectRetentionRequest(*s3.GetObjectRetentionInput) (*request.Request, *s3.GetObjectRetentionOutput) {
	return nil, nil
}

func (ms *MockS3) GetObjectTagging(*s3.GetObjectTaggingInput) (*s3.GetObjectTaggingOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectTaggingWithContext(aws.Context, *s3.GetObjectTaggingInput, ...request.Option) (*s3.GetObjectTaggingOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetObjectTaggingRequest(*s3.GetObjectTaggingInput) (*request.Request, *s3.GetObjectTaggingOutput) {
	return nil, nil
}

func (ms *MockS3) GetObjectTorrentWithContext(aws.Context, *s3.GetObjectTorrentInput, ...request.Option) (*s3.GetObjectTorrentOutput, error) {
	return nil, nil
}

func (ms *MockS3) GetPublicAccessBlock(*s3.GetPublicAccessBlockInput) (*s3.GetPublicAccessBlockOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetPublicAccessBlockWithContext(aws.Context, *s3.GetPublicAccessBlockInput, ...request.Option) (*s3.GetPublicAccessBlockOutput, error) {
	return nil, nil
}
func (ms *MockS3) GetPublicAccessBlockRequest(*s3.GetPublicAccessBlockInput) (*request.Request, *s3.GetPublicAccessBlockOutput) {
	return nil, nil
}

func (ms *MockS3) HeadBucketWithContext(aws.Context, *s3.HeadBucketInput, ...request.Option) (*s3.HeadBucketOutput, error) {
	return nil, nil
}

func (ms *MockS3) HeadObjectWithContext(aws.Context, *s3.HeadObjectInput, ...request.Option) (*s3.HeadObjectOutput, error) {
	return nil, nil
}

func (ms *MockS3) ListBucketAnalyticsConfigurations(*s3.ListBucketAnalyticsConfigurationsInput) (*s3.ListBucketAnalyticsConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketAnalyticsConfigurationsWithContext(aws.Context, *s3.ListBucketAnalyticsConfigurationsInput, ...request.Option) (*s3.ListBucketAnalyticsConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketAnalyticsConfigurationsRequest(*s3.ListBucketAnalyticsConfigurationsInput) (*request.Request, *s3.ListBucketAnalyticsConfigurationsOutput) {
	return nil, nil
}

func (ms *MockS3) ListBucketIntelligentTieringConfigurations(*s3.ListBucketIntelligentTieringConfigurationsInput) (*s3.ListBucketIntelligentTieringConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketIntelligentTieringConfigurationsWithContext(aws.Context, *s3.ListBucketIntelligentTieringConfigurationsInput, ...request.Option) (*s3.ListBucketIntelligentTieringConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketIntelligentTieringConfigurationsRequest(*s3.ListBucketIntelligentTieringConfigurationsInput) (*request.Request, *s3.ListBucketIntelligentTieringConfigurationsOutput) {
	return nil, nil
}

func (ms *MockS3) ListBucketInventoryConfigurations(*s3.ListBucketInventoryConfigurationsInput) (*s3.ListBucketInventoryConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketInventoryConfigurationsWithContext(aws.Context, *s3.ListBucketInventoryConfigurationsInput, ...request.Option) (*s3.ListBucketInventoryConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketInventoryConfigurationsRequest(*s3.ListBucketInventoryConfigurationsInput) (*request.Request, *s3.ListBucketInventoryConfigurationsOutput) {
	return nil, nil
}

func (ms *MockS3) ListBucketMetricsConfigurations(*s3.ListBucketMetricsConfigurationsInput) (*s3.ListBucketMetricsConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketMetricsConfigurationsWithContext(aws.Context, *s3.ListBucketMetricsConfigurationsInput, ...request.Option) (*s3.ListBucketMetricsConfigurationsOutput, error) {
	return nil, nil
}
func (ms *MockS3) ListBucketMetricsConfigurationsRequest(*s3.ListBucketMetricsConfigurationsInput) (*request.Request, *s3.ListBucketMetricsConfigurationsOutput) {
	return nil, nil
}

func (ms *MockS3) ListBucketsWithContext(aws.Context, *s3.ListBucketsInput, ...request.Option) (*s3.ListBucketsOutput, error) {
	return nil, nil
}

func (ms *MockS3) ListMultipartUploadsWithContext(aws.Context, *s3.ListMultipartUploadsInput, ...request.Option) (*s3.ListMultipartUploadsOutput, error) {
	return nil, nil
}

func (ms *MockS3) ListMultipartUploadsPagesWithContext(aws.Context, *s3.ListMultipartUploadsInput, func(*s3.ListMultipartUploadsOutput, bool) bool, ...request.Option) error {
	return nil
}

func (ms *MockS3) ListObjectVersionsWithContext(aws.Context, *s3.ListObjectVersionsInput, ...request.Option) (*s3.ListObjectVersionsOutput, error) {
	return nil, nil
}

func (ms *MockS3) ListObjectVersionsPagesWithContext(aws.Context, *s3.ListObjectVersionsInput, func(*s3.ListObjectVersionsOutput, bool) bool, ...request.Option) error {
	return nil
}

func (ms *MockS3) ListObjectsWithContext(aws.Context, *s3.ListObjectsInput, ...request.Option) (*s3.ListObjectsOutput, error) {
	return nil, nil
}

func (ms *MockS3) ListObjectsPagesWithContext(aws.Context, *s3.ListObjectsInput, func(*s3.ListObjectsOutput, bool) bool, ...request.Option) error {
	return nil
}

func (ms *MockS3) ListObjectsV2WithContext(aws.Context, *s3.ListObjectsV2Input, ...request.Option) (*s3.ListObjectsV2Output, error) {
	return nil, nil
}

func (ms *MockS3) ListObjectsV2PagesWithContext(aws.Context, *s3.ListObjectsV2Input, func(*s3.ListObjectsV2Output, bool) bool, ...request.Option) error {
	return nil
}

func (ms *MockS3) ListPartsWithContext(aws.Context, *s3.ListPartsInput, ...request.Option) (*s3.ListPartsOutput, error) {
	return nil, nil
}

func (ms *MockS3) ListPartsPagesWithContext(aws.Context, *s3.ListPartsInput, func(*s3.ListPartsOutput, bool) bool, ...request.Option) error {
	return nil
}

func (ms *MockS3) PutBucketAccelerateConfigurationWithContext(aws.Context, *s3.PutBucketAccelerateConfigurationInput, ...request.Option) (*s3.PutBucketAccelerateConfigurationOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketAclWithContext(aws.Context, *s3.PutBucketAclInput, ...request.Option) (*s3.PutBucketAclOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketAnalyticsConfiguration(*s3.PutBucketAnalyticsConfigurationInput) (*s3.PutBucketAnalyticsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketAnalyticsConfigurationWithContext(aws.Context, *s3.PutBucketAnalyticsConfigurationInput, ...request.Option) (*s3.PutBucketAnalyticsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketAnalyticsConfigurationRequest(*s3.PutBucketAnalyticsConfigurationInput) (*request.Request, *s3.PutBucketAnalyticsConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) PutBucketCorsWithContext(aws.Context, *s3.PutBucketCorsInput, ...request.Option) (*s3.PutBucketCorsOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketEncryption(*s3.PutBucketEncryptionInput) (*s3.PutBucketEncryptionOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketEncryptionWithContext(aws.Context, *s3.PutBucketEncryptionInput, ...request.Option) (*s3.PutBucketEncryptionOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketEncryptionRequest(*s3.PutBucketEncryptionInput) (*request.Request, *s3.PutBucketEncryptionOutput) {
	return nil, nil
}

func (ms *MockS3) PutBucketIntelligentTieringConfiguration(*s3.PutBucketIntelligentTieringConfigurationInput) (*s3.PutBucketIntelligentTieringConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketIntelligentTieringConfigurationWithContext(aws.Context, *s3.PutBucketIntelligentTieringConfigurationInput, ...request.Option) (*s3.PutBucketIntelligentTieringConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketIntelligentTieringConfigurationRequest(*s3.PutBucketIntelligentTieringConfigurationInput) (*request.Request, *s3.PutBucketIntelligentTieringConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) PutBucketInventoryConfiguration(*s3.PutBucketInventoryConfigurationInput) (*s3.PutBucketInventoryConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketInventoryConfigurationWithContext(aws.Context, *s3.PutBucketInventoryConfigurationInput, ...request.Option) (*s3.PutBucketInventoryConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketInventoryConfigurationRequest(*s3.PutBucketInventoryConfigurationInput) (*request.Request, *s3.PutBucketInventoryConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) PutBucketLifecycleWithContext(aws.Context, *s3.PutBucketLifecycleInput, ...request.Option) (*s3.PutBucketLifecycleOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketLifecycleConfigurationWithContext(aws.Context, *s3.PutBucketLifecycleConfigurationInput, ...request.Option) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketLoggingWithContext(aws.Context, *s3.PutBucketLoggingInput, ...request.Option) (*s3.PutBucketLoggingOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketMetricsConfiguration(*s3.PutBucketMetricsConfigurationInput) (*s3.PutBucketMetricsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketMetricsConfigurationWithContext(aws.Context, *s3.PutBucketMetricsConfigurationInput, ...request.Option) (*s3.PutBucketMetricsConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketMetricsConfigurationRequest(*s3.PutBucketMetricsConfigurationInput) (*request.Request, *s3.PutBucketMetricsConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) PutBucketNotificationWithContext(aws.Context, *s3.PutBucketNotificationInput, ...request.Option) (*s3.PutBucketNotificationOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketNotificationConfigurationWithContext(aws.Context, *s3.PutBucketNotificationConfigurationInput, ...request.Option) (*s3.PutBucketNotificationConfigurationOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketOwnershipControls(*s3.PutBucketOwnershipControlsInput) (*s3.PutBucketOwnershipControlsOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketOwnershipControlsWithContext(aws.Context, *s3.PutBucketOwnershipControlsInput, ...request.Option) (*s3.PutBucketOwnershipControlsOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutBucketOwnershipControlsRequest(*s3.PutBucketOwnershipControlsInput) (*request.Request, *s3.PutBucketOwnershipControlsOutput) {
	return nil, nil
}

func (ms *MockS3) PutBucketPolicyWithContext(aws.Context, *s3.PutBucketPolicyInput, ...request.Option) (*s3.PutBucketPolicyOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketReplicationWithContext(aws.Context, *s3.PutBucketReplicationInput, ...request.Option) (*s3.PutBucketReplicationOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketRequestPaymentWithContext(aws.Context, *s3.PutBucketRequestPaymentInput, ...request.Option) (*s3.PutBucketRequestPaymentOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketTaggingWithContext(aws.Context, *s3.PutBucketTaggingInput, ...request.Option) (*s3.PutBucketTaggingOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketVersioningWithContext(aws.Context, *s3.PutBucketVersioningInput, ...request.Option) (*s3.PutBucketVersioningOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutBucketWebsiteWithContext(aws.Context, *s3.PutBucketWebsiteInput, ...request.Option) (*s3.PutBucketWebsiteOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutObjectWithContext(aws.Context, *s3.PutObjectInput, ...request.Option) (*s3.PutObjectOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutObjectAclWithContext(aws.Context, *s3.PutObjectAclInput, ...request.Option) (*s3.PutObjectAclOutput, error) {
	return nil, nil
}

func (ms *MockS3) PutObjectLegalHold(*s3.PutObjectLegalHoldInput) (*s3.PutObjectLegalHoldOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectLegalHoldWithContext(aws.Context, *s3.PutObjectLegalHoldInput, ...request.Option) (*s3.PutObjectLegalHoldOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectLegalHoldRequest(*s3.PutObjectLegalHoldInput) (*request.Request, *s3.PutObjectLegalHoldOutput) {
	return nil, nil
}

func (ms *MockS3) PutObjectLockConfiguration(*s3.PutObjectLockConfigurationInput) (*s3.PutObjectLockConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectLockConfigurationWithContext(aws.Context, *s3.PutObjectLockConfigurationInput, ...request.Option) (*s3.PutObjectLockConfigurationOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectLockConfigurationRequest(*s3.PutObjectLockConfigurationInput) (*request.Request, *s3.PutObjectLockConfigurationOutput) {
	return nil, nil
}

func (ms *MockS3) PutObjectRetention(*s3.PutObjectRetentionInput) (*s3.PutObjectRetentionOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectRetentionWithContext(aws.Context, *s3.PutObjectRetentionInput, ...request.Option) (*s3.PutObjectRetentionOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectRetentionRequest(*s3.PutObjectRetentionInput) (*request.Request, *s3.PutObjectRetentionOutput) {
	return nil, nil
}

func (ms *MockS3) PutObjectTagging(*s3.PutObjectTaggingInput) (*s3.PutObjectTaggingOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectTaggingWithContext(aws.Context, *s3.PutObjectTaggingInput, ...request.Option) (*s3.PutObjectTaggingOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutObjectTaggingRequest(*s3.PutObjectTaggingInput) (*request.Request, *s3.PutObjectTaggingOutput) {
	return nil, nil
}

func (ms *MockS3) PutPublicAccessBlock(*s3.PutPublicAccessBlockInput) (*s3.PutPublicAccessBlockOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutPublicAccessBlockWithContext(aws.Context, *s3.PutPublicAccessBlockInput, ...request.Option) (*s3.PutPublicAccessBlockOutput, error) {
	return nil, nil
}
func (ms *MockS3) PutPublicAccessBlockRequest(*s3.PutPublicAccessBlockInput) (*request.Request, *s3.PutPublicAccessBlockOutput) {
	return nil, nil
}

func (ms *MockS3) RestoreObjectWithContext(aws.Context, *s3.RestoreObjectInput, ...request.Option) (*s3.RestoreObjectOutput, error) {
	return nil, nil
}

func (ms *MockS3) SelectObjectContent(*s3.SelectObjectContentInput) (*s3.SelectObjectContentOutput, error) {
	return nil, nil
}
func (ms *MockS3) SelectObjectContentWithContext(aws.Context, *s3.SelectObjectContentInput, ...request.Option) (*s3.SelectObjectContentOutput, error) {
	return nil, nil
}
func (ms *MockS3) SelectObjectContentRequest(*s3.SelectObjectContentInput) (*request.Request, *s3.SelectObjectContentOutput) {
	return nil, nil
}

func (ms *MockS3) UploadPartWithContext(aws.Context, *s3.UploadPartInput, ...request.Option) (*s3.UploadPartOutput, error) {
	return nil, nil
}

func (ms *MockS3) UploadPartCopyWithContext(aws.Context, *s3.UploadPartCopyInput, ...request.Option) (*s3.UploadPartCopyOutput, error) {
	return nil, nil
}

func (ms *MockS3) WriteGetObjectResponse(*s3.WriteGetObjectResponseInput) (*s3.WriteGetObjectResponseOutput, error) {
	return nil, nil
}
func (ms *MockS3) WriteGetObjectResponseWithContext(aws.Context, *s3.WriteGetObjectResponseInput, ...request.Option) (*s3.WriteGetObjectResponseOutput, error) {
	return nil, nil
}
func (ms *MockS3) WriteGetObjectResponseRequest(*s3.WriteGetObjectResponseInput) (*request.Request, *s3.WriteGetObjectResponseOutput) {
	return nil, nil
}

func (ms *MockS3) WaitUntilBucketExists(*s3.HeadBucketInput) error {
	return nil
}
func (ms *MockS3) WaitUntilBucketExistsWithContext(aws.Context, *s3.HeadBucketInput, ...request.WaiterOption) error {
	return nil
}

func (ms *MockS3) WaitUntilBucketNotExists(*s3.HeadBucketInput) error {
	return nil
}
func (ms *MockS3) WaitUntilBucketNotExistsWithContext(aws.Context, *s3.HeadBucketInput, ...request.WaiterOption) error {
	return nil
}

func (ms *MockS3) WaitUntilObjectExists(*s3.HeadObjectInput) error {
	return nil
}
func (ms *MockS3) WaitUntilObjectExistsWithContext(aws.Context, *s3.HeadObjectInput, ...request.WaiterOption) error {
	return nil
}

func (ms *MockS3) WaitUntilObjectNotExists(*s3.HeadObjectInput) error {
	return nil
}
func (ms *MockS3) WaitUntilObjectNotExistsWithContext(aws.Context, *s3.HeadObjectInput, ...request.WaiterOption) error {
	return nil
}

var _ s3iface.S3API = (*MockS3)(nil)
