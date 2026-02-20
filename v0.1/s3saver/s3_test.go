package s3saver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hput"
	"io"
	"io/ioutil"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
)

type testLogger struct{}

func (t *testLogger) Debugf(msg string, args ...interface{}) {}

func (t *testLogger) Errorf(msg string, args ...interface{}) {}

type testS3Client struct {
	PutObjectInput      []*s3.PutObjectInput
	PutObjectInputError error
	GetObjectInput      []*s3.GetObjectInput
	GetObjectOutput     *s3.GetObjectOutput
	GetObjectError      error
	ListObjectsV2Output map[string]*s3.ListObjectsV2Output
	outputBodyBytes     *[]byte
}

func (c *testS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	c.PutObjectInput = append(c.PutObjectInput, params)
	return nil, c.PutObjectInputError
}

func (c *testS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	c.GetObjectInput = append(c.GetObjectInput, params)
	if c.GetObjectOutput != nil && c.GetObjectOutput.Body != nil {
		// preserve outgoing body bytes so they can be resent
		if c.outputBodyBytes == nil {
			outputBodyBytes, _ := io.ReadAll(c.GetObjectOutput.Body)
			c.outputBodyBytes = &outputBodyBytes
		}
		if c.outputBodyBytes != nil {
			c.GetObjectOutput.Body = ioutil.NopCloser(bytes.NewBuffer(*c.outputBodyBytes))
		}
	}
	return c.GetObjectOutput, c.GetObjectError
}

func (c *testS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if params.ContinuationToken == nil {
		return c.ListObjectsV2Output[""], nil
	}
	return c.ListObjectsV2Output[*params.ContinuationToken], nil
}

// TestSaveText test that texts can be saved to s3
func TestSaveText(t *testing.T) {
	tt := []struct {
		name string
		c    *testS3Client
		err  error
		res  *hput.PutResult
		in   []*s3.PutObjectInput
	}{
		{
			name: "save new text",
			res:  &hput.PutResult{},
			c:    &testS3Client{},
			in: []*s3.PutObjectInput{{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBufferString("text"),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Text"},
			}},
		},
		{
			name: "save text already exists",
			res: &hput.PutResult{
				Overwrote: true,
			},
			c: &testS3Client{
				GetObjectError: &types.NoSuchKey{},
			},
			in: []*s3.PutObjectInput{{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBufferString("text"),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Text"},
			}},
		},
		{
			name: "save text error",
			res:  &hput.PutResult{},
			c:    &testS3Client{PutObjectInputError: errors.New("error")},
			err:  fmt.Errorf("failed to put string: %w", errors.New("error")),
			in: []*s3.PutObjectInput{{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBufferString("text"),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Text"},
			}},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := New(ctx, &testLogger{}, "bucket", S3ClientOption{client: test.c})
			assert.NoError(t, err)
			url, _ := url.Parse("http://localhost/path")
			r := &hput.PutResult{}
			err = s.SaveText(ctx, "text", *url, r)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.res, r)
			assert.Equal(t, test.in, test.c.PutObjectInput)
		})
	}
}

// TestSaveCode test that code can be saved to s3
func TestSaveCode(t *testing.T) {
	tt := []struct {
		name string
		c    *testS3Client
		err  error
		res  *hput.PutResult
		in   *s3.PutObjectInput
	}{
		{
			name: "save new code",
			res:  &hput.PutResult{},
			c:    &testS3Client{},
			in: &s3.PutObjectInput{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBufferString("code"),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Javascript"},
			},
		},
		{
			name: "save code already exists",
			res: &hput.PutResult{
				Overwrote: true,
			},
			c: &testS3Client{
				GetObjectError: &types.NoSuchKey{},
			},
			in: &s3.PutObjectInput{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBufferString("code"),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Javascript"},
			},
		},
		{
			name: "save text error",
			res:  &hput.PutResult{},
			c:    &testS3Client{PutObjectInputError: errors.New("error")},
			err:  fmt.Errorf("failed to put code: %w", errors.New("error")),
			in: &s3.PutObjectInput{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBufferString("code"),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Javascript"},
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := New(ctx, &testLogger{}, "bucket", S3ClientOption{client: test.c})
			assert.NoError(t, err)
			url, _ := url.Parse("http://localhost/path")
			r := &hput.PutResult{}
			err = s.SaveCode(ctx, "code", *url, r)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.res, r)
			assert.Equal(t, test.in, test.c.PutObjectInput[0])
		})
	}
}

