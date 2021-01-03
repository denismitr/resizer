package s3storage

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"io"
	"resizer/storage"
	"strings"
)

type Config struct {
	AccessKey string
	AccessSecret string
	AccessToken string
	Region string
	Endpoint string
	S3ForcePathStyle bool
	EnableSSL bool
}

type RemoteStorage struct {
	cfg Config
	s3Config *aws.Config
	client *s3.S3
}

func New(cfg Config) *RemoteStorage {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.AccessSecret, cfg.AccessToken),
		Endpoint:         aws.String(cfg.Endpoint),
		Region:           aws.String(cfg.Region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	newSession := session.New(s3Config)
	s3Client := s3.New(newSession)

	return &RemoteStorage{
		cfg: cfg,
		s3Config: s3Config,
		client: s3Client,
	}
}

func (rs *RemoteStorage) Put(ctx context.Context, bucket, filename string, source io.ReadSeeker) (*storage.Item, error) {
	// fixme: use context

	sess, err := rs.getSession()
	if err != nil {
		return nil, err
	}

	s3Client := s3.New(sess)

	// Create a new bucket using the CreateBucket call.
	b := &s3.CreateBucketInput{Bucket: aws.String(bucket)}
	if _, err := s3Client.CreateBucket(b); err != nil {
		if !strings.Contains(err.Error(), s3.ErrCodeBucketAlreadyExists) && !strings.Contains(err.Error(), s3.ErrCodeBucketAlreadyOwnedByYou) {
			return nil, errors.Wrapf(
				storage.ErrStorageFailed,
				"could not create bucket %s: %v",
				bucket, err,
			)
		}
	}

	result, err := s3Client.PutObject(&s3.PutObjectInput{
		Body:   source,
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		return nil, errors.Wrapf(
			storage.ErrStorageFailed,
			"could not upload file %s to bucket %s: %v",
			filename, bucket, err,
		)
	}

	return &storage.Item{
		Path: bucket + "/" + filename,
		Result: result.String(),
	}, nil
}

type FakeWriterAt struct {
	w io.Writer
}

func (fw FakeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	// ignore 'offset' because we forced sequential downloads
	return fw.w.Write(p)
}


func (rs *RemoteStorage) Download(ctx context.Context, dst io.Writer, bucket, file string) error {
	// fixme: use context

	newSession, err := rs.getSession()
	if err != nil {
		return err
	}


	downloader := s3manager.NewDownloader(newSession) // todo: will we have actually large files > 5Mb?
	downloader.Concurrency = 1

	w := FakeWriterAt{w: dst}
	_, err = downloader.Download(w,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(file),
		})

	if err != nil {
		return errors.Wrapf(
			storage.ErrStorageFailed,
			"could not download file %s from bucket %s: %v",
			file, bucket, err,
		)
	}

	return nil
}

func (rs *RemoteStorage) getSession() (*session.Session, error) {
	newSession, err := session.NewSession(rs.s3Config)
	if err != nil {
		return nil, errors.Wrapf(storage.ErrStorageFailed, "s3 session could not be created: %v", err)
	}

	return newSession, nil
}