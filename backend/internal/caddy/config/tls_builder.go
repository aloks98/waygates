// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

import (
	"os"

	"go.uber.org/zap"
)

// TLSBuilder builds TLS configuration for Caddy.
type TLSBuilder struct {
	logger   *zap.Logger
	settings *Settings
}

// NewTLSBuilder creates a new TLS builder.
func NewTLSBuilder(logger *zap.Logger) *TLSBuilder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TLSBuilder{
		logger: logger,
	}
}

// SetSettings sets the application settings.
func (b *TLSBuilder) SetSettings(settings *Settings) *TLSBuilder {
	b.settings = settings
	return b
}

// Build generates the TLS app configuration for the given domains.
func (b *TLSBuilder) Build(domains []string) (*TLSApp, error) {
	if len(domains) == 0 {
		return nil, nil
	}

	// Basic TLS config with automatic certificate management
	tlsApp := &TLSApp{
		Certificates: &CertificatesConfig{
			Automate: domains,
		},
	}

	// If we have settings, configure ACME based on provider
	if b.settings != nil && b.settings.ACMEProvider != "" {
		issuer, err := b.buildIssuer()
		if err != nil {
			b.logger.Warn("Failed to build ACME issuer, using defaults",
				zap.String("provider", b.settings.ACMEProvider),
				zap.Error(err),
			)
		} else if issuer != nil {
			tlsApp.Automation = &AutomationConfig{
				Policies: []*AutomationPolicy{
					{
						Subjects: domains,
						Issuers:  []*Issuer{issuer},
					},
				},
			}
		}
	}

	return tlsApp, nil
}

// buildIssuer builds an ACME issuer configuration based on settings.
func (b *TLSBuilder) buildIssuer() (*Issuer, error) {
	if b.settings == nil {
		return nil, nil
	}

	provider := b.settings.ACMEProvider

	switch provider {
	case ACMEProviderOff, "":
		// No ACME - Caddy won't automatically get certificates
		return nil, nil

	case ACMEProviderHTTP:
		// Default HTTP challenge - just needs email
		return &Issuer{
			Module: "acme",
			Email:  b.settings.AdminEmail,
		}, nil

	case ACMEProviderCloudflare:
		return b.buildCloudflareIssuer()

	case ACMEProviderRoute53:
		return b.buildRoute53Issuer()

	case ACMEProviderDuckDNS:
		return b.buildDuckDNSIssuer()

	case ACMEProviderDigitalOcean:
		return b.buildDigitalOceanIssuer()

	case ACMEProviderHetzner:
		return b.buildHetznerIssuer()

	case ACMEProviderPorkbun:
		return b.buildPorkbunIssuer()

	case ACMEProviderAzure:
		return b.buildAzureIssuer()

	case ACMEProviderVultr:
		return b.buildVultrIssuer()

	case ACMEProviderNamecheap:
		return b.buildNamecheapIssuer()

	case ACMEProviderOVH:
		return b.buildOVHIssuer()

	default:
		b.logger.Warn("Unknown ACME provider, using HTTP challenge",
			zap.String("provider", provider),
		)
		return &Issuer{
			Module: "acme",
			Email:  b.settings.AdminEmail,
		}, nil
	}
}

// buildCloudflareIssuer builds a Cloudflare DNS challenge issuer.
func (b *TLSBuilder) buildCloudflareIssuer() (*Issuer, error) {
	apiToken := b.getCredential("CLOUDFLARE_API_TOKEN")
	if apiToken == "" {
		b.logger.Warn("CLOUDFLARE_API_TOKEN not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewCloudflareProvider(apiToken),
			},
		},
	}, nil
}

// buildRoute53Issuer builds a Route53 DNS challenge issuer.
func (b *TLSBuilder) buildRoute53Issuer() (*Issuer, error) {
	// Route53 uses AWS SDK credential chain
	// Credentials are typically set via AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, etc.
	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewRoute53Provider(),
			},
		},
	}, nil
}

// buildDuckDNSIssuer builds a DuckDNS DNS challenge issuer.
func (b *TLSBuilder) buildDuckDNSIssuer() (*Issuer, error) {
	apiToken := b.getCredential("DUCKDNS_API_TOKEN")
	if apiToken == "" {
		b.logger.Warn("DUCKDNS_API_TOKEN not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewDuckDNSProvider(apiToken),
			},
		},
	}, nil
}

