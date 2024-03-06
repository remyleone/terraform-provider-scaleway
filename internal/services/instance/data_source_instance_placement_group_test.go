package instance_test

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceInstancePlacementGroup_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayInstancePlacementGroupDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_instance_placement_group main {
  						name = "test-ds-instance-placement-group-basic"
					}

					data scaleway_instance_placement_group find_by_name {
						name = scaleway_instance_placement_group.main.name
					}

					data scaleway_instance_placement_group find_by_id {
						placement_group_id = scaleway_instance_placement_group.main.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstancePlacementGroupExists(tt, "scaleway_instance_placement_group.main"),

					resource.TestCheckResourceAttrPair("scaleway_instance_placement_group.main", "name", "data.scaleway_instance_placement_group.find_by_name", "name"),
					resource.TestCheckResourceAttrPair("scaleway_instance_placement_group.main", "name", "data.scaleway_instance_placement_group.find_by_id", "name"),
					resource.TestCheckResourceAttrPair("scaleway_instance_placement_group.main", "id", "data.scaleway_instance_placement_group.find_by_name", "id"),
					resource.TestCheckResourceAttrPair("scaleway_instance_placement_group.main", "id", "data.scaleway_instance_placement_group.find_by_id", "id"),
				),
			},
		},
	})
}
