package storages

import (
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/storages/s3"
)

type StorageInstance struct {
	MainS3Client *s3.S3Instance
}

func NewStorageInstance(configs *configurations.Configs) *StorageInstance {
	s3Client := s3.NewS3Instance(configs)

	return &StorageInstance{
		MainS3Client: s3Client,
	}
}
