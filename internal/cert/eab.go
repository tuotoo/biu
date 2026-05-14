package cert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/tuotoo/biu/opt"
)

const zeroSSLAPI = "https://api.zerossl.com/acme/eab-credentials-email"

// registerAccount registers or looks up the ACME account. If ZeroSSL is selected
// and no EAB credentials are configured, it auto-fetches them from the ZeroSSL API.
func registerAccount(client *lego.Client, cfg certConfig, cacheDir string, logger *slog.Logger) (*registration.Resource, error) {
	kid := cfg.getEABKid()
	hmacKey := cfg.getEABHMACKey()

	// Auto-fetch EAB for ZeroSSL if not provided
	if cfg.getCA() == opt.CAZeroSSL && kid == "" {
		email := cfg.getEmail()
		if email == "" {
			return nil, fmt.Errorf("email is required for ZeroSSL EAB auto-fetch")
		}

		if logger != nil {
			logger.Info("auto-fetching ZeroSSL EAB credentials", slog.String("email", email))
		}

		fetched, err := fetchZeroSSLEAB(email)
		if err != nil {
			return nil, fmt.Errorf("auto-fetch ZeroSSL EAB: %w", err)
		}
		kid = fetched.KID
		hmacKey = fetched.HMACKey

		if logger != nil {
			logger.Info("ZeroSSL EAB credentials obtained", slog.String("kid", kid))
		}
	}

	// Register with or without EAB
	if kid != "" {
		reg, err := client.Registration.RegisterWithExternalAccountBinding(
			registration.RegisterEABOptions{
				TermsOfServiceAgreed: true,
				Kid:                  kid,
				HmacEncoded:          hmacKey,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("register ACME account with EAB: %w", err)
		}
		return reg, nil
	}

	reg, err := client.Registration.Register(
		registration.RegisterOptions{TermsOfServiceAgreed: true},
	)
	if err != nil {
		return nil, fmt.Errorf("register ACME account: %w", err)
	}
	return reg, nil
}

// ZeroSSLEAB contains auto-fetched ZeroSSL EAB credentials.
type ZeroSSLEAB struct {
	KID     string `json:"eab_kid"`
	HMACKey string `json:"eab_hmac_key"`
}

// fetchZeroSSLEAB calls the ZeroSSL API to obtain EAB credentials for the given email.
func fetchZeroSSLEAB(email string) (*ZeroSSLEAB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	form := url.Values{"email": []string{email}}
	body := strings.NewReader(form.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zeroSSLAPI, body)
	if err != nil {
		return nil, fmt.Errorf("forming request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting EAB credentials: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success    bool   `json:"success"`
		EABKID     string `json:"eab_kid"`
		EABHMACKey string `json:"eab_hmac_key"`
		Error      struct {
			Code int    `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if !result.Success || result.Error.Code != 0 {
		return nil, fmt.Errorf("ZeroSSL API error: %s (code %d)", result.Error.Type, result.Error.Code)
	}
	if result.EABKID == "" || result.EABHMACKey == "" {
		return nil, fmt.Errorf("ZeroSSL returned empty EAB credentials")
	}

	return &ZeroSSLEAB{KID: result.EABKID, HMACKey: result.EABHMACKey}, nil
}

// certConfig abstracts IP and domain cert configs for unified EAB handling.
type certConfig interface {
	getCA() opt.CA
	getEmail() string
	getEABKid() string
	getEABHMACKey() string
}

func (c IPCertConfig) getCA() opt.CA         { return c.CA }
func (c IPCertConfig) getEmail() string      { return c.User.Email }
func (c IPCertConfig) getEABKid() string     { return c.EABKid }
func (c IPCertConfig) getEABHMACKey() string { return c.EABHMACKey }

func (c DomainCertConfig) getCA() opt.CA         { return c.CA }
func (c DomainCertConfig) getEmail() string      { return c.User.Email }
func (c DomainCertConfig) getEABKid() string     { return c.EABKid }
func (c DomainCertConfig) getEABHMACKey() string { return c.EABHMACKey }
