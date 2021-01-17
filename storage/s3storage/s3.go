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
	cfg      Config
	s3Config *aws.Config
	client   *s3.S3
}

func New(cfg Config) *RemoteStorage {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.AccessKey, cfg.AccessSecret, cfg.AccessToken),
		Endpoint:         aws.String(cfg.Endpoint),
		Region:           aws.String(cfg.Region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	newSession := session.New(s3Config) // fixme
	s3Client := s3.New(newSession)

	return &RemoteStorage{
		cfg: cfg,
		s3Config: s3Config,
		client: s3Client,
	}
}

func (rs *RemoteStorage) Put(ctx context.Context, namespace, filename string, source io.Reader) (*storage.Item, error) {
	sess, err := rs.getSession()
	if err != nil {
		return nil, err
	}

	s3Client := s3.New(sess)

	// Create a new namespace using the CreateBucket call.
	b := &s3.CreateBucketInput{Bucket: aws.String(namespace)}
	if _, err := s3Client.CreateBucket(b); err != nil {
		if !strings.Contains(err.Error(), s3.ErrCodeBucketAlreadyExists) && !strings.Contains(err.Error(), s3.ErrCodeBucketAlreadyOwnedByYou) {
			return nil, errors.Wrapf(
				storage.ErrStorageFailed,
				"could not create namespace %s: %v",
				namespace, err,
			)
		}
	}

	uploader := s3manager.NewUploader(sess) // todo: will we have actually large files > 5Mb?
	uploader.Concurrency = 1

	result, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Body:   source,
		Bucket: aws.String(namespace),
		Key:    aws.String(filename),
	})

	if err != nil {
		return nil, errors.Wrapf(
			storage.ErrStorageFailed,
			"could not upload file %s to namespace %s: %v",
			filename, namespace, err,
		)
	}

	return &storage.Item{
		Path: namespace + "/" + filename,
		URL:  result.Location, // fixme
	}, nil
}

func (rs *RemoteStorage) Download(ctx context.Context, dst io.Writer, namespace, file string) error {
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
			Bucket: aws.String(namespace),
			Key:    aws.String(file),
		})

	if err != nil {
		return errors.Wrapf(
			storage.ErrStorageFailed,
			"could not download file %s from namespace %s: %v",
			file, namespace, err,
		)
	}

	return nil
}

// Remove file from bucket
func (rs *RemoteStorage) Remove(ctx context.Context, namespace, filename string) error {
	newSession, err := session.NewSession(rs.s3Config)
	if err != nil {
		return errors.Wrapf(
			storage.ErrStorageFailed,
			"could not create S3 session to remove file %s in namespace %s",
			filename, namespace)
	}

	s3Client := s3.New(newSession)
	_, err = s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(namespace),
		Key: aws.String(filename),
	})

	if err != nil {
		return errors.Wrapf(storage.ErrStorageFailed, "could not remove file %s from bucket %s", filename, namespace)
	}

	err = rs.client.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(namespace),
		Key:    aws.String(filename),
	})

	if err != nil {
		return errors.Wrapf(storage.ErrStorageFailed, "could not confirm removal of file %s from bucket %s", filename, namespace)
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