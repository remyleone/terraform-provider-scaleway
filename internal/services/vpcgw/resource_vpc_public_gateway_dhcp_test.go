package vpcgw_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	vpcgwSDK "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/vpcgw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests/checks"
	"testing"
)

func init() {
	resource.AddTestSweepers("scaleway_vpc_public_gateway_dhcp", &resource.Sweeper{
		Name: "scaleway_vpc_public_gateway_dhcp",
		F:    testSweepVPCPublicGatewayDHCP,
	})
}

func testSweepVPCPublicGatewayDHCP(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		api := vpcgwSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying public gateway dhcps in (%+v)", zone)

		listDHCPsResponse, err := api.ListDHCPs(&vpcgwSDK.ListDHCPsRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing public gateway dhcps in sweeper: %w", err)
		}

		for _, dhcp := range listDHCPsResponse.Dhcps {
			err := api.DeleteDHCP(&vpcgwSDK.DeleteDHCPRequest{
				Zone:   zone,
				DHCPID: dhcp.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting public gateway dhcp in sweeper: %w", err)
			}
		}

		return nil
	})
}

func TestAccScalewayVPCPublicGatewayDHCP_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      checks.TestAccCheckScalewayVPCPublicGatewayDHCPDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_public_gateway_dhcp main {
						subnet = "192.168.1.0/24"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayDHCPExists(tt, "scaleway_vpc_public_gateway_dhcp.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "subnet", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "enable_dynamic", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "valid_lifetime", "3600"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "renew_timer", "3000"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "rebind_timer", "3060"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "push_default_route", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "push_dns_server", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_server_override.#", "0"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_search.#", "0"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "dns_local_name"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "pool_low"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "pool_high"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "zone"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "organization_id"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_public_gateway_dhcp main {
						subnet = "192.168.1.0/24"
						valid_lifetime = 3000
						renew_timer = 2000
						rebind_timer = 2060
						push_default_route = false
						push_dns_server = false
						enable_dynamic = false
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayDHCPExists(tt, "scaleway_vpc_public_gateway_dhcp.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "subnet", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "push_default_route", "false"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "push_dns_server", "false"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "enable_dynamic", "false"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "valid_lifetime", "3000"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "renew_timer", "2000"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "rebind_timer", "2060"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_server_override.#", "0"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_search.#", "0"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "dns_local_name"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "pool_low"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "pool_high"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "zone"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "organization_id"),
				),
			},
			{
				Config: `
					resource "scaleway_vpc_public_gateway_dhcp" main {
					  subnet = "192.168.1.0/24"
					  push_default_route = true
					  push_dns_server = true
					  enable_dynamic = true
					  dns_servers_override = ["192.168.1.2"]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayDHCPExists(tt, "scaleway_vpc_public_gateway_dhcp.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "subnet", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "enable_dynamic", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "valid_lifetime", "3000"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "renew_timer", "2000"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "rebind_timer", "2060"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "push_default_route", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "push_dns_server", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_servers_override.#", "1"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_servers_override.0", "192.168.1.2"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "dns_local_name"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "pool_low"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "pool_high"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "zone"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp.main", "organization_id"),
				),
			},
			{
				Config: `
					resource "scaleway_vpc_public_gateway_dhcp" main {
					  subnet = "192.168.1.0/24"
					  push_default_route = true
					  push_dns_server = true
					  enable_dynamic = true
					  dns_servers_override = ["192.168.1.3"]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayDHCPExists(tt, "scaleway_vpc_public_gateway_dhcp.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "subnet", "192.168.1.0/24"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_servers_override.#", "1"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp.main", "dns_servers_override.0", "192.168.1.3"),
				),
			},
		},
	})
}

func testAccCheckScalewayVPCPublicGatewayDHCPExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = vpcgwAPI.GetDHCP(&vpcgwSDK.GetDHCPRequest{
			DHCPID: ID,
			Zone:   zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}
