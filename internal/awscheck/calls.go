package awscheck

import (
	"context"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// maxBody is how much of an object or a secret a check can read.
const maxBody = 64 << 10

// Every call returns plain maps, slices, strings, bools and numbers, so that CEL
// can read them without knowing about SDK types. A thing that does not exist is
// {"exists": false}, not an error: "the bucket should exist" is a check, and a
// missing bucket is the learner's problem, not ours.

func (c *Client) s3Bucket(ctx context.Context, a Args) (map[string]any, error) {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(a.str("bucket"))})
	if notFound(err) {
		return map[string]any{"exists": false}, nil
	}
	return map[string]any{"exists": err == nil}, err
}

func (c *Client) s3Versioning(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.s3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(a.str("bucket"))})
	if notFound(err) {
		return map[string]any{"exists": false, "status": ""}, nil
	}
	if err != nil {
		return nil, err
	}
	status := string(out.Status)
	if status == "" {
		status = "Disabled" // a bucket that never had versioning says nothing
	}
	return map[string]any{"exists": true, "status": status}, nil
}

func (c *Client) s3Object(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(a.str("bucket")), Key: aws.String(a.str("key"))})
	if notFound(err) {
		return map[string]any{"exists": false}, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = out.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(out.Body, maxBody))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"exists":       true,
		"size":         aws.ToInt64(out.ContentLength),
		"content_type": aws.ToString(out.ContentType),
		"body":         string(body),
	}, nil
}

func (c *Client) s3List(ctx context.Context, a Args) (map[string]any, error) {
	in := &s3.ListObjectsV2Input{Bucket: aws.String(a.str("bucket"))}
	if p := a.str("prefix"); p != "" {
		in.Prefix = aws.String(p)
	}
	keys := []any{}
	pager := s3.NewListObjectsV2Paginator(c.s3, in)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if notFound(err) {
			return map[string]any{"exists": false, "keys": keys}, nil
		}
		if err != nil {
			return nil, err
		}
		for _, o := range page.Contents {
			keys = append(keys, aws.ToString(o.Key))
		}
	}
	return map[string]any{"exists": true, "keys": keys}, nil
}

func (c *Client) s3Tags(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.s3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(a.str("bucket"))})
	tags := map[string]any{}
	if notFound(err) {
		return map[string]any{"tags": tags}, nil
	}
	if err != nil {
		return nil, err
	}
	for _, t := range out.TagSet {
		tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return map[string]any{"tags": tags}, nil
}

func (c *Client) ssmParameter(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.ssm.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(a.str("name")), WithDecryption: aws.Bool(true)})
	if notFound(err) {
		return map[string]any{"exists": false}, nil
	}
	if err != nil {
		return nil, err
	}
	p := out.Parameter
	return map[string]any{"exists": true, "type": string(p.Type), "value": aws.ToString(p.Value)}, nil
}

func (c *Client) dynamoTable(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.dynamo.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(a.str("name"))})
	if notFound(err) {
		return map[string]any{"exists": false}, nil
	}
	if err != nil {
		return nil, err
	}
	t := out.Table
	res := map[string]any{"exists": true, "status": string(t.TableStatus), "billing_mode": "PROVISIONED"}
	if t.BillingModeSummary != nil {
		res["billing_mode"] = string(t.BillingModeSummary.BillingMode)
	}
	attrs := []any{}
	for _, d := range t.AttributeDefinitions {
		attrs = append(attrs, aws.ToString(d.AttributeName))
	}
	res["attributes"] = attrs
	for _, k := range t.KeySchema {
		if k.KeyType == "HASH" {
			res["hash_key"] = aws.ToString(k.AttributeName)
		} else {
			res["range_key"] = aws.ToString(k.AttributeName)
		}
	}
	return res, nil
}

