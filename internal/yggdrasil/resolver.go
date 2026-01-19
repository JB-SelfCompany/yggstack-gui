package yggdrasil

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"

	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil/netstack"
)

const (
	// NameMappingSuffix is the suffix for public key based DNS resolution
	NameMappingSuffix = ".pk.ygg"
)

// NameResolver resolves hostnames for SOCKS5 proxy
// It supports:
// - Direct IPv6 addresses (passthrough)
// - .pk.ygg suffix (public key to address mapping)
// - Regular DNS (if nameserver is configured)
type NameResolver struct {
	resolver   *net.Resolver
	netstack   *netstack.YggdrasilNetstack
	nameserver string
}

// NewNameResolver creates a new name resolver
func NewNameResolver(ns *netstack.YggdrasilNetstack, nameserver string) *NameResolver {
	nr := &NameResolver{
		netstack:   ns,
		nameserver: nameserver,
	}

	// Create custom resolver if nameserver is configured
	if nameserver != "" {
		nr.resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				// Parse nameserver address - support both "host" and "host:port" formats
				host, port, err := net.SplitHostPort(nameserver)
				if err != nil {
					// No port specified, default to "dns" (port 53)
					port = "53"
					host = nameserver
				}
				address = net.JoinHostPort(host, port)
				// Route DNS queries through Yggdrasil
				return ns.DialContext(ctx, network, address)
			},
		}
	}

	return nr
}

// Resolve implements the socks5.NameResolver interface
func (nr *NameResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	// Check for .pk.ygg suffix (public key based resolution)
	if strings.HasSuffix(name, NameMappingSuffix) {
		ip, err := nr.resolvePkYgg(name)
		if err != nil {
			return ctx, nil, err
		}
		return ctx, ip, nil
	}

	// Check if it's already an IP address
	if ip := net.ParseIP(name); ip != nil {
		return ctx, ip, nil
	}

	// Use configured nameserver for regular DNS
	if nr.resolver != nil {
		ips, err := nr.resolver.LookupIP(ctx, "ip6", name)
		if err != nil {
			// Try IPv4 as well
			ips, err = nr.resolver.LookupIP(ctx, "ip4", name)
			if err != nil {
				return ctx, nil, fmt.Errorf("DNS lookup failed for %s: %w", name, err)
			}
		}
		if len(ips) > 0 {
			return ctx, ips[0], nil
		}
		return ctx, nil, fmt.Errorf("no IP addresses found for %s", name)
	}

	return ctx, nil, fmt.Errorf("cannot resolve %s: no nameserver configured (only .pk.ygg addresses supported without nameserver)", name)
}

// resolvePkYgg resolves a .pk.ygg address to an IPv6 address
// Format: <64-hex-chars>.pk.ygg -> IPv6 address
func (nr *NameResolver) resolvePkYgg(name string) (net.IP, error) {
	// Remove the .pk.ygg suffix
	pubKeyHex := strings.TrimSuffix(name, NameMappingSuffix)

	// Decode the hex public key
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key hex in %s: %w", name, err)
	}

	// Validate public key length (Ed25519 public key is 32 bytes)
	if len(pubKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid public key length in %s: expected 32 bytes, got %d", name, len(pubKeyBytes))
	}

	// Derive IPv6 address from public key
	addr := address.AddrForKey(pubKeyBytes)
	if addr == nil {
		return nil, fmt.Errorf("failed to derive address from public key in %s", name)
	}

	return net.IP(addr[:]), nil
}