// buildDigitalOceanIssuer builds a DigitalOcean DNS challenge issuer.
func (b *TLSBuilder) buildDigitalOceanIssuer() (*Issuer, error) {
	authToken := b.getCredential("DO_AUTH_TOKEN")
	if authToken == "" {
		b.logger.Warn("DO_AUTH_TOKEN not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewDigitalOceanProvider(authToken),
			},
		},
	}, nil
}

// buildHetznerIssuer builds a Hetzner DNS challenge issuer.
func (b *TLSBuilder) buildHetznerIssuer() (*Issuer, error) {
	apiToken := b.getCredential("HETZNER_API_TOKEN")
	if apiToken == "" {
		b.logger.Warn("HETZNER_API_TOKEN not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewHetznerProvider(apiToken),
			},
		},
	}, nil
}

// buildPorkbunIssuer builds a Porkbun DNS challenge issuer.
func (b *TLSBuilder) buildPorkbunIssuer() (*Issuer, error) {
	apiKey := b.getCredential("PORKBUN_API_KEY")
	apiSecretKey := b.getCredential("PORKBUN_API_SECRET_KEY")
	if apiKey == "" || apiSecretKey == "" {
		b.logger.Warn("PORKBUN credentials not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewPorkbunProvider(apiKey, apiSecretKey),
			},
		},
	}, nil
}

// buildAzureIssuer builds an Azure DNS challenge issuer.
func (b *TLSBuilder) buildAzureIssuer() (*Issuer, error) {
	tenantID := b.getCredential("AZURE_TENANT_ID")
	clientID := b.getCredential("AZURE_CLIENT_ID")
	clientSecret := b.getCredential("AZURE_CLIENT_SECRET")
	subscriptionID := b.getCredential("AZURE_SUBSCRIPTION_ID")
	resourceGroup := b.getCredential("AZURE_RESOURCE_GROUP")

	if tenantID == "" || clientID == "" || clientSecret == "" || subscriptionID == "" || resourceGroup == "" {
		b.logger.Warn("Azure credentials not complete, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewAzureProvider(tenantID, clientID, clientSecret, subscriptionID, resourceGroup),
			},
		},
	}, nil
}

// buildVultrIssuer builds a Vultr DNS challenge issuer.
func (b *TLSBuilder) buildVultrIssuer() (*Issuer, error) {
	apiKey := b.getCredential("VULTR_API_KEY")
	if apiKey == "" {
		b.logger.Warn("VULTR_API_KEY not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewVultrProvider(apiKey),
			},
		},
	}, nil
}

// buildNamecheapIssuer builds a Namecheap DNS challenge issuer.
func (b *TLSBuilder) buildNamecheapIssuer() (*Issuer, error) {
	apiKey := b.getCredential("NAMECHEAP_API_KEY")
	user := b.getCredential("NAMECHEAP_API_USER")
	if apiKey == "" || user == "" {
		b.logger.Warn("Namecheap credentials not set, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewNamecheapProvider(apiKey, user),
			},
		},
	}, nil
}

// buildOVHIssuer builds an OVH DNS challenge issuer.
func (b *TLSBuilder) buildOVHIssuer() (*Issuer, error) {
	endpoint := b.getCredential("OVH_ENDPOINT")
	applicationKey := b.getCredential("OVH_APPLICATION_KEY")
	applicationSecret := b.getCredential("OVH_APPLICATION_SECRET")
	consumerKey := b.getCredential("OVH_CONSUMER_KEY")

	if endpoint == "" || applicationKey == "" || applicationSecret == "" || consumerKey == "" {
		b.logger.Warn("OVH credentials not complete, skipping DNS challenge")
		return nil, nil
	}

	return &Issuer{
		Module: "acme",
		Email:  b.settings.AdminEmail,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: NewOVHProvider(endpoint, applicationKey, applicationSecret, consumerKey),
			},
		},
	}, nil
}

// getCredential gets a credential from settings or environment.
func (b *TLSBuilder) getCredential(key string) string {
	// First check settings
	if b.settings != nil && b.settings.DNSCredentials != nil {
		if val, ok := b.settings.DNSCredentials[key]; ok && val != "" {
			return val
		}
	}

	// Fall back to environment variable
	return os.Getenv(key)
}
