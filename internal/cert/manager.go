// Package cert provides automatic TLS certificate management via ACME (Let's Encrypt / ZeroSSL).
// It supports IP certificates (RFC 8738) via TLS-ALPN-01 or HTTP-01 challenges,
// and domain certificates via DNS-01 challenge providers.
package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certificate"

	"github.com/tuotoo/biu/opt"
)

const (
	ipRenewBefore     = 48 * time.Hour
	ipRenewCheck      = 12 * time.Hour
	domainRenewBefore = 30 * 24 * time.Hour
	domainRenewCheck  = 24 * time.Hour

	maxRetries      = 3
	defaultCacheDir = "./.biu-certs"
)

// CertManager manages certificate lifecycle: obtain, cache, hot-reload, auto-renew.
type CertManager struct {
	cfg  opt.TLSConfig
	user *CertUser

	mu      sync.RWMutex
	cert    *tls.Certificate
	certURL string

	renewCancel context.CancelFunc // cancels the auto-renewal goroutine
}

// NewCertManager creates a new CertManager. It loads cached certs or obtains new ones.
func NewCertManager(cfg opt.TLSConfig) (*CertManager, error) {
	if cfg.CacheDir == "" {
		cfg.CacheDir = defaultCacheDir
	}

	user, err := LoadOrCreateUser(cfg.CacheDir, cfg.Email)
	if err != nil {
		return nil, fmt.Errorf("load/create ACME user: %w", err)
	}

	cm := &CertManager{
		cfg:  cfg,
		user: user,
	}

	// Try to load from cache
	if err := cm.loadCache(); err == nil {
		if cm.isCacheValid() {
			cm.log().Info("using cached certificate")
			return cm, nil
		}
		cm.log().Info("cached certificate needs renewal")
	}

	// Obtain new certificate
	if err := cm.obtain(); err != nil {
		return nil, err
	}

	return cm, nil
}

// TLSConfig returns a *tls.Config with GetCertificate callback for hot-reloading.
func (cm *CertManager) TLSConfig() *tls.Config {
	cfg := &tls.Config{
		GetCertificate: cm.getCertificate,
		MinVersion:     tls.VersionTLS12,
		NextProtos:     []string{"h2", "http/1.1"},
	}

	// For TLS-ALPN-01 same-port mode (no separate ALPN listener),
	// add acme-tls/1 to NextProtos so the main listener handles ALPN.
	// When ALPN01Port is set, lego runs its own listener — don't add ALPN here.
	if cm.cfg.Mode == opt.TLSModeIP && cm.cfg.HTTP01Port == 0 && cm.cfg.ALPN01Port == 0 {
		cfg.NextProtos = append(cfg.NextProtos, "acme-tls/1")
		cfg.GetConfigForClient = cm.getConfigForClient
	}

	return cfg
}

// getConfigForClient handles TLS-ALPN-01 challenge on port 443.
func (cm *CertManager) getConfigForClient(hello *tls.ClientHelloInfo) (*tls.Config, error) {
	// Check if this is an ACME TLS-ALPN-01 validation request
	if slices.Contains(hello.SupportedProtos, "acme-tls/1") {
		// Use a separate config for ALPN challenge
		return &tls.Config{
			GetCertificate: cm.getCertificate,
			NextProtos:     []string{"acme-tls/1"},
			MinVersion:     tls.VersionTLS12,
		}, nil
	}
	return nil, nil // use default config
}

// getCertificate is the tls.Config.GetCertificate callback for hot-reloading.
func (cm *CertManager) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.cert == nil {
		return nil, fmt.Errorf("certificate not ready")
	}
	return cm.cert, nil
}

// StartAutoRenew starts background auto-renewal.
func (cm *CertManager) StartAutoRenew(ctx context.Context) {
	renewCtx, renewCancel := context.WithCancel(ctx)
	cm.renewCancel = renewCancel

	var checkInterval time.Duration
	if cm.cfg.Mode == opt.TLSModeIP {
		checkInterval = ipRenewCheck
	} else {
		checkInterval = domainRenewCheck
	}

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				if !cm.isCacheValid() {
					cm.log().Info("starting auto-renewal")
					if err := cm.renew(); err != nil {
						cm.log().Error("auto-renewal failed", slog.Any("err", err))
					}
				}
			}
		}
	}()
}

// Shutdown gracefully stops the cert manager.
func (cm *CertManager) Shutdown(ctx context.Context) error {
	if cm.renewCancel != nil {
		cm.renewCancel()
	}
	cm.log().Info("cert manager shutdown complete")
	return nil
}

// --- internal methods ---

func (cm *CertManager) log() *slog.Logger {
	if cm.cfg.Logger != nil {
		return cm.cfg.Logger
	}
	return slog.Default()
}

func (cm *CertManager) loadCache() error {
	key := cm.cfg.IP
	if cm.cfg.Mode == opt.TLSModeDomain {
		key = cm.cfg.Domain
	}

	cert, meta, err := LoadCert(cm.cfg.CacheDir, key)
	if err != nil {
		return err
	}

	cm.mu.Lock()
	cm.cert = cert
	cm.certURL = meta.CertURL
	cm.mu.Unlock()

	return nil
}

