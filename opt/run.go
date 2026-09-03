package opt

import (
	"context"
	"log/slog"

	"github.com/go-acme/lego/v4/challenge"
)

// RunFunc is the type of running config functions.
type RunFunc func(*Run)

// CA represents a certificate authority.
type CA int

const (
	// CALetsEncrypt uses Let's Encrypt as the CA (default).
	CALetsEncrypt CA = iota
	// CAZeroSSL uses ZeroSSL as the CA.
	CAZeroSSL
)

const (
	// TLSModeIP is the IP certificate mode (RFC 8738 shortlived).
	TLSModeIP = "ip"
	// TLSModeDomain is the domain certificate mode (DNS-01).
	TLSModeDomain = "domain"
)

// TLSConfig aggregates all TLS configuration.
type TLSConfig struct {
	Mode        string             // "ip" | "domain"
	IP          string             // public IP (empty = auto-detect)
	Domain      string             // domain name
	Email       string             // ACME account email
	CacheDir    string             // certificate cache directory (default ./.biu-certs)
	CA          CA                 // CA selection (default CALetsEncrypt)
	HTTP01Port  int                // HTTP-01 internal listener port (0 = use TLS-ALPN-01)
	ALPN01Port  int                // TLS-ALPN-01 listener port (0 = same as HTTPS port)
	DNSProvider challenge.Provider // DNS-01 challenge provider
	EABKid      string             // ZeroSSL EAB KID (required for ZeroSSL)
	EABHMACKey  string             // ZeroSSL EAB HMAC key (required for ZeroSSL)
	Logger      *slog.Logger       // logger for certificate operations
}

// Run is the running options of container.
type Run struct {
	BeforeShutDown func()
	AfterShutDown  func()
	AfterStart     func()
	Ctx            context.Context
	Cancel         context.CancelFunc
	TLS            *TLSConfig // nil means TLS is not enabled
	Pprof          *PprofConfig // nil means pprof is not enabled
}

// PprofConfig controls the standalone pprof debug server.
// The server listens on its own address (loopback by default) and is
// independent from the main HTTP/TLS service.
type PprofConfig struct {
	// Addr is the listen address for the pprof server.
	// Default "127.0.0.1:6060". Use a loopback address in production.
	Addr string
	// Token, when non-empty, requires every request to carry token=<value>
	// as a query parameter. Requests without a matching token get 404.
	Token string
}

func AfterStart(f func()) RunFunc {
	return func(opt *Run) {
		opt.AfterStart = f
	}
}

// BeforeShutDown will run before http server shuts down.
func BeforeShutDown(f func()) RunFunc {
	return func(opt *Run) {
		opt.BeforeShutDown = f
	}
}

// AfterShutDown will run after http server shuts down.
func AfterShutDown(f func()) RunFunc {
	return func(opt *Run) {
		opt.AfterShutDown = f
	}
}

func WithContext(ctx context.Context, cancel context.CancelFunc) RunFunc {
	return func(opt *Run) {
		opt.Ctx = ctx
		opt.Cancel = cancel
	}
}

// WithTLSIP enables IP certificate mode (RFC 8738).
// ip can be empty to auto-detect from network interfaces.
func WithTLSIP(ip, email string) RunFunc {
	return func(opt *Run) {
		opt.TLS = &TLSConfig{
			Mode:     TLSModeIP,
			IP:       ip,
			Email:    email,
			CacheDir: "./.biu-certs",
			CA:       CALetsEncrypt,
		}
	}
}

// WithTLSDomain enables domain certificate mode (DNS-01 challenge).
func WithTLSDomain(domain, email string) RunFunc {
	return func(opt *Run) {
		opt.TLS = &TLSConfig{
			Mode:     TLSModeDomain,
			Domain:   domain,
			Email:    email,
			CacheDir: "./.biu-certs",
			CA:       CALetsEncrypt,
		}
	}
}

// WithTLSDNSProvider sets the DNS-01 challenge provider for domain certificates.
func WithTLSDNSProvider(p challenge.Provider) RunFunc {
	return func(opt *Run) {
		if opt.TLS != nil {
			opt.TLS.DNSProvider = p
		}
	}
}

// WithTLSHTTP01 sets the HTTP-01 challenge internal listener port for IP certificates.
func WithTLSHTTP01(port int) RunFunc {
	return func(opt *Run) {
		if opt.TLS != nil {
			opt.TLS.HTTP01Port = port
		}
	}
}

// WithTLSALPN01 sets the TLS-ALPN-01 challenge listener port for IP certificates.
// Default is 0 (same as HTTPS port). Use this when HTTPS runs on a non-443 port
// but Let's Encrypt ALPN validation must reach port 443.
func WithTLSALPN01(port int) RunFunc {
	return func(opt *Run) {
		if opt.TLS != nil {
			opt.TLS.ALPN01Port = port
		}
	}
}

// WithTLSCacheDir sets the certificate cache directory.
func WithTLSCacheDir(dir string) RunFunc {
	return func(opt *Run) {
		if opt.TLS != nil {
			opt.TLS.CacheDir = dir
		}
	}
}

// WithTLSEAB sets ZeroSSL External Account Binding credentials.
// Required when using ZeroSSL as the CA. Obtain KID and HMAC key from
// https://app.zerossl.com/developer.
func WithTLSEAB(kid, hmacKey string) RunFunc {
	return func(opt *Run) {
		if opt.TLS != nil {
			opt.TLS.EABKid = kid
			opt.TLS.EABHMACKey = hmacKey
		}
	}
}

// WithTLSCA selects the certificate authority.
func WithTLSCA(ca CA) RunFunc {
	return func(opt *Run) {
		if opt.TLS != nil {
			opt.TLS.CA = ca
		}
	}
}

// WithPprof enables a standalone pprof debug server listening on addr.
// The server is independent from the main HTTP/TLS service and shares
// the Run lifecycle (shutdown happens alongside the main server).
// addr defaults to "127.0.0.1:6060" when empty. Bind to a loopback
// address in production; pprof exposes runtime profile data.
func WithPprof(addr string) RunFunc {
	return func(opt *Run) {
		if opt.Pprof == nil {
			opt.Pprof = &PprofConfig{}
		}
		opt.Pprof.Addr = addr
	}
}

// WithPprofToken gates the pprof server: every request must carry
// token=<value> as a query parameter. Requests without a matching
// token get 404. Must be called after WithPprof.
func WithPprofToken(token string) RunFunc {
	return func(opt *Run) {
		if opt.Pprof == nil {
			opt.Pprof = &PprofConfig{}
		}
		opt.Pprof.Token = token
	}
}
