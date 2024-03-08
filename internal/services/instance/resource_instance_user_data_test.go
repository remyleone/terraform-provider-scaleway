package instance_test

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayInstanceServerUserData_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      CheckServerDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
				resource "scaleway_instance_user_data" "main" {
					server_id = scaleway_instance_server.main.id
				   	key = "cloud-init"
					value = <<-EOF
#cloud-config
apt-update: true
apt-upgrade: true
EOF
				}

				resource "scaleway_instance_server" "main" {
					image = "ubuntu_focal"
					type  = "DEV1-S"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_instance_user_data.main", "key", "cloud-init"),
				),
			},
		},
	})
}
