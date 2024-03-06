package scaleway

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"testing"
)

// Service information constants
const (
	ServiceName = "scw"       // Name of service.
	EndpointsID = ServiceName // ID to look up a service endpoint with.
)

type ServiceErrorCheckFunc func(*testing.T) resource.ErrorCheckFunc

var serviceErrorCheckFunc map[string]ServiceErrorCheckFunc

func ErrorCheck(t *testing.T, endpointIDs ...string) resource.ErrorCheckFunc {
	t.Helper()
	return func(err error) error {
		if err == nil {
			return nil
		}

		for _, endpointID := range endpointIDs {
			if f, ok := serviceErrorCheckFunc[endpointID]; ok {
				ef := f(t)
				err = ef(err)
			}

			if err == nil {
				break
			}
		}

		return err
	}
}

// testAccCheckScalewayResourceIDPersisted checks that the ID of the resource is the same throughout tests of migration or mutation
// It can be used to check that no ForceNew has been done
func testAccCheckScalewayResourceIDPersisted(resourceName string, resourceID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource was not found: %s", resourceName)
		}
		if *resourceID != "" && *resourceID != rs.Primary.ID {
			return errors.New("resource ID changed when it should have persisted")
		}
		*resourceID = rs.Primary.ID
		return nil
	}
}

// testAccCheckScalewayResourceIDChanged checks that the ID of the resource has indeed changed, in case of ForceNew for example.
// It will fail if resourceID is empty so be sure to use testAccCheckScalewayResourceIDPersisted first in a test suite.
func testAccCheckScalewayResourceIDChanged(resourceName string, resourceID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if resourceID == nil || *resourceID == "" {
			return errors.New("resourceID was not set")
		}
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource was not found: %s", resourceName)
		}
		if *resourceID == rs.Primary.ID {
			return errors.New("resource ID persisted when it should have changed")
		}
		*resourceID = rs.Primary.ID
		return nil
	}
}

// testAccCheckScalewayResourceRawIDMatches asserts the equality of IDs from two specified attributes of two Scaleway resources.
func testAccCheckScalewayResourceRawIDMatches(res1, attr1, res2, attr2 string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs1, ok1 := s.RootModule().Resources[res1]
		if !ok1 {
			return fmt.Errorf("not found: %s", res1)
		}

		rs2, ok2 := s.RootModule().Resources[res2]
		if !ok2 {
			return fmt.Errorf("not found: %s", res2)
		}

		id1 := locality.ExpandID(rs1.Primary.Attributes[attr1])
		id2 := locality.ExpandID(rs2.Primary.Attributes[attr2])

		if id1 != id2 {
			return fmt.Errorf("ID mismatch: %s from resource %s does not match ID %s from resource %s", id1, res1, id2, res2)
		}

		return nil
	}
}
