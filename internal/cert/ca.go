package cert

import "github.com/tuotoo/biu/opt"

const (
	letsEncryptProduction = "https://acme-v02.api.letsencrypt.org/directory"
	zeroSSLProduction     = "https://acme.zerossl.com/v2/DV90"
)

// caDirectoryURL returns the ACME directory URL for the given CA.
func caDirectoryURL(ca opt.CA) string {
	switch ca {
	case opt.CAZeroSSL:
		return zeroSSLProduction
	default:
		return letsEncryptProduction
	}
}
