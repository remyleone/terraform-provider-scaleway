package container

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceContainer_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayContainerNamespaceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_container_namespace main {
					}

					resource scaleway_container main {
						name = "test-container-data"
						namespace_id = scaleway_container_namespace.main.id
					}

					data "scaleway_container" "by_name" {
						namespace_id = scaleway_container_namespace.main.id
						name = scaleway_container.main.name
					}
					
					data "scaleway_container" "by_id" {
						namespace_id = scaleway_container_namespace.main.id
						container_id = scaleway_container.main.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayContainerExists(tt, "scaleway_container.main"),
					resource.TestCheckResourceAttr("scaleway_container.main", "name", "test-container-data"),
					resource.TestCheckResourceAttrSet("data.scaleway_container.by_name", "id"),
					resource.TestCheckResourceAttrSet("data.scaleway_container.by_name", "name"),

					resource.TestCheckResourceAttr("data.scaleway_container.by_id", "name", "test-container-data"),
					resource.TestCheckResourceAttrSet("data.scaleway_container.by_id", "id"),
				),
			},
		},
	})
}
