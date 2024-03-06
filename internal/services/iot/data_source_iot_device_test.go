package iot

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceIotDevice_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayIotHubDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_iot_device" "test" {
						name = "test_iot_device_datasource"
						hub_id = scaleway_iot_hub.test.id
					}

					resource "scaleway_iot_hub" "test" {
						name = "test_iot_device_datasource"
						product_plan = "plan_shared"
					}

					data "scaleway_iot_device" "by_name" {
						name = scaleway_iot_device.test.name
					}

					data "scaleway_iot_device" "by_name_and_hub" {
						name = scaleway_iot_device.test.name
						hub_id = scaleway_iot_hub.test.id
					}

					data "scaleway_iot_device" "by_id" {
						device_id = scaleway_iot_device.test.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIotDeviceExists(tt, "scaleway_iot_device.test"),

					resource.TestCheckResourceAttr("data.scaleway_iot_device.by_name", "name", "test_iot_device_datasource"),
					resource.TestCheckResourceAttrSet("data.scaleway_iot_device.by_name", "id"),

					resource.TestCheckResourceAttr("data.scaleway_iot_device.by_name_and_hub", "name", "test_iot_device_datasource"),
					resource.TestCheckResourceAttrSet("data.scaleway_iot_device.by_name_and_hub", "id"),

					resource.TestCheckResourceAttr("data.scaleway_iot_device.by_id", "name", "test_iot_device_datasource"),
					resource.TestCheckResourceAttrSet("data.scaleway_iot_device.by_id", "id"),
				),
			},
		},
	})
}
