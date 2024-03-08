package rdb_test

import (
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests/checks"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceRdbACL_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	instanceName := "TestAccScalewayDataSourceRDBAcl_Basic"
	latestEngineVersion := checks.TestAccCheckScalewayRdbEngineGetLatestVersion(tt, tests.PostgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      checks.TestAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_rdb_instance" "main" {
						name = "%s"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
					}

					resource "scaleway_rdb_acl" "main" {
						instance_id = scaleway_rdb_instance.main.id
						acl_rules {
							ip = "1.2.3.4/32"
							description = "foo"
						}

						acl_rules {
							ip = "4.5.6.7/32"
							description = "bar"
						}
					}
					`, instanceName, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_rdb_acl.main", "acl_rules.0.ip", "1.2.3.4/32"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_rdb_instance" "main" {
						name = "%s"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
					}

					resource "scaleway_rdb_acl" "main" {
						instance_id = scaleway_rdb_instance.main.id
						acl_rules {
							ip = "1.2.3.4/32"
							description = "foo"
						}

						acl_rules {
							ip = "4.5.6.7/32"
							description = "bar"
						}
					}

					data "scaleway_rdb_acl" "maindata" {
						instance_id = scaleway_rdb_instance.main.id
					}`, instanceName, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("scaleway_rdb_acl.main", "acl_rules.0.ip", "1.2.3.4/32"),
					resource.TestCheckResourceAttr("data.scaleway_rdb_acl.maindata", "acl_rules.0.ip", "1.2.3.4/32"),
				),
			},
		},
	})
}
