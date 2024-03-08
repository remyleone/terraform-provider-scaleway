package documentdb_test

import (
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/documentdb"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ResourceScalewayDocumentDBDatabaseName(t *testing.T) {
	localizedInstanceID, databaseName, err := documentdb.ResourceScalewayDocumentDBDatabaseName("fr-par/uuid/name")
	require.NoError(t, err)
	assert.Equal(t, "fr-par/uuid", localizedInstanceID)
	assert.Equal(t, "name", databaseName)

	_, _, err = documentdb.ResourceScalewayDocumentDBDatabaseName("fr-par/uuid")
	require.Error(t, err)
}
