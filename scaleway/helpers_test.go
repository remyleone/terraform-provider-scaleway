package scaleway

import (
	"fmt"
	locality2 "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/regional"
	"net"
	"regexp"
	"strings"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLocalizedID(t *testing.T) {
	testCases := []struct {
		name       string
		localityID string
		id         string
		locality   string
		err        string
	}{
		{
			name:       "simple",
			localityID: "fr-par-1/my-id",
			id:         "my-id",
			locality:   "fr-par-1",
		},
		{
			name:       "id with a region",
			localityID: "fr-par/my-id",
			id:         "my-id",
			locality:   "fr-par",
		},
		{
			name:       "empty",
			localityID: "",
			err:        "cant parse localized id: ",
		},
		{
			name:       "without locality",
			localityID: "my-id",
			err:        "cant parse localized id: my-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			locality, id, err := locality2.ParseLocalizedID(tc.localityID)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.locality, locality)
				assert.Equal(t, tc.id, id)
			}
		})
	}
}

func TestParseLocalizedNestedID(t *testing.T) {
	testCases := []struct {
		name       string
		localityID string
		innerID    string
		outerID    string
		locality   string
		err        string
	}{
		{
			name:       "id with a sub directory",
			localityID: "fr-par/my-id/subdir",
			innerID:    "my-id",
			outerID:    "subdir",
			locality:   "fr-par",
		},
		{
			name:       "id with multiple sub directories",
			localityID: "fr-par/my-id/subdir/foo/bar",
			innerID:    "my-id",
			outerID:    "subdir/foo/bar",
			locality:   "fr-par",
		},
		{
			name:       "simple",
			localityID: "fr-par-1/my-id",
			err:        "cant parse localized id: fr-par-1/my-id",
		},
		{
			name:       "empty",
			localityID: "",
			err:        "cant parse localized id: ",
		},
		{
			name:       "without locality",
			localityID: "my-id",
			err:        "cant parse localized id: my-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			locality, innerID, outerID, err := locality2.ParseLocalizedNestedID(tc.localityID)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.locality, locality)
				assert.Equal(t, tc.innerID, innerID)
				assert.Equal(t, tc.outerID, outerID)
			}
		})
	}
}

func TestParseZonedID(t *testing.T) {
	testCases := []struct {
		name       string
		localityID string
		id         string
		zone       scw.Zone
		err        string
	}{
		{
			name:       "simple",
			localityID: "fr-par-1/my-id",
			id:         "my-id",
			zone:       scw.ZoneFrPar1,
		},
		{
			name:       "empty",
			localityID: "",
			err:        "cant parse localized id: ",
		},
		{
			name:       "without locality",
			localityID: "my-id",
			err:        "cant parse localized id: my-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			zone, id, err := zonal.ParseZonedID(tc.localityID)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.zone, zone)
				assert.Equal(t, tc.id, id)
			}
		})
	}
}

func TestParseRegionID(t *testing.T) {
	testCases := []struct {
		name       string
		localityID string
		id         string
		region     scw.Region
		err        string
	}{
		{
			name:       "simple",
			localityID: "fr-par/my-id",
			id:         "my-id",
			region:     scw.RegionFrPar,
		},
		{
			name:       "empty",
			localityID: "",
			err:        "cant parse localized id: ",
		},
		{
			name:       "without locality",
			localityID: "my-id",
			err:        "cant parse localized id: my-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			region, id, err := regional.ParseRegionalID(tc.localityID)
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.region, region)
				assert.Equal(t, tc.id, id)
			}
		})
	}
}

func TestNewZonedId(t *testing.T) {
	assert.Equal(t, "fr-par-1/my-id", zonal.NewZonedIDString(scw.ZoneFrPar1, "my-id"))
}

func TestNewRegionalId(t *testing.T) {
	assert.Equal(t, "fr-par/my-id", newRegionalIDString(scw.RegionFrPar, "my-id"))
}

func TestGetRandomName(t *testing.T) {
	name := types.NewRandomName("test")
	assert.True(t, strings.HasPrefix(name, "tf-test-"))
}

func testCheckResourceAttrFunc(name string, key string, test func(string) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		value, ok := rs.Primary.Attributes[key]
		if !ok {
			return fmt.Errorf("key not found: %s", key)
		}
		err := test(value)
		if err != nil {
			return fmt.Errorf("test for %s %s did not pass test: %s", name, key, err)
		}
		return nil
	}
}

var UUIDRegex = regexp.MustCompile(`[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}`)

func testCheckResourceAttrUUID(name string, key string) resource.TestCheckFunc {
	return resource.TestMatchResourceAttr(name, key, UUIDRegex)
}

func testCheckResourceAttrIPv4(name string, key string) resource.TestCheckFunc {
	return testCheckResourceAttrFunc(name, key, func(value string) error {
		ip := net.ParseIP(value)
		if ip.To4() == nil {
			return fmt.Errorf("%s is not a valid IPv4", value)
		}
		return nil
	})
}

func testCheckResourceAttrIPv6(name string, key string) resource.TestCheckFunc {
	return testCheckResourceAttrFunc(name, key, func(value string) error {
		ip := net.ParseIP(value)
		if ip.To16() == nil {
			return fmt.Errorf("%s is not a valid IPv6", value)
		}
		return nil
	})
}

func testCheckResourceAttrIP(name string, key string) resource.TestCheckFunc {
	return testCheckResourceAttrFunc(name, key, func(value string) error {
		ip := net.ParseIP(value)
		if ip == nil {
			return fmt.Errorf("%s is not a valid IP", value)
		}
		return nil
	})
}

func TestStringHashcode(t *testing.T) {
	v := "hello, world"
	expected := StringHashcode(v)
	for i := 0; i < 100; i++ {
		actual := StringHashcode(v)
		if actual != expected {
			t.Fatalf("bad: %#v\n\t%#v", actual, expected)
		}
	}
}

func TestStringHashcode_positiveIndex(t *testing.T) {
	// "2338615298" hashes to uint32(2147483648) which is math.MinInt32
	ips := []string{"192.168.1.3", "192.168.1.5", "2338615298"}
	for _, ip := range ips {
		if index := StringHashcode(ip); index < 0 {
			t.Fatalf("Bad Index %#v for ip %s", index, ip)
		}
	}
}