func tagMap(tags []ec2types.Tag) map[string]any {
	m := map[string]any{}
	for _, t := range tags {
		m[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return m
}

// ec2.vpcs lists the VPCs of the account, including the default VPC that every
// account starts with: a check that counts VPCs filters on is_default.
func (c *Client) ec2VPCs(ctx context.Context, _ Args) (map[string]any, error) {
	out, err := c.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}
	vpcs := []any{}
	for _, v := range out.Vpcs {
		vpcs = append(vpcs, map[string]any{
			"id": aws.ToString(v.VpcId), "cidr_block": aws.ToString(v.CidrBlock),
			"is_default": aws.ToBool(v.IsDefault), "tags": tagMap(v.Tags),
		})
	}
	return map[string]any{"vpcs": vpcs}, nil
}

func vpcFilter(a Args) []ec2types.Filter {
	if v := a.str("vpc_id"); v != "" {
		return []ec2types.Filter{{Name: aws.String("vpc-id"), Values: []string{v}}}
	}
	return nil
}

func (c *Client) ec2Subnets(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{Filters: vpcFilter(a)})
	if err != nil {
		return nil, err
	}
	subnets := []any{}
	for _, s := range out.Subnets {
		subnets = append(subnets, map[string]any{
			"id": aws.ToString(s.SubnetId), "vpc_id": aws.ToString(s.VpcId), "cidr_block": aws.ToString(s.CidrBlock),
			"availability_zone": aws.ToString(s.AvailabilityZone), "tags": tagMap(s.Tags),
		})
	}
	return map[string]any{"subnets": subnets}, nil
}

func (c *Client) ec2SecurityGroups(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{Filters: vpcFilter(a)})
	if err != nil {
		return nil, err
	}
	groups := []any{}
	for _, g := range out.SecurityGroups {
		ingress := []any{}
		for _, p := range g.IpPermissions {
			cidrs := []any{}
			for _, r := range p.IpRanges {
				cidrs = append(cidrs, aws.ToString(r.CidrIp))
			}
			ingress = append(ingress, map[string]any{
				"protocol": aws.ToString(p.IpProtocol), "from_port": int64(aws.ToInt32(p.FromPort)),
				"to_port": int64(aws.ToInt32(p.ToPort)), "cidr_blocks": cidrs,
			})
		}
		groups = append(groups, map[string]any{
			"id": aws.ToString(g.GroupId), "name": aws.ToString(g.GroupName), "vpc_id": aws.ToString(g.VpcId),
			"description": aws.ToString(g.Description), "ingress": ingress, "tags": tagMap(g.Tags),
		})
	}
	return map[string]any{"groups": groups}, nil
}

func (c *Client) iamRole(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.iam.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(a.str("name"))})
	if notFound(err) {
		return map[string]any{"exists": false}, nil
	}
	if err != nil {
		return nil, err
	}
	policy := aws.ToString(out.Role.AssumeRolePolicyDocument)
	if dec, err := url.QueryUnescape(policy); err == nil {
		policy = dec // the API returns the document URL-encoded
	}
	attached := []any{}
	pager := iam.NewListAttachedRolePoliciesPaginator(c.iam, &iam.ListAttachedRolePoliciesInput{RoleName: aws.String(a.str("name"))})
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range page.AttachedPolicies {
			attached = append(attached, aws.ToString(p.PolicyArn))
		}
	}
	return map[string]any{"exists": true, "arn": aws.ToString(out.Role.Arn), "assume_role_policy": policy, "attached_policy_arns": attached}, nil
}

func (c *Client) secret(ctx context.Context, a Args) (map[string]any, error) {
	out, err := c.secrets.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(a.str("name"))})
	if notFound(err) {
		return map[string]any{"exists": false}, nil
	}
	if err != nil {
		return nil, err
	}
	val := aws.ToString(out.SecretString)
	if len(val) > maxBody {
		val = val[:maxBody]
	}
	return map[string]any{"exists": true, "value": val}, nil
}
