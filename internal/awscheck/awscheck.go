// Package awscheck reads what an apply exercise built in Floci, for the `aws:`
// checks of check.yaml. It offers a fixed set of typed, read-only calls: a
// check can name only calls that are registered here, so a check can never
// change anything, and a new kind of check is a few lines of Go and a test.
//
// The client never reads the learner's AWS environment or files: its
// configuration is built by hand (fake credentials, the Floci endpoint), not
// loaded with the SDK's default chain.
package awscheck

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/smithy-go"
)

// Client reads one fake AWS account in Floci.
type Client struct {
	s3      *s3.Client
	ssm     *ssm.Client
	dynamo  *dynamodb.Client
	ec2     *ec2.Client
	iam     *iam.Client
	secrets *secretsmanager.Client
}

// New returns a client for the account (its 12-digit access key ID) on the
// Floci at endpoint, e.g. "http://127.0.0.1:4577".
func New(endpoint, account string) *Client {
	cfg := aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider(account, "test", ""),
		RetryMaxAttempts: 2,
	}
	return &Client{
		s3:      s3.NewFromConfig(cfg, func(o *s3.Options) { o.BaseEndpoint = aws.String(endpoint); o.UsePathStyle = true }),
		ssm:     ssm.NewFromConfig(cfg, func(o *ssm.Options) { o.BaseEndpoint = aws.String(endpoint) }),
		dynamo:  dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(endpoint) }),
		ec2:     ec2.NewFromConfig(cfg, func(o *ec2.Options) { o.BaseEndpoint = aws.String(endpoint) }),
		iam:     iam.NewFromConfig(cfg, func(o *iam.Options) { o.BaseEndpoint = aws.String(endpoint) }),
		secrets: secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) { o.BaseEndpoint = aws.String(endpoint) }),
	}
}

// Args are the arguments of a call, as written in check.yaml.
type Args map[string]any

func (a Args) str(name string) string {
	s, _ := a[name].(string)
	return s
}

type call struct {
	required []string
	optional []string
	run      func(c *Client, ctx context.Context, a Args) (map[string]any, error)
}

// calls is the allowlist.
var calls = map[string]call{
	"s3.bucket":             {required: []string{"bucket"}, run: (*Client).s3Bucket},
	"s3.versioning":         {required: []string{"bucket"}, run: (*Client).s3Versioning},
	"s3.object":             {required: []string{"bucket", "key"}, run: (*Client).s3Object},
	"s3.list_objects":       {required: []string{"bucket"}, optional: []string{"prefix"}, run: (*Client).s3List},
	"s3.tags":               {required: []string{"bucket"}, run: (*Client).s3Tags},
	"ssm.parameter":         {required: []string{"name"}, run: (*Client).ssmParameter},
	"dynamodb.table":        {required: []string{"name"}, run: (*Client).dynamoTable},
	"ec2.vpcs":              {run: (*Client).ec2VPCs},
	"ec2.subnets":           {optional: []string{"vpc_id"}, run: (*Client).ec2Subnets},
	"ec2.security_groups":   {optional: []string{"vpc_id"}, run: (*Client).ec2SecurityGroups},
	"iam.role":              {required: []string{"name"}, run: (*Client).iamRole},
	"secretsmanager.secret": {required: []string{"name"}, run: (*Client).secret},
}

// Names lists the calls a check can use.
func Names() []string {
	out := make([]string, 0, len(calls))
	for n := range calls {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Validate says whether name and args describe a call that exists: a name that
// is not in the allowlist, a missing argument or an unknown one are errors, so
// that a mistake in a check is found when the check is loaded.
func Validate(name string, args map[string]any) error {
	c, ok := calls[name]
	if !ok {
		return fmt.Errorf("aws call %q does not exist (known: %s)", name, strings.Join(Names(), ", "))
	}
	allowed := map[string]bool{}
	for _, r := range c.required {
		allowed[r] = true
		if _, ok := args[r]; !ok {
			return fmt.Errorf("aws call %s needs the argument %q", name, r)
		}
	}
	for _, o := range c.optional {
		allowed[o] = true
	}
	for k, v := range args {
		if !allowed[k] {
			return fmt.Errorf("aws call %s has no argument %q", name, k)
		}
		if _, ok := v.(string); !ok {
			return fmt.Errorf("aws call %s: argument %q must be a string", name, k)
		}
	}
	return nil
}

// Call runs one registered call and returns its result as plain data, for CEL.
func (c *Client) Call(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if err := Validate(name, args); err != nil {
		return nil, err
	}
	return calls[name].run(c, ctx, Args(args))
}

// notFound reports whether err says that the thing asked about does not exist.
func notFound(err error) bool {
	var api smithy.APIError
	if !errors.As(err, &api) {
		return false
	}
	switch code := api.ErrorCode(); {
	case code == "NoSuchBucket", code == "NotFound", code == "NoSuchKey", code == "ParameterNotFound",
		code == "ResourceNotFoundException", code == "NoSuchEntity", code == "NoSuchTagSet",
		strings.HasSuffix(code, ".NotFound"), code == "404":
		return true
	}
	return false
}
