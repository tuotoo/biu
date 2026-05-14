package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CertMeta holds metadata for a cached certificate.
type CertMeta struct {
	Domain  string `json:"domain"`
	CertURL string `json:"cert_url"`
}

// SaveCert saves certificate PEM, key PEM, and metadata to disk.
func SaveCert(dir, key string, certPEM, keyPEM []byte, meta CertMeta) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	certPath := filepath.Join(dir, key+".crt")
	keyPath := filepath.Join(dir, key+".key")
	metaPath := filepath.Join(dir, key+".json")

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	if err := os.WriteFile(metaPath, metaBytes, 0600); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}

	return nil
}

// LoadCert loads a certificate from disk. Returns the tls.Certificate, metadata, and any error.
func LoadCert(dir, key string) (*tls.Certificate, CertMeta, error) {
	certPath := filepath.Join(dir, key+".crt")
	keyPath := filepath.Join(dir, key+".key")
	metaPath := filepath.Join(dir, key+".json")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, CertMeta{}, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, CertMeta{}, fmt.Errorf("read key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, CertMeta{}, fmt.Errorf("parse cert/key: %w", err)
	}

	// Parse leaf certificate
	if len(cert.Certificate) > 0 {
		cert.Leaf, _ = x509.ParseCertificate(cert.Certificate[0])
	}

	var meta CertMeta
	metaBytes, err := os.ReadFile(metaPath)
	if err == nil {
		_ = json.Unmarshal(metaBytes, &meta)
	}

	return &cert, meta, nil
}

// IsCertValid checks whether the certificate is still valid for at least renewBefore duration.
// Returns false if the certificate expires within the renewBefore window or is already expired.
func IsCertValid(cert *tls.Certificate, renewBefore time.Duration) bool {
	if cert == nil {
		return false
	}
	if cert.Leaf != nil {
		return time.Now().Add(renewBefore).Before(cert.Leaf.NotAfter)
	}
	// Fallback: parse from raw DER
	if len(cert.Certificate) == 0 {
		return false
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return false
	}
	return time.Now().Add(renewBefore).Before(leaf.NotAfter)
}
