package checks

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	vpcgw2 "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpcgw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func TestAccCheckScalewayVPCGatewayNetworkDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_gateway_network" {
				continue
			}

			vpcgwNetworkAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwNetworkAPI.GetGatewayNetwork(&vpcgw2.GetGatewayNetworkRequest{
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

func TestAccCheckScalewayVPCPublicGatewayDHCPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_public_gateway_dhcp" {
				continue
			}

			vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwAPI.GetDHCP(&vpcgw2.GetDHCPRequest{
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

func TestAccCheckScalewayVPCPublicGatewayDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_public_gateway" {
				continue
			}

			vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwAPI.GetGateway(&vpcgw2.GetGatewayRequest{
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
