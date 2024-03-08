package object_test

import (
	"errors"
	"fmt"
	scaleway "github.com/scaleway/terraform-provider-scaleway/v2/internal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/object"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests/checks"
	"regexp"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	ResourcePrefix   = "tf-acc-test"
	resourceTestName = "scaleway_object_bucket_website_configuration.test"
)

func TestAccScalewayObjectBucketWebsiteConfiguration_Basic(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(ResourcePrefix)
	resourceName := resourceTestName

	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ErrorCheck:        scaleway.ErrorCheck(t, scaleway.EndpointsID),
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayObjectBucketWebsiteConfigurationDestroy(tt),
			checks.TestAccCheckScalewayObjectBucketDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
			  		resource "scaleway_object_bucket" "test" {
						name = %[1]q
						region = %[2]q
						acl  = "public-read"
						tags = {
							TestName = "TestAccScalewayObjectBucketWebsiteConfiguration_Basic"
						}
					}
				
				  	resource "scaleway_object_bucket_website_configuration" "test" {
						bucket = scaleway_object_bucket.test.id
						index_document {
						  suffix = "index.html"
						}
				  	}
				`, rName, object.ObjectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					checks.TestAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.test", true),
					testAccCheckScalewayObjectBucketWebsiteConfigurationExists(tt, resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "bucket", "scaleway_object_bucket.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "index_document.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "index_document.0.suffix", "index.html"),
					resource.TestCheckResourceAttrSet(resourceName, "website_domain"),
					resource.TestCheckResourceAttrSet(resourceName, "website_endpoint"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccScalewayObjectBucketWebsiteConfiguration_WithPolicy(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(ResourcePrefix)
	resourceName := resourceTestName

	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ErrorCheck:        scaleway.ErrorCheck(t, scaleway.EndpointsID),
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayObjectBucketWebsiteConfigurationDestroy(tt),
			checks.TestAccCheckScalewayObjectBucketDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
			  		resource "scaleway_object_bucket" "test" {
						name = %[1]q
						region = %[2]q
						acl  = "public-read"
						tags = {
							TestName = "TestAccScalewayObjectBucketWebsiteConfiguration_WithPolicy"
						}
					}

					resource "scaleway_object_bucket_policy" "main" {
						bucket = scaleway_object_bucket.test.id
						policy = jsonencode(
						{
							"Version" = "2012-10-17",
							"Id" = "MyPolicy",
							"Statement" = [
							{
							   "Sid" = "GrantToEveryone",
							   "Effect" = "Allow",
							   "Principal" = "*",
							   "Action" = [
								  "s3:GetObject"
							   ],
							   "Resource":[
								  "%[1]s/*"
							   ]
							}
							]
						})
					}
				
				  	resource "scaleway_object_bucket_website_configuration" "test" {
						bucket = scaleway_object_bucket.test.id
						index_document {
						  suffix = "index.html"
						}
				  	}
				`, rName, object.ObjectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					checks.TestAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.test", true),
					testAccCheckScalewayObjectBucketWebsiteConfigurationExists(tt, resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "bucket", "scaleway_object_bucket.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "index_document.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "index_document.0.suffix", "index.html"),
					resource.TestCheckResourceAttrSet(resourceName, "website_domain"),
					resource.TestCheckResourceAttrSet(resourceName, "website_endpoint"),
				),
				ExpectNonEmptyPlan: !*tests.UpdateCassettes,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccScalewayObjectBucketWebsiteConfiguration_Update(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(ResourcePrefix)
	resourceName := resourceTestName

	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ErrorCheck:        scaleway.ErrorCheck(t, scaleway.EndpointsID),
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayObjectBucketWebsiteConfigurationDestroy(tt),
			checks.TestAccCheckScalewayObjectBucketDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
			  		resource "scaleway_object_bucket" "test" {
						name = %[1]q
						region = %[2]q
						acl  = "public-read"
						tags = {
							TestName = "TestAccScalewayObjectBucketWebsiteConfiguration_Update"
						}
					}

				  	resource "scaleway_object_bucket_website_configuration" "test" {
						bucket = scaleway_object_bucket.test.id
						index_document {
						  suffix = "index.html"
						}
				  	}
				`, rName, object.ObjectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					checks.TestAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.test", true),
					testAccCheckScalewayObjectBucketWebsiteConfigurationExists(tt, resourceName),
				),
			},
			{
				Config: fmt.Sprintf(`
			  		resource "scaleway_object_bucket" "test" {
						name = %[1]q
						region = %[2]q
						acl  = "public-read"
						tags = {
							TestName = "TestAccScalewayObjectBucketWebsiteConfiguration_Update"
						}
					}

				  	resource "scaleway_object_bucket_website_configuration" "test" {
						bucket = scaleway_object_bucket.test.id
						index_document {
						  suffix = "index.html"
						}

						error_document {
							key = "error.html"
						}
				  	}
				`, rName, object.ObjectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					checks.TestAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.test", true),
					testAccCheckScalewayObjectBucketWebsiteConfigurationExists(tt, resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "bucket", "scaleway_object_bucket.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "index_document.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "index_document.0.suffix", "index.html"),
					resource.TestCheckResourceAttr(resourceName, "error_document.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "error_document.0.key", "error.html"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccScalewayObjectBucketWebsiteConfiguration_WithBucketName(t *testing.T) {
	rName := sdkacctest.RandomWithPrefix(ResourcePrefix)
	resourceName := resourceTestName

	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ErrorCheck:        scaleway.ErrorCheck(t, scaleway.EndpointsID),
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayObjectBucketWebsiteConfigurationDestroy(tt),
			checks.TestAccCheckScalewayObjectBucketDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
			  		resource "scaleway_object_bucket" "test" {
						name = %[1]q
						region = %[2]q
						acl  = "public-read"
					}
				
				  	resource "scaleway_object_bucket_website_configuration" "test" {
						bucket = scaleway_object_bucket.test.name
						index_document {
						  suffix = "index.html"
						}
				  	}
				`, rName, object.ObjectTestsMainRegion),
				ExpectError: regexp.MustCompile("couldn't read bucket: NoSuchBucket: The specified bucket does not exist"),
			},
			{
				Config: fmt.Sprintf(`
			  		resource "scaleway_object_bucket" "test" {
						name = %[1]q
						region = %[2]q
						acl  = "public-read"
					}
				
				  	resource "scaleway_object_bucket_website_configuration" "test" {
						bucket = scaleway_object_bucket.test.name
						region = %[2]q
						index_document {
						  suffix = "index.html"
						}
				  	}
				`, rName, object.ObjectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					checks.TestAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.test", true),
					testAccCheckScalewayObjectBucketWebsiteConfigurationExists(tt, resourceName),
				),
			},
		},
	})
}

