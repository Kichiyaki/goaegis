package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/scrypt"
)

type VaultHeaderSlotKeyParams struct {
	Nonce string `json:"nonce"`
	Tag   string `json:"tag"`
}

type VaultHeaderSlot struct {
	Type      int                      `json:"type"`
	UUID      string                   `json:"uuid"`
	Key       string                   `json:"key"`
	KeyParams VaultHeaderSlotKeyParams `json:"key_params"`
	N         int                      `json:"n"`
	R         int                      `json:"r"`
	P         int                      `json:"p"`
	Salt      string                   `json:"salt"`
	Repaired  bool                     `json:"repaired"`
	IsBackup  bool                     `json:"is_backup"`
}

type VaultHeaderParams struct {
	Nonce string `json:"nonce"`
	Tag   string `json:"tag"`
}

type VaultHeader struct {
	Slots  []VaultHeaderSlot `json:"slots"`
	Params VaultHeaderParams `json:"params"`
}

type Vault struct {
	Version int         `json:"version"`
	Header  VaultHeader `json:"header"`
	DB      string      `json:"db"`
}

func NewVault(r io.Reader) (Vault, error) {
	var v Vault
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return Vault{}, fmt.Errorf("couldn't decode vault: %w", err)
	}
	return v, nil
}

func NewVaultFromFile(path string) (Vault, error) {
	f, err := os.Open(path)
	if err != nil {
		return Vault{}, err
	}
	defer func() {
		_ = f.Close()
	}()

	return NewVault(f)
}

func (v Vault) DecryptDB(password []byte) (DB, error) {
	masterKey, err := v.findMasterKey(password)
	if err != nil {
		return DB{}, err
	}

	encryptedDB, err := base64.StdEncoding.DecodeString(v.DB)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't decode db: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't construct cipher.Block from masterKey: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return DB{}, err
	}

	nonce, _ := hex.DecodeString(v.Header.Params.Nonce)
	tag, _ := hex.DecodeString(v.Header.Params.Tag)

	data, err := aesGCM.Open(nil, nonce, append(encryptedDB, tag...), nil)
	if err != nil {
		return DB{}, err
	}

	var db DB

	if err = json.Unmarshal(data, &db); err != nil {
		return DB{}, fmt.Errorf("couldn't unmarshal db: %w", err)
	}

	return db, nil
}

func (v Vault) findMasterKey(password []byte) ([]byte, error) {
	var masterKey []byte

	for _, slot := range v.Header.Slots {
		if slot.Type != 1 {
			continue
		}

		salt, _ := hex.DecodeString(slot.Salt)
		key, err := scrypt.Key(password, salt, slot.N, slot.R, slot.P, 32)
		if err != nil {
			return nil, fmt.Errorf("couldn't derive scrypt key: %w", err)
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}

		aesGCM, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		encryptedMasterKey, _ := hex.DecodeString(slot.Key)
		nonce, _ := hex.DecodeString(slot.KeyParams.Nonce)
		tag, _ := hex.DecodeString(slot.KeyParams.Tag)

		masterKey, err = aesGCM.Open(nil, nonce, append(encryptedMasterKey, tag...), nil)
		if err != nil {
			continue
		}

		return masterKey, nil
	}

	return nil, errors.New("couldn't decrypt vault using provided password")
}

type DBEntryInfo struct {
	Secret string `json:"secret"`
	Algo   string `json:"algo"`
	Digits uint8  `json:"digits"`
	Period int    `json:"period"`
}

type DBEntry struct {
	Type   string      `json:"type"`
	Name   string      `json:"name"`
	Issuer string      `json:"issuer"`
	Group  string      `json:"group"`
	Info   DBEntryInfo `json:"info"`
}

const (
	typeTOTP        = "TOTP"
	algorithmSHA1   = "SHA1"
	algorithmSHA256 = "SHA256"
	algorithmSHA512 = "SHA512"
	algorithmMD5    = "MD5"
	digitsSix       = 6
	digitsEight     = 8
)

var ErrUnsupportedEntryType = errors.New("unsupported entry type")

func (e DBEntry) GenerateOTP(t time.Time) (string, int64, error) {
	algorithm, err := parseAlgorithm(e.Info.Algo)
	if err != nil {
		return "", 0, err
	}

	digits, err := parseDigits(e.Info.Digits)
	if err != nil {
		return "", 0, err
	}

	switch strings.ToUpper(e.Type) {
	case typeTOTP:
		code, totpErr := totp.GenerateCodeCustom(e.Info.Secret, t, totp.ValidateOpts{
			Algorithm: algorithm,
			Period:    uint(e.Info.Period),
			Digits:    digits,
		})
		if totpErr != nil {
			return "", 0, fmt.Errorf("couldn't generate totp: %w", totpErr)
		}
		period := int64(e.Info.Period)
		return code, period - (t.Unix() % period), nil
	default:
		return "", 0, fmt.Errorf("%w: %s", ErrUnsupportedEntryType, e.Type)
	}
}

type DB struct {
	Version int       `json:"version"`
	Entries []DBEntry `json:"entries"`
}

func parseAlgorithm(algorithm string) (otp.Algorithm, error) {
	switch strings.ToUpper(algorithm) {
	case algorithmSHA1:
		return otp.AlgorithmSHA1, nil
	case algorithmSHA256:
		return otp.AlgorithmSHA256, nil
	case algorithmSHA512:
		return otp.AlgorithmSHA512, nil
	case algorithmMD5:
		return otp.AlgorithmMD5, nil
	default:
		return 0, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

func parseDigits(digits uint8) (otp.Digits, error) {
	switch digits {
	case digitsSix:
		return otp.DigitsSix, nil
	case digitsEight:
		return otp.DigitsEight, nil
	default:
		return 0, fmt.Errorf("unsupported digits: %d", digits)
	}
}
