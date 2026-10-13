package storages

import (
	"go-serviceboilerplate/infrastructures/configurations"
	"go-serviceboilerplate/infrastructures/storages/localstorage"
	"go-serviceboilerplate/infrastructures/storages/s3"
)

// StorageInstance holds every storage the service uses. Add a field per extra bucket or directory
type StorageInstance struct {
	MainS3Client     *s3.S3Instance
	MainLocalStorage *localstorage.LocalStorageInstance
}

func NewStorageInstance(configs *configurations.Configs) *StorageInstance {
	return &StorageInstance{
		MainS3Client:     s3.NewS3Instance(configs, configs.Envs.MainS3),
		MainLocalStorage: localstorage.NewLocalStorageInstance(configs, configs.Envs.MainLocalStorage),
	}
}
