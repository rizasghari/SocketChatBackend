package interfaces

import "io"

type FileManager interface {
	UploadFile(fileName string, file io.Reader, fileSize int64, contentType string, bucketName string) (string, error)
}