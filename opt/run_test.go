package opt_test

import (
	"context"
	"testing"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/stretchr/testify/assert"

	"github.com/tuotoo/biu/opt"
)

func TestBeforeShutDown(t *testing.T) {
	run := &opt.Run{}
	a := 1
	opt.BeforeShutDown(func() {
		a = 2
	})(run)
	run.BeforeShutDown()
	assert.Equal(t, 2, a)
}

func TestAfterShutDown(t *testing.T) {
	run := &opt.Run{}
	a := 1
	opt.AfterShutDown(func() {
		a = 2
	})(run)
	run.AfterShutDown()
	assert.Equal(t, 2, a)
}

func TestAfterStart(t *testing.T) {
	run := &opt.Run{}
	a := 1
	opt.AfterStart(func() {
		a = 2
	})(run)
	run.AfterStart()
	assert.Equal(t, 2, a)
}

func TestWithContext(t *testing.T) {
	run := &opt.Run{}
	ctx, cancel := context.WithCancel(context.Background())
	opt.WithContext(ctx, cancel)(run)
	cancel()
	err := run.Ctx.Err()
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWithTLSIP(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	assert.NotNil(t, run.TLS)
	assert.Equal(t, "ip", run.TLS.Mode)
	assert.Equal(t, "1.2.3.4", run.TLS.IP)
	assert.Equal(t, "admin@example.com", run.TLS.Email)
}

func TestWithTLSIP_AutoDetect(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("", "admin@example.com")(run)
	assert.NotNil(t, run.TLS)
	assert.Equal(t, "ip", run.TLS.Mode)
	assert.Empty(t, run.TLS.IP)
	assert.Equal(t, "admin@example.com", run.TLS.Email)
}

func TestWithTLSDomain(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSDomain("api.example.com", "admin@example.com")(run)
	assert.NotNil(t, run.TLS)
	assert.Equal(t, "domain", run.TLS.Mode)
	assert.Equal(t, "api.example.com", run.TLS.Domain)
	assert.Equal(t, "admin@example.com", run.TLS.Email)
}

func TestWithTLSDNSProvider(t *testing.T) {
	run := &opt.Run{}
	// DNS provider already set by WithTLSDomain
	opt.WithTLSDomain("api.example.com", "admin@example.com")(run)
	p := &mockDNSProvider{}
	opt.WithTLSDNSProvider(p)(run)
	assert.NotNil(t, run.TLS.DNSProvider)
}

func TestWithTLSHTTP01(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	opt.WithTLSHTTP01(6080)(run)
	assert.Equal(t, 6080, run.TLS.HTTP01Port)
}

func TestWithTLSALPN01(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	opt.WithTLSALPN01(443)(run)
	assert.Equal(t, 443, run.TLS.ALPN01Port)
}

func TestWithTLSALPN01_NilTLS(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSALPN01(443)(run)
	assert.Nil(t, run.TLS) // no-op when TLS not enabled
}

func TestWithTLSCacheDir_ModifiesDefault(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	opt.WithTLSCacheDir("/tmp/certs")(run)
	assert.Equal(t, "/tmp/certs", run.TLS.CacheDir)
}

func TestWithTLSCacheDir_NilTLS(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSCacheDir("/tmp/certs")(run)
	assert.Nil(t, run.TLS) // no-op when TLS not enabled
}

func TestWithTLSCA_LetsEncrypt(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	opt.WithTLSCA(opt.CALetsEncrypt)(run)
	assert.Equal(t, opt.CALetsEncrypt, run.TLS.CA)
}

func TestWithTLSCA_ZeroSSL(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	opt.WithTLSCA(opt.CAZeroSSL)(run)
	assert.Equal(t, opt.CAZeroSSL, run.TLS.CA)
}

func TestTLS_FullConfigChain(t *testing.T) {
	run := &opt.Run{}
	p := &mockDNSProvider{}
	opt.WithTLSDomain("api.example.com", "admin@example.com")(run)
	opt.WithTLSCA(opt.CAZeroSSL)(run)
	opt.WithTLSDNSProvider(p)(run)
	opt.WithTLSCacheDir("/custom/cache")(run)

	assert.NotNil(t, run.TLS)
	assert.Equal(t, "domain", run.TLS.Mode)
	assert.Equal(t, "api.example.com", run.TLS.Domain)
	assert.Equal(t, "admin@example.com", run.TLS.Email)
	assert.Equal(t, opt.CAZeroSSL, run.TLS.CA)
	assert.Equal(t, "/custom/cache", run.TLS.CacheDir)
	assert.NotNil(t, run.TLS.DNSProvider)
}

func TestWithoutTLS(t *testing.T) {
	run := &opt.Run{}
	assert.Nil(t, run.TLS)
}

func TestTLS_Defaults(t *testing.T) {
	run := &opt.Run{}
	opt.WithTLSIP("1.2.3.4", "admin@example.com")(run)
	assert.Equal(t, opt.CALetsEncrypt, run.TLS.CA)
	assert.Equal(t, "./.biu-certs", run.TLS.CacheDir)
	assert.Equal(t, 0, run.TLS.HTTP01Port)
	assert.Nil(t, run.TLS.DNSProvider)
}

// mockDNSProvider implements challenge.Provider for testing
type mockDNSProvider struct{}

func (m *mockDNSProvider) Present(domain, token, keyAuth string) error { return nil }
func (m *mockDNSProvider) CleanUp(domain, token, keyAuth string) error { return nil }

var _ challenge.Provider = (*mockDNSProvider)(nil)
