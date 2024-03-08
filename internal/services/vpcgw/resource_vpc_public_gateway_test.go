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
	resource.AddTestSweepers("scaleway_vpc_public_gateway", &resource.Sweeper{
		Name: "scaleway_vpc_public_gateway",
		F:    testSweepVPCPublicGateway,
	})
}

func testSweepVPCPublicGateway(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		vpcgwAPI := vpcgwSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the public gateways in (%+v)", zone)

		listGatewayResponse, err := vpcgwAPI.ListGateways(&vpcgwSDK.ListGatewaysRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing public gateway in sweeper: %w", err)
		}

		for _, gateway := range listGatewayResponse.Gateways {
			err := vpcgwAPI.DeleteGateway(&vpcgwSDK.DeleteGatewayRequest{
				Zone:      zone,
				GatewayID: gateway.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting public gateway in sweeper: %w", err)
			}
		}
		return nil
	})
}

func TestAccScalewayVPCPublicGateway_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	publicGatewayName := "public-gateway-test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      checks.TestAccCheckScalewayVPCPublicGatewayDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_public_gateway main {
						name = "%s"
						type = "VPC-GW-S"
					}
				`, publicGatewayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "name", publicGatewayName),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "type", "VPC-GW-S"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "status", vpcgwSDK.GatewayStatusRunning.String()),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_public_gateway main {
						name = "%s-new"
						type = "VPC-GW-S"
						tags = ["tag0", "tag1"]
						upstream_dns_servers = [ "1.2.3.4", "4.3.2.1" ]
					}
				`, publicGatewayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "name", publicGatewayName+"-new"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "type", "VPC-GW-S"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "status", vpcgwSDK.GatewayStatusRunning.String()),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "tags.0", "tag0"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "tags.1", "tag1"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "upstream_dns_servers.0", "1.2.3.4"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "upstream_dns_servers.1", "4.3.2.1"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_public_gateway main {
						name = "%s-zone"
						type = "VPC-GW-S"
						zone = "nl-ams-1"
					}
				`, publicGatewayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "name", publicGatewayName+"-zone"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "type", "VPC-GW-S"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "status", vpcgwSDK.GatewayStatusRunning.String()),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "zone", "nl-ams-1"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_public_gateway main {
						name = "%s-zone-to-update"
						type = "VPC-GW-S"
						zone = "nl-ams-1"
					}
				`, publicGatewayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "name", publicGatewayName+"-zone-to-update"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "type", "VPC-GW-S"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "status", vpcgwSDK.GatewayStatusRunning.String()),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "zone", "nl-ams-1"),
				),
			},
		},
	})
}

func TestAccScalewayVPCPublicGateway_Bastion(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	publicGatewayName := "public-gateway-bastion-test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      checks.TestAccCheckScalewayVPCPublicGatewayDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_public_gateway main {
						name = "%s"
						type = "VPC-GW-S"
						bastion_enabled = true
					}
				`, publicGatewayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(
						tt,
						"scaleway_vpc_public_gateway.main",
					),
					resource.TestCheckResourceAttr(
						"scaleway_vpc_public_gateway.main",
						"name",
						publicGatewayName,
					),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "bastion_enabled", "true"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_vpc_public_gateway main {
						name = "%s"
						type = "VPC-GW-S"
						bastion_enabled = false
					}
				`, publicGatewayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "name", publicGatewayName),
					resource.TestCheckResourceAttr("scaleway_vpc_public_gateway.main", "bastion_enabled", "false"),
				),
			},
		},
	})
}

func TestAccScalewayVPCPublicGateway_AttachToIP(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckScalewayVPCPublicGatewayIPDestroy(tt),
			checks.TestAccCheckScalewayVPCPublicGatewayDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_public_gateway_ip main {
					}

					resource scaleway_vpc_public_gateway main {
						name = "foobar"
						type = "VPC-GW-S"
						ip_id = scaleway_vpc_public_gateway_ip.main.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayIPExists(tt, "scaleway_vpc_public_gateway_ip.main"),
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttrPair(
						"scaleway_vpc_public_gateway.main", "ip_id",
						"scaleway_vpc_public_gateway_ip.main", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewayVPCPublicGatewayExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		vpcgwAPI, zone, ID, err := vpcgw.VpcgwAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = vpcgwAPI.GetGateway(&vpcgwSDK.GetGatewayRequest{
			GatewayID: ID,
			Zone:      zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}
