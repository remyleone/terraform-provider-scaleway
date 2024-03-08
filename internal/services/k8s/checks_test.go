package k8s_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	vpcSDK "github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpc"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

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
