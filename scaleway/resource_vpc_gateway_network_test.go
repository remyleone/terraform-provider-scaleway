package scaleway

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	vpcgw "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_gateway_network", &resource.Sweeper{
		Name: "scaleway_gateway_network",
		F:    testSweepVPCGatewayNetwork,
	})
}

func testSweepVPCGatewayNetwork(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		vpcgwAPI := vpcgw.NewAPI(scwClient)
		L.Debugf("sweeper: destroying the gateway network in (%s)", zone)

		listPNResponse, err := vpcgwAPI.ListGatewayNetworks(&vpcgw.ListGatewayNetworksRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing gateway network in sweeper: %s", err)
		}

		for _, gn := range listPNResponse.GatewayNetworks {
			err := vpcgwAPI.DeleteGatewayNetwork(&vpcgw.DeleteGatewayNetworkRequest{
				GatewayNetworkID: gn.GatewayID,
				Zone:             zone,
				// Cleanup the dhcp resource related. DON'T CALL THE SWEEPER DHCP
				CleanupDHCP: true,
			})
			if err != nil {
				return fmt.Errorf("error deleting gateway network in sweeper: %s", err)
			}
		}
		return nil
	})
}

func TestAccScalewayVPCGatewayNetwork_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayVPCGatewayNetworkDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "pn_test_network"
					}

					resource scaleway_vpc_public_gateway_ip gw01 {
					}

					resource scaleway_vpc_public_gateway_dhcp dhcp01 {
						subnet = "192.168.1.0/24"
					}
				`,
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "pn_test_network"
					}

					resource scaleway_vpc_public_gateway_ip gw01 {
					}

					resource scaleway_vpc_public_gateway_dhcp dhcp01 {
						subnet = "192.168.1.0/24"
					}

					resource scaleway_vpc_public_gateway pg01 {
						name = "foobar"
						type = "VPC-GW-S"
						ip_id = scaleway_vpc_public_gateway_ip.gw01.id
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.pg01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
						dhcp_id = scaleway_vpc_public_gateway_dhcp.dhcp01.id
						cleanup_dhcp = true
						enable_masquerade = true
						depends_on = [scaleway_vpc_public_gateway_ip.gw01, scaleway_vpc_private_network.pn01]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCGatewayNetworkExists(tt, "scaleway_vpc_gateway_network.main"),
					resource.TestCheckResourceAttrPair("scaleway_vpc_gateway_network.main",
						"private_network_id", "scaleway_vpc_private_network.pn01", "id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "gateway_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "dhcp_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "zone"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "enable_dhcp", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "cleanup_dhcp", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "enable_masquerade", "true"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "pn_test_network"
					}

					resource scaleway_vpc_public_gateway_ip gw01 {
					}

					resource scaleway_vpc_public_gateway pg01 {
						name = "foobar"
						type = "VPC-GW-S"
						ip_id = scaleway_vpc_public_gateway_ip.gw01.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("scaleway_vpc_private_network.pn01", "name"),
				),
			},
		},
	})
}

func TestAccScalewayVPCGatewayNetwork_WithoutDHCP(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayVPCGatewayNetworkDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "pn_test_network"
					}

					resource scaleway_vpc_public_gateway pg01 {
						name = "foobar"
						type = "VPC-GW-S"
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.pg01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
						enable_dhcp = false
						enable_masquerade = true
						static_address = "192.168.1.42/24"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCGatewayNetworkExists(tt, "scaleway_vpc_gateway_network.main"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "gateway_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "zone"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "static_address", "192.168.1.42/24"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "enable_dhcp", "false"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "cleanup_dhcp", "false"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "enable_masquerade", "true"),
				),
			},
		},
	})
}

func TestAccScalewayVPCGatewayNetwork_WithIPAMConfig(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayVPCGatewayNetworkDestroy(tt),
			testAccCheckScalewayIPAMIPDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc vpc01 {
						name = "my vpc"
					}

					resource scaleway_vpc_private_network pn01 {
						name = "pn_test_network"
						ipv4_subnet {
							subnet = "172.16.64.0/22"
						}
						vpc_id = scaleway_vpc.vpc01.id
					}

					resource scaleway_vpc_public_gateway pg01 {
						name = "foobar"
						type = "VPC-GW-S"
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.pg01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
						enable_masquerade = true
						ipam_config {
							push_default_route = true
						}					
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCGatewayNetworkExists(tt, "scaleway_vpc_gateway_network.main"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "gateway_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "status"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "zone"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "static_address"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "ipam_config.0.push_default_route", "true"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "ipam_config.0.ipam_ip_id"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "enable_dhcp", "true"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "enable_masquerade", "true"),
				),
			},
			{
				Config: `
					resource scaleway_vpc vpc01 {
						name = "my vpc"
					}

					resource scaleway_vpc_private_network pn01 {
						name = "pn_test_network"
						ipv4_subnet {
							subnet = "172.16.64.0/22"
						}
						vpc_id = scaleway_vpc.vpc01.id
					}

					resource scaleway_vpc_public_gateway pg01 {
						name = "foobar"
						type = "VPC-GW-S"
					}

					resource "scaleway_ipam_ip" "ip01" {
					  address = "172.16.64.7"
					  source {
						private_network_id = scaleway_vpc_private_network.pn01.id
					  }
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.pg01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
						enable_masquerade = true
						ipam_config {
							push_default_route = true
							ipam_ip_id = scaleway_ipam_ip.ip01.id
						}					
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCGatewayNetworkExists(tt, "scaleway_vpc_gateway_network.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_gateway_network.main", "ipam_config.0.push_default_route", "true"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_gateway_network.main", "ipam_config.0.ipam_ip_id"),
					testAccCheckScalewayResourceRawIDMatches("scaleway_vpc_gateway_network.main", "ipam_config.0.ipam_ip_id", "scaleway_ipam_ip.ip01", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewayVPCGatewayNetworkExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		vpcgwNetworkAPI, zone, ID, err := vpcgwAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = vpcgwNetworkAPI.GetGatewayNetwork(&vpcgw.GetGatewayNetworkRequest{
			GatewayNetworkID: ID,
			Zone:             zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayVPCGatewayNetworkDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_gateway_network" {
				continue
			}

			vpcgwNetworkAPI, zone, ID, err := vpcgwAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwNetworkAPI.GetGatewayNetwork(&vpcgw.GetGatewayNetworkRequest{
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
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
