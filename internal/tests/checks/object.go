package checks

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/object"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func TestAccCheckScalewayObjectBucketExists(tt *tests.TestTools, n string, shouldBeAllowed bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs := state.RootModule().Resources[n]
		if rs == nil {
			return errors.New("resource not found")
		}
		bucketName := rs.Primary.Attributes["name"]
		bucketRegion := rs.Primary.Attributes["region"]

		s3Client, err := object.NewS3ClientFromMeta(tt.GetMeta(), bucketRegion)
		if err != nil {
			return err
		}

		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return errors.New("no ID is set")
		}

		_, err = s3Client.HeadBucket(&s3.HeadBucketInput{
			Bucket: scw.StringPtr(bucketName),
		})
		if err != nil {
			if !shouldBeAllowed && object.IsS3Err(err, object.ErrCodeForbidden, object.ErrCodeForbidden) {
				return nil
			}
			if object.IsS3Err(err, s3.ErrCodeNoSuchBucket, "") {
				return errors.New("s3 bucket not found")
			}
			return err
		}
		return nil
	}
}

func TestAccCheckScalewayObjectBucketDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway" {
				continue
			}

			regionalID := locality.ExpandRegionalID(rs.Primary.ID)
			bucketRegion := regionalID.Region.String()
			bucketName := regionalID.ID

			s3Client, err := object.NewS3ClientFromMeta(tt.GetMeta(), bucketRegion)
			if err != nil {
				return err
			}

			_, err = s3Client.ListObjects(&s3.ListObjectsInput{
				Bucket: &bucketName,
			})
			if err != nil {
				if s3err, ok := err.(awserr.Error); ok && s3err.Code() == s3.ErrCodeNoSuchBucket {
					// bucket doesn't exist
					continue
				}
				return fmt.Errorf("couldn't get bucket to verify if it stil exists: %s", err)
			}

			return errors.New("bucket should be deleted")
		}
		return nil
	}
}
