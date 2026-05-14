package cert

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"

	"github.com/tuotoo/biu/opt"
)

const (
	shortlivedProfile = "shortlived"
)

// IPCertConfig configures IP certificate acquisition.
type IPCertConfig struct {
	IP         string
	CacheDir   string
	CA         opt.CA
	HTTP01Port int    // non-zero = use HTTP-01 challenge
	ALPN01Port int    // non-zero = separate TLS-ALPN-01 listener port (for non-443 HTTPS)
	EABKid     string // ZeroSSL EAB KID
	EABHMACKey string // ZeroSSL EAB HMAC key
	Logger     *slog.Logger
	User       *CertUser
}

// ObtainIPCert obtains a new IP certificate via ACME.
func ObtainIPCert(cfg IPCertConfig) (*certificate.Resource, error) {
	caDirURL := caDirectoryURL(cfg.CA)
	config := lego.NewConfig(cfg.User)
	config.CADirURL = caDirURL
	config.Certificate.KeyType = certcrypto.EC256
	config.Certificate.DisableCommonName = true

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create lego client: %w", err)
	}

	// Register user if not already registered
	if cfg.User.Registration == nil {
		reg, err := registerAccount(client, cfg, cfg.CacheDir, cfg.Logger)
		if err != nil {
			return nil, err
		}
		cfg.User.Registration = reg
		_ = SaveUser(cfg.CacheDir, cfg.User)
	}

	// Set up challenge provider
	if cfg.HTTP01Port != 0 {
		provider := http01.NewProviderServer("", strconv.Itoa(cfg.HTTP01Port))
		if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
			return nil, fmt.Errorf("set HTTP-01 provider: %w", err)
		}
		if cfg.Logger != nil {
			cfg.Logger.Info("using HTTP-01 challenge", slog.Int("port", cfg.HTTP01Port))
		}
	} else {
		alpnPort := cfg.ALPN01Port
		portStr := ""
		if alpnPort != 0 {
			portStr = strconv.Itoa(alpnPort)
		}
		provider := tlsalpn01.NewProviderServer("", portStr)
		if err := client.Challenge.SetTLSALPN01Provider(provider); err != nil {
			return nil, fmt.Errorf("set TLS-ALPN-01 provider: %w", err)
		}
		if cfg.Logger != nil {
			if alpnPort != 0 {
				cfg.Logger.Info("using TLS-ALPN-01 challenge", slog.Int("alpnPort", alpnPort))
			} else {
				cfg.Logger.Info("using TLS-ALPN-01 challenge (same port as HTTPS)")
			}
		}
	}

	request := certificate.ObtainRequest{
		Domains: []string{cfg.IP},
		Bundle:  true,
		Profile: shortlivedProfile,
	}

	certRes, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("obtain IP certificate: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("IP certificate obtained",
			slog.String("domain", certRes.Domain),
			slog.String("certURL", certRes.CertURL),
		)
	}

	return certRes, nil
}

// RenewIPCert renews an existing IP certificate.
func RenewIPCert(cfg IPCertConfig, certRes *certificate.Resource) (*certificate.Resource, error) {
	caDirURL := caDirectoryURL(cfg.CA)
	config := lego.NewConfig(cfg.User)
	config.CADirURL = caDirURL
	config.Certificate.KeyType = certcrypto.EC256
	config.Certificate.DisableCommonName = true

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create lego client: %w", err)
	}

	newCert, err := client.Certificate.RenewWithOptions(*certRes, &certificate.RenewOptions{
		Bundle:  true,
		Profile: shortlivedProfile,
	})
	if err != nil {
		return nil, fmt.Errorf("renew IP certificate: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("IP certificate renewed",
			slog.String("domain", newCert.Domain),
			slog.String("certURL", newCert.CertURL),
		)
	}

	return newCert, nil
}
