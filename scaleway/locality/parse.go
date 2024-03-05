package locality

import (
	"fmt"
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
