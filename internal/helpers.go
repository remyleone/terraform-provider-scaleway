package scaleway

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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
