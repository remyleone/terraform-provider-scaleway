package scaleway

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceIamApplication_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayIamApplicationDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_iam_application" "app_ds_basic" {
					  name = "tf_tests_data_source_basic"
					}
				`,
			},
			{
				Config: `
					resource "scaleway_iam_application" "app_ds_basic" {
					  name = "tf_tests_data_source_basic"
					}
					
					data "scaleway_iam_application" "find_by_id_basic" {
					  application_id = scaleway_iam_application.app_ds_basic.id
					}
					data "scaleway_iam_application" "find_by_name_basic" {
					  name = "tf_tests_data_source_basic"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamApplicationExists(tt, "scaleway_iam_application.app_ds_basic"),
					resource.TestCheckResourceAttr("data.scaleway_iam_application.find_by_id_basic", "name", "tf_tests_data_source_basic"),
					resource.TestCheckResourceAttr("data.scaleway_iam_application.find_by_name_basic", "name", "tf_tests_data_source_basic"),
					resource.TestCheckResourceAttrPair("data.scaleway_iam_application.find_by_id_basic", "id", "scaleway_iam_application.app_ds_basic", "id"),
					resource.TestCheckResourceAttrPair("data.scaleway_iam_application.find_by_name_basic", "id", "scaleway_iam_application.app_ds_basic", "id"),
				),
			},
			{
				Config: `
					resource "scaleway_iam_application" "app_ds_basic" {
						name        = "tf_tests_data_source_basic_renamed"
						description = "tf_tests_data_source_basic_description"
					}
				`,
			},
			{
				Config: `
					resource "scaleway_iam_application" "app_ds_basic" {
						name        = "tf_tests_data_source_basic_renamed"
						description = "tf_tests_data_source_basic_description"
					}
			
					data "scaleway_iam_application" "find_by_id_basic" {
						application_id 	= scaleway_iam_application.app_ds_basic.id
					}
					data "scaleway_iam_application" "find_by_name_basic" {
						name        	= "tf_tests_data_source_basic_renamed"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamApplicationExists(tt, "scaleway_iam_application.app_ds_basic"),
					resource.TestCheckResourceAttr("data.scaleway_iam_application.find_by_id_basic", "name", "tf_tests_data_source_basic_renamed"),
					resource.TestCheckResourceAttr("data.scaleway_iam_application.find_by_name_basic", "name", "tf_tests_data_source_basic_renamed"),
					resource.TestCheckResourceAttr("data.scaleway_iam_application.find_by_id_basic", "description", "tf_tests_data_source_basic_description"),
					resource.TestCheckResourceAttr("data.scaleway_iam_application.find_by_name_basic", "description", "tf_tests_data_source_basic_description"),
					resource.TestCheckResourceAttrPair("data.scaleway_iam_application.find_by_id_basic", "id", "scaleway_iam_application.app_ds_basic", "id"),
					resource.TestCheckResourceAttrPair("data.scaleway_iam_application.find_by_name_basic", "id", "scaleway_iam_application.app_ds_basic", "id"),
				),
			},
		},
	})
}
