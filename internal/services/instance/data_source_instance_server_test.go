package instance_test

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceInstanceServer_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	serverName := "tf-server"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_instance_server" "main" {
						name 	= "%s"
						image = "ubuntu_focal"
						type  = "DEV1-S"
						state = "stopped"
						tags  = [ "terraform-test", "data_scaleway_instance_server", "basic" ]
					}`, serverName),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_instance_server" "main" {
						name 	= "%s"
						image = "ubuntu_focal"
						type  = "DEV1-S"
						state = "stopped"
						tags  = [ "terraform-test", "data_scaleway_instance_server", "basic" ]
					}
					
					data "scaleway_instance_server" "prod" {
						name = "${scaleway_instance_server.main.name}"
					}
					
					data "scaleway_instance_server" "stg" {
						server_id = "${scaleway_instance_server.main.id}"
					}`, serverName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceServerExists(tt, "data.scaleway_instance_server.prod"),
					resource.TestCheckResourceAttr("data.scaleway_instance_server.prod", "name", serverName),
					testAccCheckScalewayInstanceServerExists(tt, "data.scaleway_instance_server.stg"),
					resource.TestCheckResourceAttr("data.scaleway_instance_server.stg", "name", serverName),
				),
			},
		},
	})
}
