package account_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/account"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	accountV3 "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_account_project", &resource.Sweeper{
		Name: "scaleway_account_project",
		F:    testSweepAccountProject,
	})
}

func testSweepAccountProject(_ string) error {
	return tests.Sweep(func(scwClient *scw.Client) error {
		accountAPI := accountV3.NewProjectAPI(scwClient)

		logging.L.Debugf("sweeper: destroying the project")

		listProjects, err := accountAPI.ListProjects(&accountV3.ProjectAPIListProjectsRequest{}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}
		for _, project := range listProjects.Projects {
			if project.Name == "default" || !tests.IsTestResource(project.Name) {
				continue
			}
			err = accountAPI.DeleteProject(&accountV3.ProjectAPIDeleteProjectRequest{
				ProjectID: project.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to delete project: %w", err)
			}
		}
		return nil
	})
}

func TestAccScalewayAccountProject_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayAccountProjectDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_account_project" "main" {
							name = "tf_tests_project_basic"
							description = "a description"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayAccountProjectExists(tt, "scaleway_account_project.main"),
					resource.TestCheckResourceAttr("scaleway_account_project.main", "name", "tf_tests_project_basic"),
					resource.TestCheckResourceAttr("scaleway_account_project.main", "description", "a description"),
				),
			},
			{
				Config: `
						resource "scaleway_account_project" "main" {
							name = "tf_tests_project_basic_rename"
							description = "another description"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayAccountProjectExists(tt, "scaleway_account_project.main"),
					resource.TestCheckResourceAttr("scaleway_account_project.main", "name", "tf_tests_project_basic_rename"),
					resource.TestCheckResourceAttr("scaleway_account_project.main", "description", "another description"),
				),
			},
		},
	})
}

func TestAccScalewayAccountProject_NoUpdate(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayAccountProjectDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_account_project" "main" {
							name = "tf_tests_project_noupdate"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayAccountProjectExists(tt, "scaleway_account_project.main"),
				),
			},
			{
				Config: `
						resource "scaleway_account_project" "main" {
							name = "tf_tests_project_noupdate"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayAccountProjectExists(tt, "scaleway_account_project.main"),
				),
			},
		},
	})
}

func testAccCheckScalewayAccountProjectExists(tt *tests.TestTools, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}

		accountAPI := account.AccountV3ProjectAPI(tt.GetMeta())

		_, err := accountAPI.GetProject(&accountV3.ProjectAPIGetProjectRequest{
			ProjectID: rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("could not find project: %w", err)
		}

		return nil
	}
}

func testAccCheckScalewayAccountProjectDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "scaleway_account_project" {
				continue
			}

			accountAPI := account.AccountV3ProjectAPI(tt.GetMeta())

			_, err := accountAPI.GetProject(&accountV3.ProjectAPIGetProjectRequest{
				ProjectID: rs.Primary.ID,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("resource %s(%s) still exist", rs.Type, rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
