package services

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ValidatorService handles URL validation and sanitization
type ValidatorService struct{}

// NewValidatorService creates a new validator service
func NewValidatorService() *ValidatorService {
	return &ValidatorService{}
}

// ValidateAndSanitizeURL validates and sanitizes a destination URL
func (v *ValidatorService) ValidateAndSanitizeURL(rawURL string) (string, error) {
	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.New("invalid URL format")
	}

	// Require HTTP or HTTPS scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("URL must use http or https scheme")
	}

	// Require host
	if parsed.Host == "" {
		return "", errors.New("URL must have a host")
	}

	// Block local and private IPs (SSRF prevention)
	if err := v.checkHostSafety(parsed.Hostname()); err != nil {
		return "", err
	}

	// Max length check
	if len(rawURL) > 2048 {
		return "", errors.New("URL too long (max 2048 characters)")
	}

	// Normalize URL
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = "" // Remove fragment

	return parsed.String(), nil
}

// checkHostSafety checks if a hostname is safe (not local/private)
func (v *ValidatorService) checkHostSafety(hostname string) error {
	// Try to parse as IP
	ip := net.ParseIP(hostname)
	if ip != nil {
		// Check if IP is loopback or private
		if ip.IsLoopback() {
			return errors.New("loopback addresses not allowed")
		}
		if ip.IsPrivate() {
			return errors.New("private IP addresses not allowed")
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.New("link-local addresses not allowed")
		}
	}

	// Check for localhost variants
	hostname = strings.ToLower(hostname)
	if hostname == "localhost" || strings.HasSuffix(hostname, ".local") {
		return errors.New("localhost addresses not allowed")
	}

	// Block common internal hostnames
	blocked := []string{
		"metadata.google.internal",
		"169.254.169.254", // AWS/GCP metadata service
		"127.0.0.1",
		"0.0.0.0",
		"::1",
	}
	for _, b := range blocked {
		if hostname == b {
			return errors.New("blocked hostname")
		}
	}

	return nil
}