func testAccCheckScalewayObjectBucketWebsiteConfigurationDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "scaleway_object_bucket_website_configuration" {
				continue
			}

			regionalID := locality.ExpandRegionalID(rs.Primary.ID)
			bucket := regionalID.ID
			bucketRegion := regionalID.Region

			conn, err := object.NewS3ClientFromMeta(tt.GetMeta(), bucketRegion.String())
			if err != nil {
				return err
			}

			input := &s3.GetBucketWebsiteInput{
				Bucket: aws.String(bucket),
			}

			output, err := conn.GetBucketWebsite(input)

			if tfawserr.ErrCodeEquals(err, s3.ErrCodeNoSuchBucket, object.ErrCodeNoSuchWebsiteConfiguration) {
				continue
			}

			if err != nil {
				return fmt.Errorf("error getting object bucket website configuration (%s): %w", rs.Primary.ID, err)
			}

			if output != nil {
				return fmt.Errorf("object bucket website configuration (%s) still exists", rs.Primary.ID)
			}
		}

		return nil
	}
}

func testAccCheckScalewayObjectBucketWebsiteConfigurationExists(tt *tests.TestTools, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[resourceName]
		if rs == nil {
			return errors.New("resource not found")
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource (%s) ID not set", resourceName)
		}

		regionalID := locality.ExpandRegionalID(rs.Primary.ID)
		bucket := regionalID.ID
		bucketRegion := regionalID.Region

		conn, err := object.NewS3ClientFromMeta(tt.GetMeta(), bucketRegion.String())
		if err != nil {
			return err
		}

		input := &s3.GetBucketWebsiteInput{
			Bucket: aws.String(bucket),
		}

		output, err := conn.GetBucketWebsite(input)
		if err != nil {
			return fmt.Errorf("error getting object bucket website configuration (%s): %w", rs.Primary.ID, err)
		}

		if output == nil {
			return fmt.Errorf("object bucket website configuration (%s) not found", rs.Primary.ID)
		}

		return nil
	}
}
