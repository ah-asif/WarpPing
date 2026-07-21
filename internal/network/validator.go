package network

import (
	"net"
	"regexp"
	"strings"
)

// hostnameRe matches a reasonably strict RFC-1123-style domain name:
// dot-separated labels of letters/digits/hyphens (no leading/trailing
// hyphen per label), ending in a 2+ letter TLD.
var hostnameRe = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}$`)

// IsValidHost reports whether s looks like a usable ping target: a valid
// IPv4/IPv6 address, or a syntactically valid domain name.
func IsValidHost(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if net.ParseIP(s) != nil {
		return true
	}
	return hostnameRe.MatchString(s)
}
