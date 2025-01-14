package s3

import "io"

type File interface {
	Relative() string
	Size() int64
	MD5() []byte
	Reader() (io.ReadCloser, error)
	Delete() error
	String() string
	IsDirectory() bool
	CheckSum() (string, error)
}

type Filesystem interface {
	Files() <-chan File
	Create(src File) error
	Delete(path string) error
	Error() error
	CreateMultiPart(src File, buffer []byte) error
}
