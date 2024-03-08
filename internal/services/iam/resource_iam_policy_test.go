package iam_test

import (
	"context"
	"errors"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/iam"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	iamSDK "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/require"
)

func init() {
	resource.AddTestSweepers("scaleway_iam_policy", &resource.Sweeper{
		Name: "scaleway_iam_policy",
		F:    testSweepIamPolicy,
	})
}

func testSweepIamPolicy(_ string) error {
	return tests.Sweep(func(scwClient *scw.Client) error {
		api := iamSDK.NewAPI(scwClient)

		orgID, exists := scwClient.GetDefaultOrganizationID()
		if !exists {
			return errors.New("missing organizationID")
		}

		listPols, err := api.ListPolicies(&iamSDK.ListPoliciesRequest{
			OrganizationID: orgID,
		})
		if err != nil {
			return fmt.Errorf("failed to list policies: %w", err)
		}
		for _, pol := range listPols.Policies {
			if !tests.IsTestResource(pol.Name) {
				continue
			}
			err = api.DeletePolicy(&iamSDK.DeletePolicyRequest{
				PolicyID: pol.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to delete policy: %w", err)
			}
		}
		return nil
	})
}

func TestAccScalewayIamPolicy_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ctx := context.Background()
	project, iamAPIKey, terminateFakeSideProject, err := tests.CreateFakeIAMManager(tt)
	require.NoError(t, err)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tests.FakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayIamPolicyDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_basic"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  rule {
						organization_id      = "%[1]s"
						permission_set_names = ["ContainerRegistryReadOnly"]
					  }
					  provider = side
					  tags = ["tf_tests", "tests"]
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_basic"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.1.permission_set_names.0", "ContainerRegistryReadOnly"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "tags.#", "2"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "tags.0", "tf_tests"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "tags.1", "tests"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_basic"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_basic"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("scaleway_iam_policy.main", "rule.*", map[string]string{"organization_id": project.OrganizationID}),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayIamPolicy_NoUpdate(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ctx := context.Background()
	project, iamAPIKey, terminateFakeSideProject, err := tests.CreateFakeIAMManager(tt)
	require.NoError(t, err)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tests.FakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayIamPolicyDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_noupdate"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_noupdate"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttrSet("scaleway_iam_policy.main", "rule.0.organization_id"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_noupdate"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_noupdate"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttrSet("scaleway_iam_policy.main", "rule.0.organization_id"),
				),
			},
		},
	})
}

func TestAccScalewayIamPolicy_ChangeLinkedEntity(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	ctx := context.Background()
	project, iamAPIKey, terminateFakeSideProject, err := tests.CreateFakeIAMManager(tt)
	require.NoError(t, err)
	randAppName := "tf-tests-scaleway-iamSDK-app-policy-permissions"
	randGroupName := "tf-tests-scaleway-iamSDK-group-policy-permissions"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tests.FakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayIamPolicyDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_change_linked_entity"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_change_linked_entity"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.organization_id", project.OrganizationID),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_application" "main" {
					  name        = "tf_tests_policy_change_linked_entity"
					  description = "a description"
					  provider = side
					}

					resource "scaleway_iam_policy" "main" {
					  name           = "tf_tests_policy_change_linked_entity"
					  description    = "a description"
					  application_id = scaleway_iam_application.main.id
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttrPair("scaleway_iam_policy.main", "application_id", "scaleway_iam_application.main", "id"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_change_linked_entity"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttrSet("scaleway_iam_policy.main", "rule.0.organization_id"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_application" "app01" {
					  name = "%[2]s"
					  provider = side
					}

					resource "scaleway_iam_group" "main_app" {
					  name = "%[3]s"
					  application_ids = [
						scaleway_iam_application.app01.id
					  ]
					  provider = side
					}

					resource "scaleway_iam_policy" "main" {
					  name        = "tf_tests_policy_change_linked_entity"
					  description = "a description"
					  group_id    = scaleway_iam_group.main_app.id
					  rule {
						organization_id      = "%[1]s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID, randAppName, randGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttrPair("scaleway_iam_policy.main", "group_id", "scaleway_iam_group.main_app", "id"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_change_linked_entity"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttrSet("scaleway_iam_policy.main", "rule.0.organization_id"),
				),
			},
		},
	})
}

