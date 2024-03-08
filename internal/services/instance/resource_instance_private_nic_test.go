package instance_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	instanceSDK "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"testing"
)

func init() {
	resource.AddTestSweepers("scaleway_instance_private_nic", &resource.Sweeper{
		Name: "scaleway_instance_private_nic",
		F:    testSweepInstanceServer,
	})
}

func TestAccScalewayInstancePrivateNIC_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstancePrivateNICDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "TestAccScalewayInstancePrivateNIC_Basic"
					}

					resource "scaleway_instance_server" "server01" {
						image = "ubuntu_focal"
						type  = "DEV1-S"
					}

					resource scaleway_instance_private_nic nic01 {
						server_id          = scaleway_instance_server.server01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICExists(tt, "scaleway_instance_private_nic.nic01"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "server_id"),
				),
			},
		},
	})
}

func TestAccScalewayInstancePrivateNIC_Tags(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstancePrivateNICDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "TestAccScalewayInstancePrivateNIC_Tags"
					}

					resource "scaleway_instance_server" "server01" {
						image = "ubuntu_jammy"
						type  = "PLAY2-PICO"
						state = "stopped"
					}

					resource scaleway_instance_private_nic nic01 {
						server_id          = scaleway_instance_server.server01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICExists(tt, "scaleway_instance_private_nic.nic01"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "server_id"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "TestAccScalewayInstancePrivateNIC_Tags"
					}

					resource "scaleway_instance_server" "server01" {
						image = "ubuntu_jammy"
						type  = "PLAY2-PICO"
						state = "stopped"
					}

					resource scaleway_instance_private_nic nic01 {
						server_id          = scaleway_instance_server.server01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
						tags = ["tag1", "tag2"]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICExists(tt, "scaleway_instance_private_nic.nic01"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "server_id"),
					resource.TestCheckResourceAttr("scaleway_instance_private_nic.nic01", "tags.0", "tag1"),
					resource.TestCheckResourceAttr("scaleway_instance_private_nic.nic01", "tags.1", "tag2"),
				),
			},
			{
				Config: `
					resource scaleway_vpc_private_network pn01 {
						name = "TestAccScalewayInstancePrivateNIC_Tags"
					}

					resource "scaleway_instance_server" "server01" {
						image = "ubuntu_jammy"
						type  = "PLAY2-PICO"
						state = "stopped"
					}

					resource scaleway_instance_private_nic nic01 {
						server_id          = scaleway_instance_server.server01.id
						private_network_id = scaleway_vpc_private_network.pn01.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePrivateNICExists(tt, "scaleway_instance_private_nic.nic01"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "mac_address"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "private_network_id"),
					resource.TestCheckResourceAttrSet("scaleway_instance_private_nic.nic01", "server_id"),
					resource.TestCheckResourceAttr("scaleway_instance_private_nic.nic01", "tags.#", "0"),
				),
			},
		},
	})
}

func testAccCheckScalewayInstancePrivateNICExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		instanceAPI, zone, innerID, outerID, err := instance.InstanceAPIWithZoneAndNestedID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = instanceAPI.GetPrivateNIC(&instanceSDK.GetPrivateNICRequest{
			ServerID:     outerID,
			PrivateNicID: innerID,
			Zone:         zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayInstancePrivateNICDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_instance_private_nic" {
				continue
			}

			instanceAPI, zone, innerID, outerID, err := instance.InstanceAPIWithZoneAndNestedID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = instanceAPI.GetPrivateNIC(&instanceSDK.GetPrivateNICRequest{
				ServerID:     outerID,
				PrivateNicID: innerID,
				Zone:         zone,
			})

			if err == nil {
				return fmt.Errorf(
					"instanceSDK private NIC %s still exists",
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
