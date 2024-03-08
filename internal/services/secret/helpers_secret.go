package secret

import (
	"encoding/base64"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"
	"time"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultSecretTimeout = 5 * time.Minute
)

// SecretAPIWithRegion returns a new Secret API and the region for a Create request
func SecretAPIWithRegion(d *schema.ResourceData, m interface{}) (*secret.API, scw.Region, error) {
	meta := m.(*meta2.Meta)
	api := secret.NewAPI(meta.GetScwClient())

	region, err := locality.ExtractRegion(d, meta)
	if err != nil {
		return nil, "", err
	}
	return api, region, nil
}

// SecretAPIWithRegionAndDefault returns a new Secret API and the region for a Create request
func SecretAPIWithRegionAndDefault(d *schema.ResourceData, m interface{}, defaultRegion scw.Region) (*secret.API, scw.Region, error) {
	meta := m.(*meta2.Meta)
	api := secret.NewAPI(meta.GetScwClient())

	region, err := locality.ExtractRegionWithDefault(d, meta, defaultRegion)
	if err != nil {
		return nil, "", err
	}
	return api, region, nil
}

// SecretAPIWithRegionAndProjectID returns a new Secret API, with region and projectID
func SecretAPIWithRegionAndProjectID(d *schema.ResourceData, m interface{}) (*secret.API, scw.Region, string, error) {
	meta := m.(*meta2.Meta)
	api := secret.NewAPI(meta.GetScwClient())

	region, err := locality.ExtractRegion(d, meta)
	if err != nil {
		return nil, "", "", err
	}

	projectID, _, err := project.ExtractProjectID(d, meta)
	if err != nil {
		return nil, "", "", err
	}

	return api, region, projectID, nil
}

// SecretAPIWithRegionAndID returns a Secret API with locality and ID extracted from the state
func SecretAPIWithRegionAndID(m interface{}, id string) (*secret.API, scw.Region, string, error) {
	meta := m.(*meta2.Meta)
	api := secret.NewAPI(meta.GetScwClient())

	region, id, err := locality.ParseRegionalID(id)
	if err != nil {
		return nil, "", "", err
	}
	return api, region, id, nil
}

// SecretVersionAPIWithRegionAndID returns a Secret API with locality and Nested ID extracted from the state
func SecretVersionAPIWithRegionAndID(m interface{}, id string) (*secret.API, scw.Region, string, string, error) {
	meta := m.(*meta2.Meta)

	region, id, revision, err := locality.ParseLocalizedNestedID(id)
	if err != nil {
		return nil, "", "", "", err
	}

	api := secret.NewAPI(meta.GetScwClient())
	return api, scw.Region(region), id, revision, nil
}

func IsBase64Encoded(data []byte) bool {
	_, err := base64.StdEncoding.DecodeString(string(data))
	return err == nil
}

func Base64Encoded(data []byte) string {
	if IsBase64Encoded(data) {
		return string(data)
	}
	return base64.StdEncoding.EncodeToString(data)
}
