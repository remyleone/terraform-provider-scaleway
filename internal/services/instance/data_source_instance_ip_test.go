package instance

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceInstanceIP_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `resource "scaleway_instance_ip" "ip" {}`,
			},
			{
				Config: `
					resource "scaleway_instance_ip" "ip" {}

					data "scaleway_instance_ip" "ip-from-address" {
						address = "${scaleway_instance_ip.ip.address}"
					}

					data "scaleway_instance_ip" "ip-from-id" {
						id = "${scaleway_instance_ip.ip.id}"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceAttrIP("scaleway_instance_ip.ip", "address"),
					testCheckResourceAttrIP("data.scaleway_instance_ip.ip-from-address", "address"),
					testCheckResourceAttrIP("data.scaleway_instance_ip.ip-from-id", "address"),
					resource.TestCheckResourceAttrPair("scaleway_instance_ip.ip", "address", "data.scaleway_instance_ip.ip-from-address", "address"),
					resource.TestCheckResourceAttrPair("scaleway_instance_ip.ip", "address", "data.scaleway_instance_ip.ip-from-id", "address"),
				),
			},
		},
	})
}
