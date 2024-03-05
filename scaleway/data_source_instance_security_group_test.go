package scaleway

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceInstanceSecurityGroup_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	securityGroupName := "tf-security-group"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceSecurityGroupDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_instance_security_group" "main" {
						name = "%s"
					}`, securityGroupName),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_instance_security_group" "main" {
						name = "%s"
					}
					
					data "scaleway_instance_security_group" "prod" {
						name = "${scaleway_instance_security_group.main.name}"
					}
					
					data "scaleway_instance_security_group" "stg" {
						security_group_id = "${scaleway_instance_security_group.main.id}"
					}`, securityGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupExists(tt, "data.scaleway_instance_security_group.prod"),
					resource.TestCheckResourceAttr("data.scaleway_instance_security_group.prod", "name", securityGroupName),
					testAccCheckScalewayInstanceSecurityGroupExists(tt, "data.scaleway_instance_security_group.stg"),
					resource.TestCheckResourceAttr("data.scaleway_instance_security_group.stg", "name", securityGroupName),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_instance_security_group" "main" {
						name = "%s"
					}
					
					data "scaleway_instance_security_group" "prod" {
						security_group_id = "${scaleway_instance_security_group.main.id}"
					}
					
					data "scaleway_instance_security_group" "stg" {
						name = "${scaleway_instance_security_group.main.name}"
					}`, securityGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSecurityGroupExists(tt, "data.scaleway_instance_security_group.prod"),
					resource.TestCheckResourceAttr("data.scaleway_instance_security_group.prod", "name", securityGroupName),
					testAccCheckScalewayInstanceSecurityGroupExists(tt, "data.scaleway_instance_security_group.stg"),
					resource.TestCheckResourceAttr("data.scaleway_instance_security_group.stg", "name", securityGroupName),
				),
			},
		},
	})
}
