package storages

import (
	"go-serviceboilerplate/infrastructures/configurations"
	cloudflare_r2 "go-serviceboilerplate/infrastructures/storages/cloudflare-r2"
)

type StorageInstance struct {
	MainR2Client *cloudflare_r2.CloudflareR2Instance
}

func NewStorageInstance(configs *configurations.Configs) *StorageInstance {
	r2Client := cloudflare_r2.NewCloudflareR2Instance(configs)

	return &StorageInstance{
		MainR2Client: r2Client,
	}
}