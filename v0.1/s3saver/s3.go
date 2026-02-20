package s3saver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hput"
	"io/ioutil"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
)

const metadataInput = "input"

// Logger logs out.
type Logger interface {
	Debugf(msg string, args ...interface{})
	Errorf(msg string, args ...interface{})
}

type S3Saver struct {
	Logger Logger
	Client client
	Prefix string
	Bucket string
}

// client models the s3 client
type client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type option interface {
	apply(s *S3Saver) error
}

// PrefixOption sets the prefix for the S3 bucket
type PrefixOption struct {
	Prefix string
}

func (p PrefixOption) apply(s *S3Saver) error {
	s.Prefix = p.Prefix
	return nil
}

// S3ClientOption allows you to set the s3 client
type S3ClientOption struct {
	client client
}

func (o S3ClientOption) apply(s *S3Saver) error {
	s.Client = o.client
	return nil
}

func New(ctx context.Context, l Logger, b string, options ...option) (S3Saver, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		l.Errorf("failed to load config: %v", err)
		return S3Saver{}, err
	}
	// Create Session
	client := s3.NewFromConfig(cfg)
	if b == "" {
		return S3Saver{}, errors.New("bucket must be provided")
	}
	sa := S3Saver{
		Logger: l,
		Client: client,
		Bucket: b,
	}
	for _, o := range options {
		err = o.apply(&sa)
		if err != nil {
			return S3Saver{}, err
		}
	}
	return sa, nil
}

// SaveText saves text to the configured bucket and prefix at the provided path
func (sa S3Saver) SaveText(ctx context.Context, s string, p url.URL, r *hput.PutResult) error {
	key := sa.getKey(p.Path)
	exists, err := sa.checkExists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if text exists: %w", err)
	}

	i := s3.PutObjectInput{
		Bucket: &sa.Bucket,
		Key:    &key,
		Metadata: map[string]string{
			metadataInput: string(hput.Text),
		},
		Body: bytes.NewBufferString(s),
	}
	_, err = sa.Client.PutObject(ctx, &i)
	if err != nil {
		sa.Logger.Errorf("failed to put string: %v", err)
		return fmt.Errorf("failed to put string: %w", err)
	}
	r.Overwrote = exists
	return nil
}

// SaveCode saves code as text to the configured bucket and prefix at the provided path
func (sa S3Saver) SaveCode(ctx context.Context, c string, p url.URL, r *hput.PutResult) error {
	key := sa.getKey(p.Path)
	exists, err := sa.checkExists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if code exists: %w", err)
	}

	i := s3.PutObjectInput{
		Bucket: &sa.Bucket,
		Key:    &key,
		Metadata: map[string]string{
			metadataInput: string(hput.Js),
		},
		Body: bytes.NewBufferString(c),
	}
	_, err = sa.Client.PutObject(ctx, &i)
	if err != nil {
		sa.Logger.Errorf("failed to put code: %v", err)
		return fmt.Errorf("failed to put code: %w", err)
	}
	r.Overwrote = exists
	return nil
}

// SaveBinary saves code as text to the configured bucket and prefix at the provided path
func (sa S3Saver) SaveBinary(ctx context.Context, b []byte, p url.URL, r *hput.PutResult) error {
	key := sa.getKey(p.Path)
	exists, err := sa.checkExists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if binary exists: %w", err)
	}

	i := s3.PutObjectInput{
		Bucket: &sa.Bucket,
		Key:    &key,
		Metadata: map[string]string{
			metadataInput: string(hput.Binary),
		},
		Body: bytes.NewBuffer(b),
	}
	_, err = sa.Client.PutObject(ctx, &i)
	if err != nil {
		sa.Logger.Errorf("failed to put binary: %v", err)
		return fmt.Errorf("failed to put binary: %w", err)
	}
	r.Overwrote = exists
	return nil
}