func (cm *CertManager) isCacheValid() bool {
	cm.mu.RLock()
	cert := cm.cert
	cm.mu.RUnlock()

	if cert == nil {
		return false
	}

	renewBefore := domainRenewBefore
	if cm.cfg.Mode == opt.TLSModeIP {
		renewBefore = ipRenewBefore
	}

	return IsCertValid(cert, renewBefore)
}

func (cm *CertManager) obtain() error {
	return cm.obtainWithRetry()
}

func (cm *CertManager) obtainWithRetry() error {
	var lastErr error
	delays := []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			cm.log().Info("retrying certificate obtain",
				slog.Int("attempt", attempt),
				slog.Duration("delay", delays[attempt-1]),
			)
			time.Sleep(delays[attempt-1])
		}

		var certRes *certificate.Resource
		var err error

		if cm.cfg.Mode == opt.TLSModeIP {
			certRes, err = ObtainIPCert(IPCertConfig{
				IP:         cm.cfg.IP,
				CacheDir:   cm.cfg.CacheDir,
				CA:         cm.cfg.CA,
				HTTP01Port: cm.cfg.HTTP01Port,
				ALPN01Port: cm.cfg.ALPN01Port,
				EABKid:     cm.cfg.EABKid,
				EABHMACKey: cm.cfg.EABHMACKey,
				Logger:     cm.cfg.Logger,
				User:       cm.user,
			})
		} else {
			certRes, err = ObtainDomainCert(DomainCertConfig{
				Domain:      cm.cfg.Domain,
				CacheDir:    cm.cfg.CacheDir,
				CA:          cm.cfg.CA,
				DNSProvider: cm.cfg.DNSProvider,
				EABKid:      cm.cfg.EABKid,
				EABHMACKey:  cm.cfg.EABHMACKey,
				Logger:      cm.cfg.Logger,
				User:        cm.user,
			})
		}

		if err == nil {
			return cm.loadFromResource(certRes)
		}

		lastErr = err
		cm.log().Error("certificate obtain failed",
			slog.Int("attempt", attempt+1),
			slog.Any("err", err),
		)
	}

	return fmt.Errorf("certificate obtain failed after %d retries: %w", maxRetries+1, lastErr)
}

func (cm *CertManager) loadFromResource(res *certificate.Resource) error {
	cert, err := tls.X509KeyPair(res.Certificate, res.PrivateKey)
	if err != nil {
		return fmt.Errorf("parse obtained certificate: %w", err)
	}
	if len(cert.Certificate) > 0 {
		cert.Leaf, _ = x509.ParseCertificate(cert.Certificate[0])
	}

	key := cm.cfg.IP
	if cm.cfg.Mode == opt.TLSModeDomain {
		key = cm.cfg.Domain
	}

	meta := CertMeta{
		Domain:  res.Domain,
		CertURL: res.CertURL,
	}

	if err := SaveCert(cm.cfg.CacheDir, key, res.Certificate, res.PrivateKey, meta); err != nil {
		return fmt.Errorf("save certificate: %w", err)
	}

	cm.mu.Lock()
	cm.cert = &cert
	cm.certURL = res.CertURL
	cm.mu.Unlock()

	return nil
}

func (cm *CertManager) renew() error {
	cm.mu.RLock()
	certURL := cm.certURL
	cm.mu.RUnlock()

	if certURL == "" {
		return fmt.Errorf("no certificate URL, cannot renew")
	}

	// Load existing resource from cache
	key := cm.cfg.IP
	if cm.cfg.Mode == opt.TLSModeDomain {
		key = cm.cfg.Domain
	}

	_, meta, err := LoadCert(cm.cfg.CacheDir, key)
	if err != nil {
		cm.log().Error("cache corrupted, re-obtaining", slog.Any("err", err))
		return cm.obtain()
	}

	// Build a resource from cache for renewal
	certRes := &certificate.Resource{
		Domain:  meta.Domain,
		CertURL: meta.CertURL,
	}

	var newCert *certificate.Resource

	if cm.cfg.Mode == opt.TLSModeIP {
		newCert, err = RenewIPCert(IPCertConfig{
			IP:         cm.cfg.IP,
			CacheDir:   cm.cfg.CacheDir,
			CA:         cm.cfg.CA,
			HTTP01Port: cm.cfg.HTTP01Port,
			ALPN01Port: cm.cfg.ALPN01Port,
			Logger:     cm.cfg.Logger,
			User:       cm.user,
		}, certRes)
	} else {
		newCert, err = RenewDomainCert(DomainCertConfig{
			Domain:      cm.cfg.Domain,
			CacheDir:    cm.cfg.CacheDir,
			CA:          cm.cfg.CA,
			DNSProvider: cm.cfg.DNSProvider,
			Logger:      cm.cfg.Logger,
			User:        cm.user,
		}, certRes)
	}

	if err != nil {
		return fmt.Errorf("renew failed: %w", err)
	}

	return cm.loadFromResource(newCert)
}