// TestSaveBinary test that a binary can be saved to s3
func TestSaveBinary(t *testing.T) {
	tt := []struct {
		name string
		c    *testS3Client
		err  error
		res  *hput.PutResult
		in   *s3.PutObjectInput
	}{
		{
			name: "save new binary",
			res:  &hput.PutResult{},
			c:    &testS3Client{},
			in: &s3.PutObjectInput{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBuffer([]byte{255, 255, 255}),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Binary"},
			},
		},
		{
			name: "save code already exists",
			res: &hput.PutResult{
				Overwrote: true,
			},
			c: &testS3Client{
				GetObjectError: &types.NoSuchKey{},
			},
			in: &s3.PutObjectInput{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBuffer([]byte{255, 255, 255}),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Binary"},
			},
		},
		{
			name: "save text error",
			res:  &hput.PutResult{},
			c:    &testS3Client{PutObjectInputError: errors.New("error")},
			err:  fmt.Errorf("failed to put binary: %w", errors.New("error")),
			in: &s3.PutObjectInput{
				Bucket:   aws.String("bucket"),
				Body:     bytes.NewBuffer([]byte{255, 255, 255}),
				Key:      aws.String("/path"),
				Metadata: map[string]string{"input": "Binary"},
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := New(ctx, &testLogger{}, "bucket", S3ClientOption{client: test.c})
			assert.NoError(t, err)
			url, _ := url.Parse("http://localhost/path")
			r := &hput.PutResult{}
			err = s.SaveBinary(ctx, []byte{255, 255, 255}, *url, r)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.res, r)
			assert.Equal(t, test.in, test.c.PutObjectInput[0])
		})
	}
}

// TestGetRunnable verify that a runnable can be retrieved from s3
func TestGetRunnable(t *testing.T) {
	tt := []struct {
		name string
		c    *testS3Client
		r    hput.Runnable
		in   []*s3.GetObjectInput
		err  error
	}{
		{
			name: "get text that exists",
			c: &testS3Client{
				GetObjectOutput: &s3.GetObjectOutput{
					Body:     ioutil.NopCloser(bytes.NewBufferString("text")),
					Metadata: map[string]string{"input": "Text"},
				},
			},
			in: []*s3.GetObjectInput{{
				Bucket: aws.String("bucket"),
				Key:    aws.String("/path"),
			}},
			r: hput.Runnable{
				Path: "/path",
				Text: "text",
				Type: hput.Text,
			},
		},
		{
			name: "get binary that exists",
			c: &testS3Client{
				GetObjectOutput: &s3.GetObjectOutput{
					Body:     ioutil.NopCloser(bytes.NewBuffer([]byte{255, 255, 255})),
					Metadata: map[string]string{"input": "Binary"},
				},
			},
			in: []*s3.GetObjectInput{{
				Bucket: aws.String("bucket"),
				Key:    aws.String("/path"),
			}},
			r: hput.Runnable{
				Path:   "/path",
				Binary: []byte{255, 255, 255},
				Type:   hput.Binary,
			},
		},
		{
			name: "runnable doesn't exist",
			c: &testS3Client{
				GetObjectError: &types.NoSuchKey{},
			},
			in: []*s3.GetObjectInput{{
				Bucket: aws.String("bucket"),
				Key:    aws.String("/path"),
			}},
			r: hput.Runnable{},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := New(ctx, &testLogger{}, "bucket", S3ClientOption{client: test.c})
			assert.NoError(t, err)
			url, _ := url.Parse("http://localhost/path")
			r, err := s.GetRunnable(ctx, *url)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.r, r)
			assert.Equal(t, test.in, test.c.GetObjectInput)
		})
	}
}

func TestSendRunnables(t *testing.T) {
	tt := []struct {
		name string
		c    *testS3Client
		r    []hput.Runnable
	}{
		{
			name: "3 runnables to send from 2 pages",
			c: &testS3Client{
				GetObjectOutput: &s3.GetObjectOutput{
					Body:     ioutil.NopCloser(bytes.NewBufferString("text")),
					Metadata: map[string]string{"input": "Text"},
				},
				ListObjectsV2Output: map[string]*s3.ListObjectsV2Output{
					"": {
						Contents: []types.Object{
							{Key: aws.String("/path1")},
							{Key: aws.String("/path2")},
						},
						NextContinuationToken: aws.String("token"),
					},
					"token": {
						Contents: []types.Object{
							{Key: aws.String("/path3")},
						},
					},
				},
			},
			r: []hput.Runnable{
				{
					Path: "/path1",
					Text: "text",
					Type: hput.Text,
				},
				{
					Path: "/path2",
					Text: "text",
					Type: hput.Text,
				},
				{
					Path: "/path3",
					Text: "text",
					Type: hput.Text,
				},
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			s, err := New(ctx, &testLogger{}, "bucket", S3ClientOption{client: test.c})
			assert.NoError(t, err)
			runnablesChan := make(chan hput.Runnable)
			doneChan := make(chan bool)
			sentRunnables := []hput.Runnable{}
			go func() {
				err = s.SendRunnables(ctx, "/path", runnablesChan, doneChan)
			}()
			assert.NoError(t, err)
			for done := false; !done; {
				select {
				case r := <-runnablesChan:
					sentRunnables = append(sentRunnables, r)
				case <-doneChan:
					done = true
				}
			}
			assert.Equal(t, test.r, sentRunnables)
		})
	}
}
