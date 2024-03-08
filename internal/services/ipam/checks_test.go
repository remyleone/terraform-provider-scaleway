package ipam_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	baremetalSDK "github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	ipamSDK "github.com/scaleway/scaleway-sdk-go/api/ipam/v1"
	rdbSDK "github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/ipam"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/rdb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func CheckIPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
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

func CheckRdbInstanceDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_rdb_instance" {
				continue
			}

			rdbAPI, region, ID, err := rdb.RdbAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = rdbAPI.GetInstance(&rdbSDK.GetInstanceRequest{
				InstanceID: ID,
				Region:     region,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("instance (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !errs.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckServerDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_baremetal_server" {
				continue
			}

			baremetalAPI, zonedID, err := baremetal.BaremetalAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = baremetalAPI.GetServer(&baremetalSDK.GetServerRequest{
				ServerID: zonedID.ID,
				Zone:     zonedID.Zone,
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
