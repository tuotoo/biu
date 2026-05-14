package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/registration"
)

// CertUser implements the registration.User interface for ACME.
type CertUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *CertUser) GetEmail() string                        { return u.Email }
func (u *CertUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *CertUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

type userMeta struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
}

// LoadOrCreateUser loads an existing ACME account or creates a new one.
func LoadOrCreateUser(cacheDir, email string) (*CertUser, error) {
	keyPath := filepath.Join(cacheDir, "account.key")
	metaPath := filepath.Join(cacheDir, "account.json")

	// Try to load existing account
	keyPEM, err := os.ReadFile(keyPath)
	if err == nil {
		metaData, err := os.ReadFile(metaPath)
		if err == nil {
			block, _ := pem.Decode(keyPEM)
			if block != nil {
				key, err := x509.ParseECPrivateKey(block.Bytes)
				if err == nil {
					var meta userMeta
					if json.Unmarshal(metaData, &meta) == nil {
						if email != "" && email != meta.Email {
							meta.Email = email
						}
						return &CertUser{
							Email:        meta.Email,
							Registration: meta.Registration,
							key:          key,
						}, nil
					}
				}
			}
		}
	}

	// Create new account
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	user := &CertUser{
		Email: email,
		key:   key,
	}

	if err := SaveUser(cacheDir, user); err != nil {
		return nil, err
	}

	return user, nil
}

// SaveUser persists the ACME user account.
func SaveUser(cacheDir string, user *CertUser) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	keyPath := filepath.Join(cacheDir, "account.key")
	metaPath := filepath.Join(cacheDir, "account.json")

	ecKey, ok := user.key.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("unsupported key type: %T", user.key)
	}

	keyBytes, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	meta := userMeta{
		Email:        user.Email,
		Registration: user.Registration,
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	return nil
}
