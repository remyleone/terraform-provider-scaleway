package marketplace

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceMarketplaceImage_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "scaleway_marketplace_image" "test1" {
						label = "ubuntu_focal"
					}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceImageExists(tt, "data.scaleway_marketplace_image.test1"),
					resource.TestCheckResourceAttr("data.scaleway_marketplace_image.test1", "label", "ubuntu_focal"),
				),
			},
		},
	})
}
