package internal_test

import (
	"fmt"
	"path"
	"testing"
	"time"

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

func TestDBEntry_GenerateOTP(t *testing.T) {
	t.Parallel()

	for _, algo := range []string{"SHA1", "SHA256", "SHA512", "MD5"} {
		for _, digits := range []uint8{6, 8} {
			t.Run(fmt.Sprintf("totp - %s - %d digits", algo, digits), func(t *testing.T) {
				t.Parallel()

				e := internal.DBEntry{
					Type:   "totp",
					Name:   "Test",
					Issuer: "Deno",
					Info: internal.DBEntryInfo{
						Secret: "4SJHB4GSD43FZBAI7C2HLRJGPQ",
						Algo:   algo,
						Digits: digits,
						Period: 30,
					},
				}

				otp, remaining, err := e.GenerateOTP(time.Now())
				require.NoError(t, err)
				assert.Len(t, otp, int(e.Info.Digits))
				assert.Greater(t, remaining, int64(0))
			})
		}
	}
}
