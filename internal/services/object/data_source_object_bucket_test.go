package object

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAccScalewayDataSourceObjectBucket_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	bucketName := sdkacctest.RandomWithPrefix("test-acc-scaleway-object-bucket")
	objectBucketTestDefaultRegion, _ := tt.meta.GetScwClient().GetDefaultRegion()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayObjectBucketDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "scaleway_object_bucket" "base-01" {
					name = "%s"
					region = "%s"
					tags = {
						foo = "bar"
					}
				}

				data "scaleway_object_bucket" "by-id" {
					name = scaleway_object_bucket.base-01.id
				}
				`, bucketName, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.base-01", true),
					resource.TestCheckResourceAttr("data.scaleway_object_bucket.by-id", "name", bucketName),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "scaleway_object_bucket" "base-01" {
					name = "%s"
					region = "%s"
					tags = {
						foo = "bar"
					}
				}

				data "scaleway_object_bucket" "by-name" {
					name = scaleway_object_bucket.base-01.name
				}
				`, bucketName, objectBucketTestDefaultRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.base-01", true),
					resource.TestCheckResourceAttr("data.scaleway_object_bucket.by-name", "name", bucketName),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "scaleway_object_bucket" "base-01" {
					name = "%s"
					region = "%s"
					tags = {
						foo = "bar"
					}
				}

				data "scaleway_object_bucket" "by-name" {
					name = scaleway_object_bucket.base-01.name
				}
				`, bucketName, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.base-01", true),
					resource.TestCheckResourceAttr("data.scaleway_object_bucket.by-name", "name", bucketName),
				),
				ExpectError: regexp.MustCompile("failed getting Object Storage bucket"),
			},
		},
	})
}

func TestAccScalewayDataSourceObjectBucket_ProjectIDAllowed(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	bucketName := sdkacctest.RandomWithPrefix("test-acc-scaleway-object-bucket")

	project, iamAPIKey, terminateFakeSideProject, err := createFakeSideProject(tt)
	require.NoError(t, err)

	ctx := context.Background()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: fakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayObjectBucketDestroy(tt),
		),
		Steps: []resource.TestStep{
			// Create a bucket from the main provider into the side project and read it from the side provider
			// The side provider should only be able to read the bucket from the side project
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "base" {
						name = "%[1]s"
						project_id = "%[2]s"
						region = "%[3]s"
					}

					data "scaleway_object_bucket" "selected" {
						name = scaleway_object_bucket.base.id
						provider = side
					}
				`,
					bucketName,
					project.ID,
					objectTestsMainRegion,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.base", false),
					resource.TestCheckResourceAttr("data.scaleway_object_bucket.selected", "name", bucketName),
					resource.TestCheckResourceAttr("data.scaleway_object_bucket.selected", "project_id", project.ID),
				),
			},
		},
	})
}

func TestAccScalewayDataSourceObjectBucket_ProjectIDForbidden(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	bucketName := sdkacctest.RandomWithPrefix("test-acc-scaleway-object-bucket")

	project, iamAPIKey, terminateFakeSideProject, err := createFakeSideProject(tt)
	require.NoError(t, err)

	ctx := context.Background()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: fakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayObjectBucketDestroy(tt),
		),
		Steps: []resource.TestStep{
			// The side provider should not be able to read the bucket from the main project
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "base" {
						name = "%[1]s"
						region = "%[3]s"
					}

					data "scaleway_object_bucket" "selected" {
						name = scaleway_object_bucket.base.id
						provider = side
					}
				`,
					bucketName,
					project.ID,
					objectTestsMainRegion,
				),
				Check:       testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.base", false),
				ExpectError: regexp.MustCompile("failed getting Object Storage bucket"),
			},
		},
	})
}
