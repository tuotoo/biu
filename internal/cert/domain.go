package cert

import (
	"fmt"
	"log/slog"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"

	"github.com/tuotoo/biu/opt"
)

// DomainCertConfig configures domain certificate acquisition.
type DomainCertConfig struct {
	Domain      string
	CacheDir    string
	CA          opt.CA
	DNSProvider challenge.Provider
	EABKid      string // ZeroSSL EAB KID
	EABHMACKey  string // ZeroSSL EAB HMAC key
	Logger      *slog.Logger
	User        *CertUser
}

// ObtainDomainCert obtains a new domain certificate via DNS-01 challenge.
func ObtainDomainCert(cfg DomainCertConfig) (*certificate.Resource, error) {
	caDirURL := caDirectoryURL(cfg.CA)
	config := lego.NewConfig(cfg.User)
	config.CADirURL = caDirURL
	config.Certificate.KeyType = certcrypto.EC256

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

	// Set DNS-01 provider
	if err := client.Challenge.SetDNS01Provider(cfg.DNSProvider); err != nil {
		return nil, fmt.Errorf("set DNS-01 provider: %w", err)
	}
	if cfg.Logger != nil {
		cfg.Logger.Info("using DNS-01 challenge")
	}

	request := certificate.ObtainRequest{
		Domains: []string{cfg.Domain},
		Bundle:  true,
	}

	certRes, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("obtain domain certificate: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("domain certificate obtained",
			slog.String("domain", certRes.Domain),
			slog.String("certURL", certRes.CertURL),
		)
	}

	return certRes, nil
}

// RenewDomainCert renews an existing domain certificate.
func RenewDomainCert(cfg DomainCertConfig, certRes *certificate.Resource) (*certificate.Resource, error) {
	caDirURL := caDirectoryURL(cfg.CA)
	config := lego.NewConfig(cfg.User)
	config.CADirURL = caDirURL
	config.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create lego client: %w", err)
	}

	// Re-set DNS provider for the new client
	if err := client.Challenge.SetDNS01Provider(cfg.DNSProvider); err != nil {
		return nil, fmt.Errorf("set DNS-01 provider: %w", err)
	}

	newCert, err := client.Certificate.RenewWithOptions(*certRes, &certificate.RenewOptions{
		Bundle: true,
	})
	if err != nil {
		return nil, fmt.Errorf("renew domain certificate: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("domain certificate renewed",
			slog.String("domain", newCert.Domain),
			slog.String("certURL", newCert.CertURL),
		)
	}

	return newCert, nil
}