func TestAccScalewayIamPolicy_ChangePermissions(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ctx := context.Background()
	project, iamAPIKey, terminateFakeSideProject, err := tests.CreateFakeIAMManager(tt)
	require.NoError(t, err)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tests.FakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayIamPolicyDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_changepermissions"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_changepermissions"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.organization_id", project.OrganizationID),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "1"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_changepermissions"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess", "ContainerRegistryReadOnly"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_changepermissions"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.organization_id", project.OrganizationID),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "2"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.1", "ContainerRegistryReadOnly"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_changepermissions"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["ContainerRegistryReadOnly"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_changepermissions"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.organization_id", project.OrganizationID),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "1"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "ContainerRegistryReadOnly"),
				),
			},
		},
	})
}

func TestAccScalewayIamPolicy_ProjectID(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ctx := context.Background()
	project, iamAPIKey, terminateFakeSideProject, err := tests.CreateFakeIAMManager(tt)
	require.NoError(t, err)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tests.FakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayIamPolicyDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_projectid"
					  description  = "a description"
					  no_principal = true
					  rule {
						project_ids          = ["%s"]
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_projectid"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "1"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
					resource.TestCheckResourceAttrSet("scaleway_iam_policy.main", "rule.0.project_ids.0"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_projectid"
					  description  = "a description"
					  no_principal = true
					  rule {
						project_ids          = ["%s"]
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_projectid"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "1"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
				),
			},
		},
	})
}

func TestAccScalewayIamPolicy_ChangeRulePrincipal(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	ctx := context.Background()
	project, iamAPIKey, terminateFakeSideProject, err := tests.CreateFakeIAMManager(tt)
	require.NoError(t, err)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tests.FakeSideProjectProviders(ctx, tt, project, iamAPIKey),
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			func(_ *terraform.State) error {
				return terminateFakeSideProject()
			},
			testAccCheckScalewayIamPolicyDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_changeruleprincipal"
					  description  = "a description"
					  no_principal = true
					  rule {
						organization_id      = "%s"
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_changeruleprincipal"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.organization_id", project.OrganizationID),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "1"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_iam_policy" "main" {
					  name         = "tf_tests_policy_changeruleprincipal"
					  description  = "a description"
					  no_principal = true
					  rule {
						project_ids          = ["%s"]
						permission_set_names = ["AllProductsFullAccess"]
					  }
					  provider = side
					}
					`, project.OrganizationID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayIamPolicyExists(tt, "scaleway_iam_policy.main"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "name", "tf_tests_policy_changeruleprincipal"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "description", "a description"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "no_principal", "true"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.organization_id", ""),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.#", "1"),
					resource.TestCheckResourceAttr("scaleway_iam_policy.main", "rule.0.permission_set_names.0", "AllProductsFullAccess"),
				),
			},
		},
	})
}

func testAccCheckScalewayIamPolicyExists(tt *tests.TestTools, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}

		iamAPI := iam.IAMAPI(tt.GetMeta())

		_, err := iamAPI.GetPolicy(&iamSDK.GetPolicyRequest{
			PolicyID: rs.Primary.ID,
		})
		if err != nil {
			return fmt.Errorf("could not find policy: %w", err)
		}

		return nil
	}
}

func testAccCheckScalewayIamPolicyDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "scaleway_iam_policy" {
				continue
			}

			iamAPI := iam.IAMAPI(tt.GetMeta())

			_, err := iamAPI.GetPolicy(&iamSDK.GetPolicyRequest{
				PolicyID: rs.Primary.ID,
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
