package vpcgw_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	instanceSDK "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	ipamSDK "github.com/scaleway/scaleway-sdk-go/api/ipam/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/ipam"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func testAccCheckScalewayInstanceIPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "scaleway_instance_ip" {
				continue
			}

			instanceAPI, zone, id, err := instance.InstanceAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, errIP := instanceAPI.GetIP(&instanceSDK.GetIPRequest{
				Zone: zone,
				IP:   id,
			})

			// If no error resource still exist
			if errIP == nil {
				return fmt.Errorf("resource %s(%s) still exist", rs.Type, rs.Primary.ID)
			}

			// Unexpected api error we return it
			// We check for 403 because instance API return 403 for deleted IP
			if !errs.Is404Error(errIP) && !errs.Is403Error(errIP) {
				return errIP
			}
		}

		return nil
	}
}

func CheckIPAMIPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_ipam_ip" {
				continue
			}

			ipamAPI, region, ID, err := ipam.IpamAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = ipamAPI.GetIP(&ipamSDK.GetIPRequest{
				IPID:   ID,
				Region: region,
			})

			if err == nil {
				return fmt.Errorf("IP (%s) still exists", rs.Primary.ID)
			}

			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
