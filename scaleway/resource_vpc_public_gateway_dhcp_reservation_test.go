package scaleway

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
)

func TestAccScalewayVPCPublicGatewayDHCPEntry_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayVPCPublicGatewayDHCPEntryDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network main {
						name = "pn_test_network"
					}

					resource "scaleway_instance_server" "main" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						zone = "fr-par-1"

						private_network {
							pn_id = scaleway_vpc_private_network.main.id
						}
					}

					resource scaleway_vpc_public_gateway_ip main {
					}

					resource scaleway_vpc_public_gateway_dhcp main {
						subnet = "192.168.1.0/24"
					}

					resource scaleway_vpc_public_gateway main {
						name = "foobar"
						type = "VPC-GW-S"
						ip_id = scaleway_vpc_public_gateway_ip.main.id
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.main.id
						private_network_id = scaleway_vpc_private_network.main.id
						dhcp_id = scaleway_vpc_public_gateway_dhcp.main.id
						cleanup_dhcp = true
						enable_masquerade = true
						depends_on = [scaleway_vpc_public_gateway_ip.main, scaleway_vpc_private_network.main]
					}

					resource scaleway_vpc_public_gateway_dhcp_reservation main {
						gateway_network_id = scaleway_vpc_gateway_network.main.id
						mac_address = scaleway_instance_server.main.private_network.0.mac_address
						ip_address = "192.168.1.1"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayDHCPReservationExists(tt, "scaleway_vpc_public_gateway_dhcp_reservation.main"),
					resource.TestCheckResourceAttrPair("scaleway_vpc_public_gateway_dhcp_reservation.main",
						"mac_address", "scaleway_instance_server.main", "private_network.0.mac_address"),
					resource.TestCheckResourceAttrPair("scaleway_vpc_public_gateway_dhcp_reservation.main", "gateway_network_id",
						"scaleway_vpc_gateway_network.main", "id"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp_reservation.main", "ip_address", "192.168.1.1"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp_reservation.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp_reservation.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp_reservation.main", "type"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network main {
						name = "pn_test_network"
					}

					resource "scaleway_instance_server" "main" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
						zone = "fr-par-1"

						private_network {
							pn_id = scaleway_vpc_private_network.main.id
						}
					}

					resource scaleway_vpc_public_gateway_ip main {
					}

					resource scaleway_vpc_public_gateway_dhcp main {
						subnet = "192.168.1.0/24"
					}

					resource scaleway_vpc_public_gateway main {
						name = "foobar"
						type = "VPC-GW-S"
						ip_id = scaleway_vpc_public_gateway_ip.main.id
					}

					resource scaleway_vpc_gateway_network main {
						gateway_id = scaleway_vpc_public_gateway.main.id
						private_network_id = scaleway_vpc_private_network.main.id
						dhcp_id = scaleway_vpc_public_gateway_dhcp.main.id
						cleanup_dhcp = true
						enable_masquerade = true
						depends_on = [scaleway_vpc_public_gateway_ip.main, scaleway_vpc_private_network.main]
					}

					resource scaleway_vpc_public_gateway_dhcp_reservation main {
						gateway_network_id = scaleway_vpc_gateway_network.main.id
						mac_address = scaleway_instance_server.main.private_network.0.mac_address
						ip_address = "192.168.1.2"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayDHCPReservationExists(tt, "scaleway_vpc_public_gateway_dhcp_reservation.main"),
					resource.TestCheckResourceAttrPair("scaleway_vpc_public_gateway_dhcp_reservation.main",
						"mac_address", "scaleway_instance_server.main", "private_network.0.mac_address"),
					resource.TestCheckResourceAttrPair("scaleway_vpc_public_gateway_dhcp_reservation.main", "gateway_network_id",
						"scaleway_vpc_gateway_network.main", "id"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway_dhcp_reservation.main", "ip_address", "192.168.1.2"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp_reservation.main", "created_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp_reservation.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_vpc_public_gateway_dhcp_reservation.main", "type"),
				),
			},
		},
	})
}

func testAccCheckScalewayVPCPublicGatewayDHCPReservationExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		vpcgwAPI, zone, ID, err := vpcgwAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		entry, err := vpcgwAPI.GetDHCPEntry(&vpcgw.GetDHCPEntryRequest{
			DHCPEntryID: ID,
			Zone:        zone,
		})
		if err != nil {
			return err
		}

		L.Debugf("reservation: ID: (%s) exist", entry.ID)
		return nil
	}
}

func testAccCheckScalewayVPCPublicGatewayDHCPEntryDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_vpc_public_gateway_dhcp_reservation" {
				continue
			}

			vpcgwAPI, zone, ID, err := vpcgwAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = vpcgwAPI.GetDHCPEntry(&vpcgw.GetDHCPEntryRequest{
				DHCPEntryID: ID,
				Zone:        zone,
			})

			if err == nil {
				return fmt.Errorf(
					"VPC public gateway DHCP Entry config %s still exists",
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
