package documentdb

import (
	"fmt"
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"strings"
	"time"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	telemetryDocumentDBReporting       = "telemetry_reporting"
	defaultDocumentDBInstanceTimeout   = 15 * time.Minute
	defaultWaitDocumentDBRetryInterval = 5 * time.Second
)

// DocumentDBAPIWithRegion returns a new documentdb API and the region for a Create request
func DocumentDBAPIWithRegion(d *schema.ResourceData, m interface{}) (*documentdb.API, scw.Region, error) {
	meta := m.(*meta2.Meta)
	api := documentdb.NewAPI(meta.GetScwClient())

	region, err := locality.ExtractRegion(d, meta)
	if err != nil {
		return nil, "", err
	}

	return api, region, nil
}

// documentDBAPIWithRegionalAndID returns a new documentdb API with region and ID extracted from the state
func DocumentDBAPIWithRegionAndID(m interface{}, regionalID string) (*documentdb.API, scw.Region, string, error) {
	meta := m.(*meta2.Meta)
	api := documentdb.NewAPI(meta.GetScwClient())

	region, ID, err := locality.ParseRegionalID(regionalID)
	if err != nil {
		return nil, "", "", err
	}

	return api, region, ID, nil
}

// Build the resource identifier
// The resource identifier format is "Region/InstanceId/DatabaseName"
func ResourceScalewayDocumentDBDatabaseID(region scw.Region, instanceID string, databaseName string) (resourceID string) {
	return fmt.Sprintf("%s/%s/%s", region, instanceID, databaseName)
}

// ResourceScalewayDocumentDBDatabaseName extract regional instanceID and databaseName from composed ID
// returned by ResourceScalewayDocumentDBDatabaseID()
func ResourceScalewayDocumentDBDatabaseName(id string) (string, string, error) {
	elems := strings.Split(id, "/")
	if len(elems) != 3 {
		return "", "", fmt.Errorf("cant parse terraform database id: %s", id)
	}

	return elems[0] + "/" + elems[1], elems[2], nil
}
