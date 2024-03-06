package object_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayObjectBucketACL_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	testBucketName := sdkacctest.RandomWithPrefix("test-acc-scw-object-acl-basic")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayObjectBucketDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%s"
						region = "%s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.id
						acl = "private"
					}
					`, testBucketName, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.main", true),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "bucket", testBucketName),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "acl", "private"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%s"
						region = "%s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.id
						acl = "public-read"
					}
					`, testBucketName, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.main", true),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "bucket", testBucketName),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "acl", "public-read"),
				),
			},
		},
	})
}

func TestAccScalewayObjectBucketACL_Grantee(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	testBucketName := sdkacctest.RandomWithPrefix("test-acc-scw-object-acl-grantee")

	ownerID := "105bdce1-64c0-48ab-899d-868455867ecf"
	ownerIDChild := "50ab77d5-56bd-4981-a118-4e0fa5309b59"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayObjectBucketDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%[1]s"
						region = "%[3]s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.id
						access_control_policy {
						  	grant {
								grantee {
									id   = "%[2]s"
									type = "CanonicalUser"
								}
								permission = "FULL_CONTROL"
						  	}
						
						  	grant {
								grantee {
							  		id   = "%[2]s"
							  		type = "CanonicalUser"
								}
								permission = "WRITE"
						  	}
						
						  	owner {
								id = "%[2]s"
						  	}
						}
					}
					`, testBucketName, ownerID, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.main", true),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "bucket", testBucketName),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%[1]s"
						region = "%[4]s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.id
						access_control_policy {
						  	grant {
								grantee {
									id   = "%[2]s"
									type = "CanonicalUser"
								}
								permission = "FULL_CONTROL"
						  	}
						
						  	grant {
								grantee {
							  		id   = "%[2]s"
							  		type = "CanonicalUser"
								}
								permission = "WRITE"
						  	}

							grant {
								grantee {
								  	id   = "%[3]s"
								  	type = "CanonicalUser"
								}
								permission = "FULL_CONTROL"
							}
						
						  	owner {
								id = "%[2]s"
						  	}
						}
					}
				`, testBucketName, ownerID, ownerIDChild, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.main", true),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "bucket", testBucketName),
				),
			},
			{
				ResourceName:      "scaleway_object_bucket_acl.main",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccScalewayObjectBucketACL_GranteeWithOwner(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	testBucketName := sdkacctest.RandomWithPrefix("test-acc-scw-object-acl-owner")
	ownerID := "105bdce1-64c0-48ab-899d-868455867ecf"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayObjectBucketDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%[1]s"
						region = "%[3]s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.id
						expected_bucket_owner = "%[2]s"
						access_control_policy {
						  grant {
							grantee {
								id   = "%[2]s"
								type = "CanonicalUser"
							}
							permission = "FULL_CONTROL"
						  }
						
						  grant {
							grantee {
							  id   = "%[2]s"
							  type = "CanonicalUser"
							}
							permission = "WRITE"
						  }
						
						  owner {
							id = "%[2]s"
						  }
						}
					}
					`, testBucketName, ownerID, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.main", true),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "bucket", testBucketName),
				),
			},
		},
	})
}

func TestAccScalewayObjectBucketACL_WithBucketName(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	testBucketName := sdkacctest.RandomWithPrefix("test-acc-scw-object-acl-name")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayObjectBucketDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%s"
						region = "%s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.name
						acl = "public-read"

					}
					`, testBucketName, objectTestsMainRegion),
				ExpectError: regexp.MustCompile("error putting Object Storage ACL: NoSuchBucket: The specified bucket does not exist"),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_object_bucket" "main" {
						name = "%s"
						region = "%s"
					}
				
					resource "scaleway_object_bucket_acl" "main" {
						bucket = scaleway_object_bucket.main.name
						acl = "public-read"
						region = "%[2]s"
					}
					`, testBucketName, objectTestsMainRegion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayObjectBucketExists(tt, "scaleway_object_bucket.main", true),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "bucket", testBucketName),
					resource.TestCheckResourceAttr("scaleway_object_bucket_acl.main", "acl", "public-read"),
				),
			},
		},
	})
}
