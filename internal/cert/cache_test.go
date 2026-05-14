package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCert(t *testing.T, domain string, notAfter time.Time) *tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	return &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

func TestSaveAndLoadCert(t *testing.T) {
	dir := t.TempDir()
	cert := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))
	meta := CertMeta{Domain: "example.com", CertURL: "https://acme.example.com/cert/123"}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	err = SaveCert(dir, "example.com", certPEM, keyPEM, meta)
	require.NoError(t, err)

	loaded, loadedMeta, err := LoadCert(dir, "example.com")
	require.NoError(t, err)
	assert.Equal(t, meta.Domain, loadedMeta.Domain)
	assert.Equal(t, meta.CertURL, loadedMeta.CertURL)
	assert.NotNil(t, loaded)
	assert.NotNil(t, loaded.PrivateKey)
}

func TestLoadCert_NotExist(t *testing.T) {
	dir := t.TempDir()
	_, _, err := LoadCert(dir, "nonexistent")
	assert.Error(t, err)
}

func TestSaveCert_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "path")
	cert := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	err = SaveCert(dir, "example.com", certPEM, keyPEM, CertMeta{})
	require.NoError(t, err)

	// Verify the directory was created
	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestIsCertValid_Valid(t *testing.T) {
	cert := generateTestCert(t, "example.com", time.Now().Add(90*24*time.Hour))
	assert.True(t, IsCertValid(cert, 30*24*time.Hour))
}

func TestIsCertValid_Expired(t *testing.T) {
	cert := generateTestCert(t, "example.com", time.Now().Add(-time.Hour))
	assert.False(t, IsCertValid(cert, 30*24*time.Hour))
}

func TestIsCertValid_NearExpiry(t *testing.T) {
	// Certificate expires in 5 days, renewBefore is 30 days
	cert := generateTestCert(t, "example.com", time.Now().Add(5*24*time.Hour))
	assert.False(t, IsCertValid(cert, 30*24*time.Hour))
}

func TestIsCertValid_JustBeforeRenew(t *testing.T) {
	// Certificate expires in 31 days, renewBefore is 30 days
	cert := generateTestCert(t, "example.com", time.Now().Add(31*24*time.Hour))
	assert.True(t, IsCertValid(cert, 30*24*time.Hour))
}
