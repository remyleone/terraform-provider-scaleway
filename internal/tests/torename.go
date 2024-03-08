package tests

import (
	"context"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/scaleway/scaleway-sdk-go/scw"
	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/object"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// IsTestResource returns true if given resource identifier is from terraform test
// identifier should be resource name but some resource don't have names
// return true if identifier match regex "tf[-_]test"
// common used prefixes are "tf_tests", "tf_test", "tf-tests", "tf-test"
func IsTestResource(identifier string) bool {
	return len(identifier) >= len("tf_test") &&
		strings.HasPrefix(identifier, "tf") &&
		(identifier[2] == '_' || identifier[2] == '-') &&
		identifier[3:7] == "test"
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

// SharedS3ClientForRegion returns a common S3 client needed for the sweeper
func SharedS3ClientForRegion(region scw.Region) (*s3.S3, error) {
	ctx := context.Background()
	meta, err := meta2.BuildMeta(ctx, &meta2.MetaConfig{
		TerraformVersion: "terraform-tests",
		ForceZone:        region.GetZones()[0],
	})
	if err != nil {
		return nil, err
	}
	return object.NewS3ClientFromMeta(meta, region.String())
}

func TestIsTestResource(t *testing.T) {
	assert.True(t, IsTestResource("tf_tests_mnq_sqs_queue_default_project"))
}
