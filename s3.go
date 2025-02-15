package s3

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/barnybug/s3/pkg/mys3"
)

const (
	PART_SIZE = 6_000_000 // Has to be 5_000_000 minimim
	RETRIES   = 2
)

type S3Filesystem struct {
	err    error
	conn   s3iface.S3API
	bucket string
	path   string
	mys3   mys3.Mys3
}

type S3File struct {
	conn   s3iface.S3API
	bucket string
	object *s3.Object
	path   string
	md5    []byte
	mys3   mys3.Mys3
}

func strMd5(str string) (retMd5 string) {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func (s3f *S3File) CheckSum() (string, error) {
	data, err := ioutil.ReadFile(s3f.path)
	if err != nil {
		return "", err
	}
	return strMd5(string(data)), nil
}

func (s3f *S3File) Relative() string {
	return s3f.path
}

func (s3f *S3File) Size() int64 {
	return *s3f.object.Size
}

func (s3f *S3File) IsDirectory() bool {
	return strings.HasSuffix(s3f.path, "/") && *s3f.object.Size == 0
}

func (s3f *S3File) MD5() []byte {
	if s3f.md5 == nil {
		etag := *s3f.object.ETag
		v := etag[1 : len(etag)-1]
		s3f.md5, _ = hex.DecodeString(v)
	}
	return s3f.md5
}

func (s3f *S3File) Reader() (io.ReadCloser, error) {
	input := s3.GetObjectInput{
		Bucket: aws.String(s3f.bucket),
		Key:    s3f.object.Key,
	}
	output, err := s3f.mys3.GetObject(&input)

	if err != nil {
		return nil, err
	}
	if onlyShow {

		out, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			return nil, err
		}
		fmt.Println(string(out))
	}
	return output.Body, err
}

func (s3f *S3File) Delete() error {
	input := s3.DeleteObjectInput{
		Bucket: aws.String(s3f.bucket),
		Key:    s3f.object.Key,
	}
	_, err := s3f.conn.DeleteObject(&input)
	return err
}

func (s3f *S3File) String() string {
	return fmt.Sprintf("s3://%s/%s", s3f.bucket, *s3f.object.Key)
}

func (s3fs *S3Filesystem) Error() error {
	return s3fs.err
}

func (s3fs *S3Filesystem) Files() <-chan File {
	ch := make(chan File, 1000)
	stripLen := strings.LastIndex(s3fs.path, "/") + 1
	if stripLen == -1 {
		stripLen = 0
	}
	go func() {
		defer close(ch)
		truncated := true
		marker := ""
		for truncated {
			input := s3.ListObjectsInput{
				Bucket: aws.String(s3fs.bucket),
				Prefix: aws.String(s3fs.path),
				Marker: aws.String(marker),
			}
			output, err := s3fs.mys3.ListObject(&input)
			if err != nil {
				s3fs.err = err
				return
			}
			for _, c := range output.Contents {
				key := c
				relpath := (*key.Key)[stripLen:]
				ch <- &S3File{s3fs.conn, s3fs.bucket, key, relpath, nil, s3fs.mys3}
				marker = *c.Key
			}
			truncated = *output.IsTruncated
		}
	}()
	return ch
}

func guessMimeType(filename string) string {
	ext := mime.TypeByExtension(filepath.Ext(filename))
	if ext == "" {
		ext = "application/binary"
	}
	return ext
}

func (s3fs *S3Filesystem) Create(src File) error {
	var fullpath string
	if s3fs.path == "" || strings.HasSuffix(s3fs.path, "/") {
		fullpath = filepath.Join(s3fs.path, src.Relative())
	} else {
		fullpath = s3fs.path
	}
	checkSum, err := src.CheckSum()
	if err != nil {
		return err
	}
	input := s3manager.UploadInput{
		ACL:      aws.String(acl),
		Bucket:   aws.String(s3fs.bucket),
		Key:      aws.String(fullpath),
		Metadata: map[string]*string{"md5_checksum": &checkSum},
	}
	switch t := src.(type) {
	case *S3File:
		// special case for S3File to preserve header information
		getObjectInput := s3.GetObjectInput{
			Bucket: aws.String(t.bucket),
			Key:    t.object.Key,
		}
		output, err := s3fs.mys3.GetObject(&getObjectInput)
		//output, err := s3fs.conn.GetObject(&getObjectInput)
		if err != nil {
			return err
		}
		defer output.Body.Close()
		input.Body = output.Body
		// transfer existing headers across
		input.ContentType = output.ContentType
		// input.LastModified = output.LastModified
		input.StorageClass = output.StorageClass
	default:
		reader, err := src.Reader()
		if err != nil {
			return err
		}
		input.Body = reader
		defer reader.Close()
		input.ContentType = aws.String(guessMimeType(src.Relative()))
	}
	_, err = s3fs.mys3.Upload(&input)
	return err
}

