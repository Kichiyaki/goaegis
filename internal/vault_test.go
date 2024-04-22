package internal_test

import (
	"path"
	"testing"

	"gitea.dwysokinski.me/Kichiyaki/goaegis/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var vaultTestPassword = []byte("test")

func TestNewVaultFromFile(t *testing.T) {
	t.Parallel()

	vault, err := internal.NewVaultFromFile(path.Join("testdata", "encrypted.json"))
	require.NoError(t, err)
	assert.NotEmpty(t, vault)
}

func TestVault_DecryptDB(t *testing.T) {
	t.Parallel()

	vault, err := internal.NewVaultFromFile(path.Join("testdata", "encrypted.json"))
	require.NoError(t, err)

	db, err := vault.DecryptDB(vaultTestPassword)
	require.NoError(t, err)
	assert.NotEmpty(t, db.Version)
	assert.NotEmpty(t, db.Entries)
}
