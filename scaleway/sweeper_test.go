package scaleway

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"
	"strings"
	"testing"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
)

// isTestResource returns true if given resource identifier is from terraform test
// identifier should be resource name but some resource don't have names
// return true if identifier match regex "tf[-_]test"
// common used prefixes are "tf_tests", "tf_test", "tf-tests", "tf-test"
func isTestResource(identifier string) bool {
	return len(identifier) >= len("tf_test") &&
		strings.HasPrefix(identifier, "tf") &&
		(identifier[2] == '_' || identifier[2] == '-') &&
		identifier[3:7] == "test"
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func SweepRegions(regions []scw.Region, f func(scwClient *scw.Client, region scw.Region) error) error {
	zones := make([]scw.Zone, len(regions))
	for i, region := range regions {
		zones[i] = region.GetZones()[0]
	}

	return tests.SweepZones(zones, func(scwClient *scw.Client, zone scw.Zone) error {
		r, _ := zone.Region()
		return f(scwClient, r)
	})
}

// sharedClientForZone returns a Scaleway client needed for the sweeper
// functions for a given zone
func sharedClientForZone(zone scw.Zone) (*scw.Client, error) {
	ctx := context.Background()
	meta, err := buildMeta(ctx, &meta2.metaConfig{
		terraformVersion: "terraform-tests",
		forceZone:        zone,
	})
	if err != nil {
		return nil, err
	}
	return meta.GetScwClient(), nil
}

// sharedS3ClientForRegion returns a common S3 client needed for the sweeper
func sharedS3ClientForRegion(region scw.Region) (*s3.S3, error) {
	ctx := context.Background()
	meta, err := buildMeta(ctx, &meta2.metaConfig{
		terraformVersion: "terraform-tests",
		forceZone:        region.GetZones()[0],
	})
	if err != nil {
		return nil, err
	}
	return newS3ClientFromMeta(meta, region.String())
}

func TestIsTestResource(t *testing.T) {
	assert.True(t, isTestResource("tf_tests_mnq_sqs_queue_default_project"))
}
