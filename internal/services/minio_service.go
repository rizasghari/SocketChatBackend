package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"socketChat/configs"
	"socketChat/internal/enums"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioService struct {
	minioClient *minio.Client
	config      *configs.Config
}

var (
	minioService *MinioService
	once         sync.Once
)

func NewMinioService(config *configs.Config) *MinioService {
	once.Do(func() {
		endpoint := config.Viper.GetString("minio.endpoint")
		accessKeyID := config.Viper.GetString("minio.access_key_id")
		secretAccessKey := config.Viper.GetString("minio.secret_access_key")
		useSSL := config.Viper.GetBool("minio.use_ssl")

		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: useSSL,
		})

		if err != nil {
			log.Fatalln(err)
		}

		bucketName := enums.FILE_BUCKET_USER_PROFILE
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
			if errBucketExists == nil && exists {
				log.Printf("We already own %s\n", bucketName)
			} else {
				log.Fatalln(err)
			}
		} else {
			log.Printf("Successfully created %s\n", bucketName)
		}

		minioService = &MinioService{
			minioClient: minioClient,
			config:      config,
		}
	})

	if minioService == nil {
		log.Fatalln("MinioService is not initialized")
	}
	return minioService
}

func (ms *MinioService) UploadFile(fileName string, file io.Reader, fileSize int64, contentType string, bucketName string) (string, error) {
	info, err := ms.minioClient.PutObject(context.Background(), bucketName, fileName, file, fileSize, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		log.Println(err)
		return "", err
	}

	publicUrl, err := ms.GetPublicFileUrl(bucketName, info.Key)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return publicUrl, nil
}

func (ms *MinioService) GetPublicFileUrl(bucketName, fileKey string) (string, error) {
	externalEndpoint := ms.config.Viper.GetString("minio.external_endpoint")
	path := fmt.Sprintf("http://%s/%s/%s", externalEndpoint, bucketName, fileKey)
	return path, nil
}
