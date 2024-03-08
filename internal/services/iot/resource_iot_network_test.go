package iot_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	iotSDK "github.com/scaleway/scaleway-sdk-go/api/iot/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iot"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"testing"
)

func TestAccScalewayIotNetwork_Minimal(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		// Destruction is done via the hub destruction.
		CheckDestroy: testAccCheckScalewayIotHubDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_iot_hub" "minimal" {
							name         = "minimal"
							product_plan = "plan_shared"
						}

						resource "scaleway_iot_network" "default" {
							name   = "default"
							hub_id = scaleway_iot_hub.minimal.id
							type   = "rest"
						}
						`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIotHubExists(tt, "scaleway_iot_hub.minimal"),
					testAccCheckScalewayIotNetworkExists(tt, "scaleway_iot_network.default"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "id"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "hub_id"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "name", "default"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "type", "rest"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "endpoint"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "secret"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "created_at"),
				),
			},
		},
	})
}

func TestAccScalewayIotNetwork_RESTWithTopicPrefix(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		// Destruction is done via the hub destruction.
		CheckDestroy: testAccCheckScalewayIotHubDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_iot_hub" "minimal" {
							name         = "minimal"
							product_plan = "plan_shared"
						}

						resource "scaleway_iot_network" "default" {
							name         = "default"
							hub_id       = scaleway_iot_hub.minimal.id
							type         = "rest"
							topic_prefix = "foo/bar"
						}
						`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIotHubExists(tt, "scaleway_iot_hub.minimal"),
					testAccCheckScalewayIotNetworkExists(tt, "scaleway_iot_network.default"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "id"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "hub_id"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "name", "default"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "type", "rest"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "topic_prefix", "foo/bar"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "endpoint"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "secret"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "created_at"),
				),
			},
		},
	})
}

func TestAccScalewayIotNetwork_Sigfox(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		// Destruction is done via the hub destruction.
		CheckDestroy: testAccCheckScalewayIotHubDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_iot_hub" "minimal" {
							name         = "minimal"
							product_plan = "plan_shared"
						}

						resource "scaleway_iot_network" "default" {
							name   = "default"
							hub_id = scaleway_iot_hub.minimal.id
							type   = "sigfox"
						}
						`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIotHubExists(tt, "scaleway_iot_hub.minimal"),
					testAccCheckScalewayIotNetworkExists(tt, "scaleway_iot_network.default"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "id"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "hub_id"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "name", "default"),
					resource.TestCheckResourceAttr("scaleway_iot_network.default", "type", "sigfox"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "endpoint"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "secret"),
					resource.TestCheckResourceAttrSet("scaleway_iot_network.default", "created_at"),
				),
			},
		},
	})
}

func testAccCheckScalewayIotNetworkExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		iotAPI, region, networkID, err := iot.NewAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = iotAPI.GetNetwork(&iotSDK.GetNetworkRequest{
			Region:    region,
			NetworkID: networkID,
		})
		if err != nil {
			return err
		}

		return nil
	}
}
