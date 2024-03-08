package lb_test

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	instanceSDK "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	vpcSDK "github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	vpcgwSDK "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/object"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpc"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpcgw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func CheckServerDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_instance_server" {
				continue
			}

			instanceAPI, zone, ID, err := instance.InstanceAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = instanceAPI.GetServer(&instanceSDK.GetServerRequest{
				ServerID: ID,
				Zone:     zone,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("server (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckPrivateNetworkDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_private_network" {
				continue
			}

			vpcAPI, region, ID, err := vpc.VpcAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}
			_, err = vpcAPI.GetPrivateNetwork(&vpcSDK.GetPrivateNetworkRequest{
				PrivateNetworkID: ID,
				Region:           region,
			})

			if err == nil {
				return fmt.Errorf(
					"VPC private network %s still exists",
					rs.Primary.ID,
				)
			}
			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckObjectDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway" {
				continue
			}

			regionalID := locality.ExpandRegionalID(rs.Primary.Attributes["bucket"])
			bucketRegion := regionalID.Region.String()
			bucketName := regionalID.ID
			key := rs.Primary.Attributes["key"]

			s3Client, err := object.NewS3ClientFromMeta(tt.GetMeta(), bucketRegion)
			if err != nil {
				return err
			}

			_, err = s3Client.GetObject(&s3.GetObjectInput{
				Bucket: scw.StringPtr(bucketName),
				Key:    scw.StringPtr(key),
			})
			if err != nil {
				if s3err, ok := err.(awserr.Error); ok && s3err.Code() == s3.ErrCodeNoSuchBucket {
					// bucket doesn't exist
					continue
				}
				return fmt.Errorf("couldn't get object to verify if it stil exists: %s", err)
			}

			return errors.New("object should be deleted")
		}
		return nil
	}
}

func CheckVPCGatewayNetworkDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_gateway_network" {
				continue
			}

			vpcgwNetworkAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwNetworkAPI.GetGatewayNetwork(&vpcgwSDK.GetGatewayNetworkRequest{
				GatewayNetworkID: ID,
				Zone:             zone,
			})

			if err == nil {
				return fmt.Errorf(
					"VPC gateway network %s still exists",
					rs.Primary.ID,
				)
			}

			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckObjectBucketDestroy(tt *tests.TestTools) resource.TestCheckFunc {
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

func CheckVPCPublicGatewayIPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_public_gateway_ip" {
				continue
			}

			vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwAPI.GetIP(&vpcgwSDK.GetIPRequest{
				IPID: ID,
				Zone: zone,
			})

			if err == nil {
				return fmt.Errorf(
					"VPC public gateway ip %s still exists",
					rs.Primary.ID,
				)
			}

			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckVPCPublicGatewayDHCPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_public_gateway_dhcp" {
				continue
			}

			vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwAPI.GetDHCP(&vpcgwSDK.GetDHCPRequest{
				DHCPID: ID,
				Zone:   zone,
			})

			if err == nil {
				return fmt.Errorf(
					"VPC public gateway DHCP config %s still exists",
					rs.Primary.ID,
				)
			}

			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckVPCPublicGatewayDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_public_gateway" {
				continue
			}

			vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwAPI.GetGateway(&vpcgwSDK.GetGatewayRequest{
				GatewayID: ID,
				Zone:      zone,
			})

			if err == nil {
				return fmt.Errorf(
					"VPC public gateway %s still exists",
					rs.Primary.ID,
				)
			}

			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckObjectBucketWebsiteConfigurationExists(tt *tests.TestTools, resourceName string) resource.TestCheckFunc {
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

func CheckObjectBucketWebsiteConfigurationDestroy(tt *tests.TestTools) resource.TestCheckFunc {
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
