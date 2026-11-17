package awscheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestValidate(t *testing.T) {
	ok := []struct {
		name string
		args map[string]any
	}{
		{"s3.versioning", map[string]any{"bucket": "b"}},
		{"s3.list_objects", map[string]any{"bucket": "b", "prefix": "p/"}},
		{"ec2.vpcs", map[string]any{}},
		{"ec2.subnets", map[string]any{"vpc_id": "vpc-1"}},
	}
	for _, c := range ok {
		if err := Validate(c.name, c.args); err != nil {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	bad := map[string]struct {
		name string
		args map[string]any
	}{
		"unknown call":     {"s3.delete_bucket", map[string]any{"bucket": "b"}},
		"missing argument": {"s3.versioning", map[string]any{}},
		"unknown argument": {"s3.versioning", map[string]any{"bucket": "b", "force": "yes"}},
		"not a string":     {"s3.versioning", map[string]any{"bucket": 5}},
	}
	for n, c := range bad {
		if err := Validate(c.name, c.args); err == nil {
			t.Errorf("%s: expected an error", n)
		}
	}
}

func TestEveryCallIsReadOnly(t *testing.T) {
	// The allowlist is the safety: a name that suggests a change never appears.
	for _, n := range Names() {
		_, op, _ := strings.Cut(n, ".")
		for _, bad := range []string{"create", "delete", "put", "update", "remove", "set", "attach", "detach", "terminate", "run"} {
			if strings.HasPrefix(op, bad) {
				t.Errorf("%s looks like a change", n)
			}
		}
	}
}

// The client is built by hand: nothing in the learner's AWS environment can
// redirect it or change its credentials.
func TestClientIgnoresTheLearnersAWSEnvironment(t *testing.T) {
	t.Setenv("AWS_PROFILE", "prod")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAREALKEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "real")
	t.Setenv("AWS_ENDPOINT_URL", "https://s3.amazonaws.com")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_CONFIG_FILE", "/does/not/exist")

	var mu sync.Mutex
	var auth []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		auth = append(auth, r.Header.Get("Authorization"))
		mu.Unlock()
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL, "123456789012")
	res, err := c.Call(context.Background(), "s3.bucket", map[string]any{"bucket": "nope"})
	if err != nil || res["exists"] != false {
		t.Fatalf("%v, %v", res, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(auth) == 0 {
		t.Fatal("the request did not go to the test server")
	}
	if !strings.Contains(auth[0], "Credential=123456789012/") || !strings.Contains(auth[0], "/us-east-1/") {
		t.Errorf("the account and region must be tflings's, got %q", auth[0])
	}
}

// The typed calls against a real Floci, in an account of their own. Skipped
// when Floci is not running (CI starts it).
func TestTypedCallsAgainstFloci(t *testing.T) {
	const endpoint = "http://127.0.0.1:4577"
	hc := &http.Client{Timeout: time.Second}
	resp, err := hc.Get(endpoint + "/_floci/health")
	if err != nil {
		t.Skip("Floci is not running on " + endpoint)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Skip("Floci is not healthy on " + endpoint)
	}
	account := "9" + strconv.FormatInt(time.Now().UnixNano()%100000000000, 10)
	for len(account) < 12 {
		account = "9" + account
	}
	ctx := context.Background()
	cfg := aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider(account, "test", "")}

	s3c := s3.NewFromConfig(cfg, func(o *s3.Options) { o.BaseEndpoint = aws.String(endpoint); o.UsePathStyle = true })
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = s3c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("chk")})
	must(err)
	_, err = s3c.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{Bucket: aws.String("chk"),
		VersioningConfiguration: &s3types.VersioningConfiguration{Status: s3types.BucketVersioningStatusEnabled}})
	must(err)
	_, err = s3c.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("chk"), Key: aws.String("dir/a.txt"), Body: strings.NewReader("hello")})
	must(err)
	_, err = s3c.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{Bucket: aws.String("chk"),
		Tagging: &s3types.Tagging{TagSet: []s3types.Tag{{Key: aws.String("env"), Value: aws.String("dev")}}}})
	must(err)

	ssmc := ssm.NewFromConfig(cfg, func(o *ssm.Options) { o.BaseEndpoint = aws.String(endpoint) })
	_, err = ssmc.PutParameter(ctx, &ssm.PutParameterInput{Name: aws.String("/chk/p"), Type: ssmtypes.ParameterTypeString, Value: aws.String("v1")})
	must(err)

	ddb := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(endpoint) })
	_, err = ddb.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:            aws.String("chk"),
		BillingMode:          ddbtypes.BillingModePayPerRequest,
		AttributeDefinitions: []ddbtypes.AttributeDefinition{{AttributeName: aws.String("id"), AttributeType: ddbtypes.ScalarAttributeTypeS}},
		KeySchema:            []ddbtypes.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: ddbtypes.KeyTypeHash}},
	})
	must(err)

	ec := ec2.NewFromConfig(cfg, func(o *ec2.Options) { o.BaseEndpoint = aws.String(endpoint) })
	vpc, err := ec.CreateVpc(ctx, &ec2.CreateVpcInput{CidrBlock: aws.String("10.9.0.0/16")})
	must(err)
	vpcID := aws.ToString(vpc.Vpc.VpcId)
	_, err = ec.CreateSubnet(ctx, &ec2.CreateSubnetInput{VpcId: aws.String(vpcID), CidrBlock: aws.String("10.9.1.0/24")})
	must(err)
	_, err = ec.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{GroupName: aws.String("chk-sg"), Description: aws.String("d"), VpcId: aws.String(vpcID)})
	must(err)

	iamc := iam.NewFromConfig(cfg, func(o *iam.Options) { o.BaseEndpoint = aws.String(endpoint) })
	_, err = iamc.CreateRole(ctx, &iam.CreateRoleInput{RoleName: aws.String("chk-role"),
		AssumeRolePolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Principal":{"Service":"ec2.amazonaws.com"}}]}`)})
	must(err)

	sm := secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) { o.BaseEndpoint = aws.String(endpoint) })
	_, err = sm.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: aws.String("chk/s"), SecretString: aws.String("s3cr3t")})
	must(err)

	c := New(endpoint, account)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := c.Call(ctx, name, args)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return res
	}
	b := map[string]any{"bucket": "chk"}
	if r := call("s3.bucket", b); r["exists"] != true {
		t.Errorf("s3.bucket: %v", r)
	}
	if r := call("s3.bucket", map[string]any{"bucket": "missing"}); r["exists"] != false {
		t.Errorf("a missing bucket: %v", r)
	}
	if r := call("s3.versioning", b); r["status"] != "Enabled" {
		t.Errorf("s3.versioning: %v", r)
	}
	if r := call("s3.object", map[string]any{"bucket": "chk", "key": "dir/a.txt"}); r["exists"] != true || r["body"] != "hello" || r["size"] != int64(5) {
		t.Errorf("s3.object: %v", r)
	}
	if r := call("s3.object", map[string]any{"bucket": "chk", "key": "nope"}); r["exists"] != false {
		t.Errorf("a missing object: %v", r)
	}
	if r := call("s3.list_objects", map[string]any{"bucket": "chk", "prefix": "dir/"}); len(r["keys"].([]any)) != 1 {
		t.Errorf("s3.list_objects: %v", r)
	}
	if r := call("s3.tags", b); r["tags"].(map[string]any)["env"] != "dev" {
		t.Errorf("s3.tags: %v", r)
	}
	if r := call("ssm.parameter", map[string]any{"name": "/chk/p"}); r["value"] != "v1" || r["type"] != "String" {
		t.Errorf("ssm.parameter: %v", r)
	}
	if r := call("ssm.parameter", map[string]any{"name": "/nope"}); r["exists"] != false {
		t.Errorf("a missing parameter: %v", r)
	}
	if r := call("dynamodb.table", map[string]any{"name": "chk"}); r["hash_key"] != "id" || r["billing_mode"] != "PAY_PER_REQUEST" {
		t.Errorf("dynamodb.table: %v", r)
	}
	vpcs := call("ec2.vpcs", nil)["vpcs"].([]any)
	nonDefault := 0
	for _, v := range vpcs {
		if v.(map[string]any)["is_default"] == false {
			nonDefault++
		}
	}
	if len(vpcs) != 2 || nonDefault != 1 {
		t.Errorf("a fresh account has a default VPC plus the one made here: %v", vpcs)
	}
	if r := call("ec2.subnets", map[string]any{"vpc_id": vpcID}); len(r["subnets"].([]any)) != 1 {
		t.Errorf("ec2.subnets: %v", r)
	}
	if r := call("ec2.security_groups", map[string]any{"vpc_id": vpcID}); len(r["groups"].([]any)) < 1 {
		t.Errorf("ec2.security_groups: %v", r)
	}
	if r := call("iam.role", map[string]any{"name": "chk-role"}); r["exists"] != true || !strings.Contains(r["assume_role_policy"].(string), "ec2.amazonaws.com") {
		t.Errorf("iam.role: %v", r)
	}
	if r := call("secretsmanager.secret", map[string]any{"name": "chk/s"}); r["value"] != "s3cr3t" {
		t.Errorf("secretsmanager.secret: %v", r)
	}
}

func TestMutationAllowlist(t *testing.T) {
	if err := ValidateMutation("ssm.put_parameter", map[string]any{"name": "/a", "value": "b"}); err != nil {
		t.Errorf("a valid seeding call was refused: %v", err)
	}
	for name, args := range map[string]map[string]any{
		"ssm.delete_everything": {"name": "/a"},
		"ssm.put_parameter":     {"name": "/a"},
	} {
		if ValidateMutation(name, args) == nil {
			t.Errorf("%s %v should be refused", name, args)
		}
	}
	if ValidateMutation("ssm.put_parameter", map[string]any{"name": "/a", "value": "b", "extra": "c"}) == nil {
		t.Error("an unknown argument should be refused")
	}
	// check.yaml can never name a seeding call.
	for _, n := range MutationNames() {
		if Validate(n, map[string]any{}) == nil {
			t.Errorf("%s must not be usable in a check", n)
		}
	}
}