func (s3fs *S3Filesystem) CreateMultiPart(src File, buffer []byte) error {
	var fullpath string
	if s3fs.path == "" || strings.HasSuffix(s3fs.path, "/") {
		fullpath = filepath.Join(s3fs.path, src.Relative())
	} else {
		fullpath = s3fs.path
	}
	input := s3manager.UploadInput{
		ACL:    aws.String(acl),
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(fullpath),
	}
	checkSum, err := src.CheckSum()
	if err != nil {
		return err
	}
	switch t := src.(type) {
	case *S3File:
		// special case for S3File to preserve header information
		getObjectInput := s3.GetObjectInput{
			Bucket: aws.String(t.bucket),
			Key:    t.object.Key,
		}
		output, err := s3fs.mys3.GetObject(&getObjectInput)
		//output, err := s3fs.conn.GetObject(&getObjectInput)
		if err != nil {
			return err
		}
		defer output.Body.Close()
		input.Body = output.Body
		// transfer existing headers across
		input.ContentType = output.ContentType
		// input.LastModified = output.LastModified
		input.StorageClass = output.StorageClass
	default:
		reader, err := src.Reader()
		if err != nil {
			return err
		}
		input.Body = reader
		defer reader.Close()
		input.ContentType = aws.String(guessMimeType(src.Relative()))
	}

	expiryDate := time.Now().AddDate(0, 0, 1)
	createdResp, err := s3fs.mys3.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket:   aws.String(s3fs.bucket),
		Key:      aws.String(fullpath),
		Metadata: map[string]*string{"md5_checksum": &checkSum},
		Expires:  &expiryDate,
	})
	if err != nil {
		return err
	}
	var start, currentSize int
	var remaining = int(src.Size())
	var partNum = 1
	var completedParts []*s3.CompletedPart
	// Loop till remaining upload size is 0
	for start = 0; remaining != 0; start += PART_SIZE {
		if remaining < PART_SIZE {
			currentSize = remaining
		} else {
			currentSize = PART_SIZE
		}

		completed, err := Upload(s3fs.mys3, createdResp, buffer[start:start+currentSize], partNum)
		// If upload function failed (meaning it retried acoording to RETRIES)
		if err != nil {
			_, err = s3fs.mys3.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
				Bucket:   createdResp.Bucket,
				Key:      createdResp.Key,
				UploadId: createdResp.UploadId,
			})
			if err != nil {
				// god speed

				return err
			}
		}

		// Detract the current part size from remaining
		remaining -= currentSize
		fmt.Printf("Part %v complete, %v btyes remaining\n", partNum, remaining)

		// Add the completed part to our list
		completedParts = append(completedParts, completed)
		partNum++

	}
	_, err = s3fs.mys3.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   createdResp.Bucket,
		Key:      createdResp.Key,
		UploadId: createdResp.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s3fs *S3Filesystem) Delete(path string) error {
	fullpath := filepath.Join(s3fs.path, path)
	input := s3.DeleteObjectInput{
		Bucket: aws.String(s3fs.bucket),
		Key:    aws.String(fullpath),
	}
	_, err := s3fs.conn.DeleteObject(&input)
	return err
}

func Upload(mys3 mys3.Mys3, resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNum int) (completedPart *s3.CompletedPart, err error) {
	var try int
	for try <= RETRIES {
		uploadResp, err := mys3.UploadPart(&s3.UploadPartInput{
			Body:          bytes.NewReader(fileBytes),
			Bucket:        resp.Bucket,
			Key:           resp.Key,
			PartNumber:    aws.Int64(int64(partNum)),
			UploadId:      resp.UploadId,
			ContentLength: aws.Int64(int64(len(fileBytes))),
		})
		// Upload failed
		if err != nil {
			// Max retries reached! Quitting
			if try == RETRIES {
				return nil, err
			} else {
				// Retrying
				try++
			}
		} else {
			// Upload is done!
			return &s3.CompletedPart{
				ETag:       uploadResp.ETag,
				PartNumber: aws.Int64(int64(partNum)),
			}, nil
		}
	}

	return nil, nil
}
