package cert

import (
	"fmt"
	"net"
)

// DetectPublicIP returns the public IP address.
// If ip is non-empty, it validates and returns it as-is.
// If ip is empty, it auto-detects the first public (non-RFC 1918 / non-loopback / non-link-local) IPv4 address
// from available network interfaces.
func DetectPublicIP(ip string) (string, error) {
	if ip != "" {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return "", fmt.Errorf("invalid IP address: %s", ip)
		}
		return ip, nil
	}

	// Auto-detect from network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to list network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ipAddr := ipNet.IP
			// Only IPv4
			if ipAddr.To4() == nil {
				continue
			}
			// Exclude private, loopback, link-local
			if ipAddr.IsPrivate() || ipAddr.IsLoopback() || ipAddr.IsLinkLocalUnicast() {
				continue
			}
			return ipAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no public IP address found on any network interface; please specify IP explicitly")
}
