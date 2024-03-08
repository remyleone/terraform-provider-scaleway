package redis_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	vpcSDK "github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpc"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func CheckPrivateNetworkExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		vpcAPI, region, ID, err := vpc.VpcAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = vpcAPI.GetPrivateNetwork(&vpcSDK.GetPrivateNetworkRequest{
			PrivateNetworkID: ID,
			Region:           region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}
