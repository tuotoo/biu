package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tuotoo/biu/opt"
)

func TestNewCertManager_CacheHit(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(7*24*time.Hour))
	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	// Pre-seed cache with a valid certificate
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "1.2.3.4",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	cm, err := NewCertManager(cfg)
	require.NoError(t, err)
	require.NotNil(t, cm)

	// Verify certificate is loaded
	tlsCfg := cm.TLSConfig()
	require.NotNil(t, tlsCfg)
	assert.NotNil(t, tlsCfg.GetCertificate)

	// Verify shutdown doesn't panic
	assert.NoError(t, cm.Shutdown(nil))
}

func TestNewCertManager_CacheMiss_NoNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: makes real ACME API calls")
	}
	dir := t.TempDir()

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "10.0.0.1",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	// No cache + no network → should fail (won't reach ACME server)
	_, err := NewCertManager(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "obtain")
}

func TestCertManager_TLSConfig_IP_ALPN(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(7*24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "1.2.3.4",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	cm, err := NewCertManager(cfg)
	require.NoError(t, err)

	tlsCfg := cm.TLSConfig()
	require.NotNil(t, tlsCfg)

	// Should include acme-tls/1 in NextProtos for ALPN
	assert.Contains(t, tlsCfg.NextProtos, "acme-tls/1")
	assert.Contains(t, tlsCfg.NextProtos, "h2")
	assert.Contains(t, tlsCfg.NextProtos, "http/1.1")

	// GetConfigForClient should be set for ALPN
	assert.NotNil(t, tlsCfg.GetConfigForClient)
}

func TestCertManager_TLSConfig_Domain(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "example.com", certPEM, keyPEM, CertMeta{
		Domain:  "example.com",
		CertURL: "https://acme.example.com/cert/2",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeDomain,
		Domain:   "example.com",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	cm, err := NewCertManager(cfg)
	require.NoError(t, err)

	tlsCfg := cm.TLSConfig()
	require.NotNil(t, tlsCfg)

	// Domain mode: no acme-tls/1, no GetConfigForClient
	assert.NotContains(t, tlsCfg.NextProtos, "acme-tls/1")
	assert.Nil(t, tlsCfg.GetConfigForClient)
}

func TestCertManager_TLSConfig_IP_ALPN_SeparatePort(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(7*24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:       opt.TLSModeIP,
		IP:         "1.2.3.4",
		Email:      "test@example.com",
		CacheDir:   dir,
		ALPN01Port: 443, // separate ALPN listener
	}

	cm, err := NewCertManager(cfg)
	require.NoError(t, err)

	tlsCfg := cm.TLSConfig()
	require.NotNil(t, tlsCfg)

	// When ALPN01Port is set, acme-tls/1 NOT on the main listener (lego handles separately)
	assert.NotContains(t, tlsCfg.NextProtos, "acme-tls/1")
	assert.Nil(t, tlsCfg.GetConfigForClient)
}

func TestCertManager_Shutdown(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(7*24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "1.2.3.4",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	cm, err := NewCertManager(cfg)
	require.NoError(t, err)

	// Shutdown without renewal (renewCancel is nil) — should not panic
	assert.NoError(t, cm.Shutdown(nil))
}

func TestCertManager_Shutdown_WithRenewal(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(7*24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "1.2.3.4",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	cm, err := NewCertManager(cfg)
	require.NoError(t, err)

	// Start auto-renewal in a background goroutine
	cm.StartAutoRenew(t.Context())

	// Shutdown should cancel the renewal goroutine
	assert.NoError(t, cm.Shutdown(nil))
}

func TestCertManager_CacheExpired_NoNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: makes real ACME API calls")
	}
	dir := t.TempDir()
	// Create an expired cert
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(-time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "1.2.3.4",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	// Expired cache → tries to obtain → fails (no network)
	_, err := NewCertManager(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "obtain")
}

func TestCertManager_CacheNearExpiry_IP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: makes real ACME API calls")
	}
	dir := t.TempDir()
	// IP cert expiring in 24h (renewBefore is 48h)
	cert := generateTestCert(t, "1.2.3.4", time.Now().Add(24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, SaveCert(dir, "1.2.3.4", certPEM, keyPEM, CertMeta{
		Domain:  "1.2.3.4",
		CertURL: "https://acme.example.com/cert/1",
	}))

	cfg := opt.TLSConfig{
		Mode:     opt.TLSModeIP,
		IP:       "1.2.3.4",
		Email:    "test@example.com",
		CacheDir: dir,
	}

	// Near expiry → tries to obtain → fails (no network)
	_, err := NewCertManager(cfg)
	assert.Error(t, err)
}

func TestCaDirectoryURL(t *testing.T) {
	assert.Equal(t, "https://acme-v02.api.letsencrypt.org/directory", caDirectoryURL(opt.CALetsEncrypt))
	assert.Equal(t, "https://acme.zerossl.com/v2/DV90", caDirectoryURL(opt.CAZeroSSL))
}

func TestCertMeta_Serialization(t *testing.T) {
	dir := t.TempDir()
	meta := CertMeta{Domain: "example.com", CertURL: "https://acme.example.com/cert/1"}

	require.NoError(t, SaveCert(dir, "test", []byte("fake-cert"), []byte("fake-key"), meta))

	// Write a PEM cert so LoadCert can parse it
	cert := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))
	keyDER, _ := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	// Overwrite just the cert/key files, keep the meta from above
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.crt"), certPEM, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.key"), keyPEM, 0600))

	_, loadedMeta, err := LoadCert(dir, "test")
	require.NoError(t, err)
	assert.Equal(t, "example.com", loadedMeta.Domain)
	assert.Equal(t, "https://acme.example.com/cert/1", loadedMeta.CertURL)
}

func TestCertManager_getCertificate_NotReady(t *testing.T) {
	cm := &CertManager{}
	_, err := cm.getCertificate(&tls.ClientHelloInfo{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}
