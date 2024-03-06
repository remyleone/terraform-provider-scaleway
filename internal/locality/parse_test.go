package locality

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
			locality, id, err := ParseLocalizedID(tc.localityID)
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
