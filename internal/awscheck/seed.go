package awscheck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// The calls above only read, and a check can name only those. A fix exercise
// also has to make a mess on purpose (change a parameter "by hand"), and it
// does that with the calls below. They are a separate, small allowlist, used by
// the seeding of an exercise (written by us, embedded in the binary) and never
// by check.yaml.

type mutation struct {
	required []string
	run      func(c *Client, ctx context.Context, a Args) error
}

var mutations = map[string]mutation{
	"ssm.put_parameter":    {required: []string{"name", "value"}, run: (*Client).putParameter},
	"ssm.delete_parameter": {required: []string{"name"}, run: (*Client).deleteParameter},
	"s3.create_bucket":     {required: []string{"bucket"}, run: (*Client).createBucket},
	"s3.put_object":        {required: []string{"bucket", "key", "body"}, run: (*Client).putObject},
}

// MutationNames lists the calls a seed can use.
func MutationNames() []string {
	out := make([]string, 0, len(mutations))
	for n := range mutations {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ValidateMutation says whether name and args describe a seeding call.
func ValidateMutation(name string, args map[string]any) error {
	m, ok := mutations[name]
	if !ok {
		return fmt.Errorf("seed call %q does not exist (known: %s)", name, strings.Join(MutationNames(), ", "))
	}
	for _, r := range m.required {
		if _, ok := args[r].(string); !ok {
			return fmt.Errorf("seed call %s needs the string argument %q", name, r)
		}
	}
	for k := range args {
		found := false
		for _, r := range m.required {
			found = found || r == k
		}
		if !found {
			return fmt.Errorf("seed call %s has no argument %q", name, k)
		}
	}
	return nil
}

// Mutate runs one seeding call.
func (c *Client) Mutate(ctx context.Context, name string, args map[string]any) error {
	if err := ValidateMutation(name, args); err != nil {
		return err
	}
	return mutations[name].run(c, ctx, Args(args))
}

// putParameter overwrites a String parameter, as someone with console access
// would. It keeps the type of an existing parameter.
func (c *Client) putParameter(ctx context.Context, a Args) error {
	typ := ssmType("String")
	if cur, err := c.ssm.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(a.str("name"))}); err == nil && cur.Parameter != nil {
		typ = cur.Parameter.Type
	}
	_, err := c.ssm.PutParameter(ctx, &ssm.PutParameterInput{
		Name: aws.String(a.str("name")), Value: aws.String(a.str("value")), Type: typ, Overwrite: aws.Bool(true),
	})
	return err
}

func ssmType(s string) ssmtypes.ParameterType { return ssmtypes.ParameterType(s) }

// deleteParameter removes a parameter, as someone with console access would.
func (c *Client) deleteParameter(ctx context.Context, a Args) error {
	_, err := c.ssm.DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: aws.String(a.str("name"))})
	return err
}

// createBucket makes a bucket the way someone with console access would, so that
// Terraform knows nothing about it.
func (c *Client) createBucket(ctx context.Context, a Args) error {
	_, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(a.str("bucket"))})
	return err
}

// putObject writes a text object, for example a file somebody left in a bucket,
// or the lock file of a run that was killed.
func (c *Client) putObject(ctx context.Context, a Args) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(a.str("bucket")), Key: aws.String(a.str("key")), Body: strings.NewReader(a.str("body")),
	})
	return err
}
