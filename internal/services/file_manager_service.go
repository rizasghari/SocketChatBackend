package services

import (
	"io"
	"socketChat/internal/interfaces"
)

type FileManagerService struct {
	fileManager interfaces.FileManager
}

func NewFileManagerService(fileManager interfaces.FileManager) *FileManagerService {
	return &FileManagerService{
		fileManager: fileManager,
	}
}

func (fs *FileManagerService) UploadUserProfilePhoto(fileName string, file io.Reader, fileSize int64, contentType string, bucketName string) (string, error) {
	return fs.fileManager.UploadFile(fileName, file, fileSize, contentType, bucketName)
}
