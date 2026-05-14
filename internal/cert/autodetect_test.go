package cert

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectPublicIP_Explicit(t *testing.T) {
	ip := "1.2.3.4"
	result, err := DetectPublicIP(ip)
	assert.NoError(t, err)
	assert.Equal(t, ip, result)
}

func TestDetectPublicIP_Empty(t *testing.T) {
	// When empty string is passed, auto-detect from network interfaces.
	// This test may not find a public IP on machines behind NAT (CI, dev laptops).
	ip, err := DetectPublicIP("")
	if err != nil {
		t.Skipf("no public IP on this machine (expected in NAT/CI): %v", err)
		return
	}
	// Must be a valid IP
	parsed := net.ParseIP(ip)
	assert.NotNil(t, parsed, "detected IP should be valid: %s", ip)
	// Should not be private
	assert.False(t, parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast(), "detected IP should be public")
}

func TestDetectPublicIP_InvalidIP(t *testing.T) {
	_, err := DetectPublicIP("not-an-ip")
	assert.Error(t, err)
}
