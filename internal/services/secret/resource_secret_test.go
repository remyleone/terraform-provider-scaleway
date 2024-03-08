package secret_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	secretSDK "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/secret"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"testing"
)

func init() {
	resource.AddTestSweepers("scaleway_secret", &resource.Sweeper{
		Name: "scaleway_secret",
		F:    testSweepSecret,
	})
}

func testSweepSecret(_ string) error {
	return tests.SweepRegions(scw.AllRegions, func(scwClient *scw.Client, region scw.Region) error {
		secretAPI := secretSDK.NewAPI(scwClient)

		logging.L.Debugf("sweeper: deleting the secrets in (%s)", region)

		listSecrets, err := secretAPI.ListSecrets(&secretSDK.ListSecretsRequest{Region: region}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing secrets in (%s) in sweeper: %s", region, err)
		}

		for _, se := range listSecrets.Secrets {
			err := secretAPI.DeleteSecret(&secretSDK.DeleteSecretRequest{
				SecretID: se.ID,
				Region:   region,
			})
			if err != nil {
				logging.L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting secret in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewaySecret_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	secretName := "secretNameBasic"
	updatedName := "secretNameBasicUpdated"
	secretDescription := "secret description"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewaySecretDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "scaleway_secret" "main" {
				  name        = "%s"
				  description = "%s"
				  tags        = ["devtools", "provider", "terraform"]
				}
				`, secretName, secretDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewaySecretExists(tt, "scaleway_secret.main"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "name", secretName),
					resource.TestCheckResourceAttr("scaleway_secret.main", "description", secretDescription),
					resource.TestCheckResourceAttr("scaleway_secret.main", "status", secretSDK.SecretStatusReady.String()),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.0", "devtools"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.1", "provider"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.2", "terraform"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.#", "3"),
					resource.TestCheckResourceAttrSet("scaleway_secret.main", "updated_at"),
					resource.TestCheckResourceAttrSet("scaleway_secret.main", "created_at"),
					tests.TestCheckResourceAttrUUID("scaleway_secret.main", "id"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "scaleway_secret" "main" {
				  name        = "%s"
				  description = "update description"
				  tags        = ["devtools"]
				}
				`, updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewaySecretExists(tt, "scaleway_secret.main"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "name", updatedName),
					resource.TestCheckResourceAttr("scaleway_secret.main", "description", "update description"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.0", "devtools"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.#", "1"),
					tests.TestCheckResourceAttrUUID("scaleway_secret.main", "id"),
				),
			},
			{
				Config: fmt.Sprintf(`
				resource "scaleway_secret" "main" {
				  name        = "%s"
				}
				`, secretName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewaySecretExists(tt, "scaleway_secret.main"),
					resource.TestCheckResourceAttr("scaleway_secret.main", "name", secretName),
					resource.TestCheckResourceAttr("scaleway_secret.main", "description", ""),
					resource.TestCheckResourceAttr("scaleway_secret.main", "tags.#", "0"),
					tests.TestCheckResourceAttrUUID("scaleway_secret.main", "id"),
				),
			},
		},
	})
}

func testAccCheckScalewaySecretExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := secret.SecretAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetSecret(&secretSDK.GetSecretRequest{
			SecretID: id,
			Region:   region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewaySecretDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_secret" {
				continue
			}

			api, region, id, err := secret.SecretAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.GetSecret(&secretSDK.GetSecretRequest{
				SecretID: id,
				Region:   region,
			})

			if err == nil {
				return fmt.Errorf("secret (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
