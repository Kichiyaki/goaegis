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

type DB struct {
	Version int       `json:"version"`
	Entries []DBEntry `json:"entries"`
}
