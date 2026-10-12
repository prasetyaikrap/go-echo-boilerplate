package s3

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

// S3Instance works with any S3-compatible storage (AWS S3, Cloudflare R2, MinIO, etc.)
type S3Instance struct {
	Client        *awsS3.Client
	PresignClient *awsS3.PresignClient
	BucketName    string
	PublicHost    string
	BaseFolder    string
	DefaultPresignExpire time.Duration
}

func NewS3Instance(configs *configurations.Configs) *S3Instance {
	s3Config := configs.Envs.StorageS3

	if s3Config.BucketName == "" {
		configs.Logger.Fatal("S3 storage bucket name is not configured (S3_BUCKET_NAME)")
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithCredentialsProvider(awsCredentials.NewStaticCredentialsProvider(s3Config.AccessKeyID, s3Config.SecretAccessKey, "")),
		awsConfig.WithRegion(s3Config.Region),
	)

	if err != nil {
		configs.Logger.Fatal("Failed to load AWS SDK config for S3 storage", "error", err)
	}

	client := awsS3.NewFromConfig(cfg, func(o *awsS3.Options) {
		// Empty endpoint falls back to AWS S3 default endpoint for the region
		if s3Config.Endpoint != "" {
			o.BaseEndpoint = aws.String(s3Config.Endpoint)
		}
		o.UsePathStyle = s3Config.UsePathStyle
	})

	return &S3Instance{
		Client:        client,
		PresignClient: awsS3.NewPresignClient(client),
		BucketName:    s3Config.BucketName,
		PublicHost:    s3Config.PublicHost,
		BaseFolder:    s3Config.BaseFolder,
		DefaultPresignExpire: s3Config.DefaultPresignExpire,
	}
}

func (c *S3Instance) buildKey(key string) string {
	return strings.TrimPrefix(path.Join(c.BaseFolder, key), "/")
}

func (c *S3Instance) UploadObject(ctx context.Context, key string, byteReader io.Reader, contentType string) (*awsS3.PutObjectOutput, error) {
	assignedKey := c.buildKey(key)

	object, err := c.Client.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket:      aws.String(c.BucketName),
		Key:         aws.String(assignedKey),
		Body:        byteReader,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return nil, fmt.Errorf("s3 upload object %q: %w", assignedKey, err)
	}

	return object, nil
}

func (c *S3Instance) GetObject(ctx context.Context, key string) (*awsS3.GetObjectOutput, error) {
	assignedKey := c.buildKey(key)

	object, err := c.Client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(c.BucketName),
		Key:    aws.String(assignedKey),
	})

	if err != nil {
		return nil, fmt.Errorf("s3 get object %q: %w", assignedKey, err)
	}

	return object, nil
}

func (c *S3Instance) DeleteObject(ctx context.Context, key string) (*awsS3.DeleteObjectOutput, error) {
	assignedKey := c.buildKey(key)

	object, err := c.Client.DeleteObject(ctx, &awsS3.DeleteObjectInput{
		Bucket: aws.String(c.BucketName),
		Key:    aws.String(assignedKey),
	})

	if err != nil {
		return nil, fmt.Errorf("s3 delete object %q: %w", assignedKey, err)
	}

	return object, nil
}

func (c *S3Instance) GetPublicObjectURL(key string) string {
	assignedKey := c.buildKey(key)
	host := strings.TrimSuffix(c.PublicHost, "/")

	return fmt.Sprintf("%s/%s", host, assignedKey)
}

// GetPresignedURL returns a temporary URL to download a private object. Zero expireTime uses DefaultPresignExpire.
func (c *S3Instance) GetPresignedURL(ctx context.Context, key string, expireTime time.Duration) (string, error) {
	assignedKey := c.buildKey(key)

	presignedURL, err := c.PresignClient.PresignGetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(c.BucketName),
		Key:    aws.String(assignedKey),
	}, awsS3.WithPresignExpires(c.resolvePresignExpire(expireTime)))

	if err != nil {
		return "", fmt.Errorf("s3 presign get object %q: %w", assignedKey, err)
	}

	return presignedURL.URL, nil
}

// GetPresignedUploadURL returns a temporary URL the client can PUT a file to directly. Zero expireTime uses DefaultPresignExpire.
// The client must send the same Content-Type header as the one signed here.
func (c *S3Instance) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expireTime time.Duration) (string, error) {
	assignedKey := c.buildKey(key)

	presignedURL, err := c.PresignClient.PresignPutObject(ctx, &awsS3.PutObjectInput{
		Bucket:      aws.String(c.BucketName),
		Key:         aws.String(assignedKey),
		ContentType: aws.String(contentType),
	}, awsS3.WithPresignExpires(c.resolvePresignExpire(expireTime)))

	if err != nil {
		return "", fmt.Errorf("s3 presign put object %q: %w", assignedKey, err)
	}

	return presignedURL.URL, nil
}

// resolvePresignExpire returns 0 when nothing is configured, which the SDK treats as its default (15 minutes)
func (c *S3Instance) resolvePresignExpire(expireTime time.Duration) time.Duration {
	if expireTime <= 0 {
		return c.DefaultPresignExpire
	}

	return expireTime
}
