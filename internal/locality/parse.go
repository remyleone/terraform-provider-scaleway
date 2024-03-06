package locality

import (
	"fmt"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"strings"
)

// ParseLocalizedID parses a localizedID and extracts the resource locality and id.
func ParseLocalizedID(localizedID string) (locality, id string, err error) {
	tab := strings.Split(localizedID, "/")
	if len(tab) != 2 {
		return "", localizedID, fmt.Errorf("cant parse localized id: %s", localizedID)
	}
	return tab[0], tab[1], nil
}

// ParseLocalizedNestedID parses a localizedNestedID and extracts the resource locality, the inner and outer id.
func ParseLocalizedNestedID(localizedID string) (locality string, innerID, outerID string, err error) {
	tab := strings.Split(localizedID, "/")
	if len(tab) < 3 {
		return "", "", localizedID, fmt.Errorf("cant parse localized id: %s", localizedID)
	}
	return tab[0], tab[1], strings.Join(tab[2:], "/"), nil
}

// ParseLocalizedNestedID parses a localizedNestedOwnerID and extracts the resource locality, the inner and outer id and owner.
func ParseLocalizedNestedOwnerID(localizedID string) (locality string, innerID, outerID string, err error) {
	tab := strings.Split(localizedID, "/")
	n := len(tab)
	switch n {
	case 2:
		locality = tab[0]
		innerID = tab[1]
	case 3:
		locality, innerID, outerID, err = ParseLocalizedNestedID(localizedID)
	default:
		err = fmt.Errorf("cant parse localized id: %s", localizedID)
	}

	if err != nil {
		return "", "", localizedID, err
	}

	return locality, innerID, outerID, nil
}

// ParseRegionalID parses a regionalID and extracts the resource region and id.
func ParseRegionalID(regionalID string) (region scw.Region, id string, err error) {
	locality, id, err := ParseLocalizedID(regionalID)
	if err != nil {
		return
	}

	region, err = scw.ParseRegion(locality)
	return
}

// ParseRegionalNestedID parses a regionalNestedID and extracts the resource region, inner and outer ID.
func ParseRegionalNestedID(regionalNestedID string) (region scw.Region, outerID, innerID string, err error) {
	locality, innerID, outerID, err := ParseLocalizedNestedID(regionalNestedID)
	if err != nil {
		return
	}

	region, err = scw.ParseRegion(locality)
	return
}

// ParseZonedNestedID parses a zonedNestedID and extracts the resource zone ,inner and outer ID.
func ParseZonedNestedID(zonedNestedID string) (zone scw.Zone, outerID, innerID string, err error) {
	locality, innerID, outerID, err := ParseLocalizedNestedID(zonedNestedID)
	if err != nil {
		return
	}

	zone, err = scw.ParseZone(locality)
	return
}
