package cloudflare_r2

import (
	"context"
	"fmt"
	"go-serviceboilerplate/infrastructures/configurations"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsCredentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type CloudflareR2Instance struct {
	Client *awsS3.Client
	BucketName *string
	PublicHost string
	BaseFolder string
}

func NewCloudflareR2Instance(configs *configurations.Configs) *CloudflareR2Instance {
	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithCredentialsProvider(awsCredentials.NewStaticCredentialsProvider(configs.Envs.StorageR2.AccessKeyID, configs.Envs.StorageR2.SecretAccessKey, "")),
		awsConfig.WithRegion("auto"), // Required by SDK but not used by R2
	)

	if err != nil {
		configs.Logger.Fatal("Failed to load AWS SDK config for Cloudflare R2", err)
	}

	client := awsS3.NewFromConfig(cfg, func(o *awsS3.Options) {
      o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", configs.Envs.StorageR2.AccountID))
  	})

	return &CloudflareR2Instance{
		Client: client, 
		BucketName: aws.String(configs.Envs.StorageR2.BucketName), 
		PublicHost: configs.Envs.StorageR2.PublicHost,
		BaseFolder: configs.Envs.StorageR2.BaseFolder,
	}
}

func (c *CloudflareR2Instance) buildKey(key string) string {
	normalizedBaseFolder := strings.TrimPrefix(c.BaseFolder, "/")
	return path.Join(normalizedBaseFolder, key)
}

func (c *CloudflareR2Instance) UploadObject(ctx context.Context, key string, byteReader io.Reader, ContentType string) (*awsS3.PutObjectOutput, error) {
	assignedKey := c.buildKey(key)

	object, err := c.Client.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: c.BucketName,
		Key:    aws.String(assignedKey),
		Body:   byteReader,
		ContentType: aws.String(ContentType),
	})
	return object, err
}

func (c *CloudflareR2Instance) GetObject(ctx context.Context, key string) (*awsS3.GetObjectOutput, error) {
	assignedKey := c.buildKey(key)
	
	object, err := c.Client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: c.BucketName,
		Key:    aws.String(assignedKey),
	})

	if err != nil {
		return nil, err
	}

	return object, nil
}

func (c *CloudflareR2Instance) DeleteObject(ctx context.Context, key string) (*awsS3.DeleteObjectOutput, error) {
	assignedKey := c.buildKey(key)

	object, err := c.Client.DeleteObject(ctx, &awsS3.DeleteObjectInput{
		Bucket: c.BucketName,
		Key:    aws.String(assignedKey),
	})

	if err != nil {
		return nil, err
	}

	return object, nil
}

func (c *CloudflareR2Instance) GetPublicObjectURL(key string) string {
	assignedKey := c.buildKey(key)
	host := strings.TrimSuffix(c.PublicHost, "/")
	
	return fmt.Sprintf("%s/%s", host, assignedKey)
}

func (c *CloudflareR2Instance) GetPresignedURL(ctx context.Context, key string, expireTime *time.Duration) (string, error) {
	assignedKey := c.buildKey(key)

	if expireTime == nil {
		defaultExpire := 15 * time.Minute
		expireTime = &defaultExpire
	}

	presignClient := awsS3.NewPresignClient(c.Client)
	presignedURL, err := presignClient.PresignGetObject(ctx, &awsS3.GetObjectInput{
		Bucket: c.BucketName,
		Key:    aws.String(assignedKey),
	}, awsS3.WithPresignExpires(*expireTime))

	if err != nil {
		return "", err
	}

	return presignedURL.URL, nil
}