func (sa S3Saver) checkExists(ctx context.Context, key string) (bool, error) {
	exists := false
	checkI := s3.GetObjectInput{
		Bucket: &sa.Bucket,
		Key:    &key,
	}
	_, err := sa.Client.GetObject(ctx, &checkI)
	if err != nil {
		var notFoundErr *types.NoSuchKey
		var noAccessErr *ssotypes.UnauthorizedException
		if !errors.As(err, &notFoundErr) && !errors.As(err, &noAccessErr) {
			// If you don't have listBucket and the object isn't there, you get UnauthorizedException
			sa.Logger.Errorf("failed to check if object exists: %v", err)
			return exists, fmt.Errorf("failed to check if object exists: %w", err)
		}
		exists = true
	}
	return exists, nil
}

// getRunnableFromKey returns the runnable associated with the exact key
func (sa S3Saver) getRunnableFromKey(ctx context.Context, key string) (hput.Runnable, error) {
	i := s3.GetObjectInput{
		Bucket: &sa.Bucket,
		Key:    &key,
	}
	o, err := sa.Client.GetObject(ctx, &i)
	if err != nil {
		var notFoundErr *types.NoSuchKey
		var noAccessErr *ssotypes.UnauthorizedException
		// If you don't have listBucket and the object isn't there, you get ResourceNotFoundException
		if !errors.As(err, &notFoundErr) && !errors.As(err, &noAccessErr) {
			sa.Logger.Errorf("failed access runnable: %v", err)
			return hput.Runnable{}, fmt.Errorf("failed access runnable: %w", err)
		}
		sa.Logger.Debugf("runnable not found: %v", err)
		// return empty runnable because none was found
		return hput.Runnable{}, nil
	}
	sa.Logger.Debugf("s3 object found: %#v with metadata: %+v", o, o.Metadata)
	r := hput.Runnable{
		Path: key[len(sa.Prefix):],
		Type: hput.Input(o.Metadata[metadataInput]),
	}
	bts, err := ioutil.ReadAll(o.Body)
	if err != nil {
		sa.Logger.Errorf("failed to read runnable: %v", err)
		return hput.Runnable{}, fmt.Errorf("failed to read runnable: %w", err)
	}
	sa.Logger.Debugf("runnable type found: %s", o.Metadata[metadataInput])
	switch o.Metadata[metadataInput] {
	case string(hput.Text), string(hput.Js):
		r.Text = string(bts)
	case string(hput.Binary):
		r.Binary = bts
	default:
		sa.Logger.Errorf("unknown runnable type: %v", o.Metadata[metadataInput])
		return hput.Runnable{}, fmt.Errorf("unknown runnable type: %v", o.Metadata[metadataInput])
	}
	return r, nil
}

// GetRunnable returns a runnable from an S3 location associated with the path
func (sa S3Saver) GetRunnable(ctx context.Context, p url.URL) (hput.Runnable, error) {
	key := sa.getKey(p.Path)
	return sa.getRunnableFromKey(ctx, key)
}

func (sa S3Saver) getKey(path string) string {
	return sa.Prefix + path
}

// SendRunnables stream out a list of runnables associated with a path
func (sa S3Saver) SendRunnables(ctx context.Context, p string, runnables chan<- hput.Runnable, done chan<- bool) error {
	prefix := sa.getKey(p)
	in := s3.ListObjectsV2Input{
		Bucket: &sa.Bucket,
		Prefix: &prefix,
	}
	for {
		res, err := sa.Client.ListObjectsV2(ctx, &in)
		if err != nil {
			sa.Logger.Errorf("failed to list objects: %v", err)
			done <- true
			return fmt.Errorf("failed to list objects: %w", err)
		}
		for _, obj := range res.Contents {
			key := *obj.Key
			r, err := sa.getRunnableFromKey(ctx, key)
			if err != nil {
				sa.Logger.Errorf("failed to get runnable for list: %v", err)
				done <- true
				return fmt.Errorf("failed to get runnable for list: %w", err)
			}
			runnables <- r
		}
		if res.NextContinuationToken == nil {
			done <- true
			return nil
		}
		in.ContinuationToken = res.NextContinuationToken
	}

}
