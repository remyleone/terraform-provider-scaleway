package locality

import (
	"github.com/scaleway/scaleway-sdk-go/scw"
	"strings"
)

// ExpandID returns the id whether it is a localizedID or a raw ID.
func ExpandID(id interface{}) string {
	_, ID, err := ParseLocalizedID(id.(string))
	if err != nil {
		return id.(string)
	}
	return ID
}

func ExpandZonedID(id interface{}) ZonedID {
	zonedID := ZonedID{}
	tab := strings.Split(id.(string), "/")
	if len(tab) != 2 {
		zonedID.ID = id.(string)
	} else {
		zone, _ := scw.ParseZone(tab[0])
		zonedID.ID = tab[1]
		zonedID.Zone = zone
	}

	return zonedID
}

func ExpandSliceIDsPtr(rawIDs interface{}) *[]string {
	stringSlice := make([]string, 0, len(rawIDs.([]interface{})))
	if _, ok := rawIDs.([]interface{}); !ok || rawIDs == nil {
		return &stringSlice
	}
	for _, s := range rawIDs.([]interface{}) {
		stringSlice = append(stringSlice, ExpandID(s.(string)))
	}
	return &stringSlice
}
