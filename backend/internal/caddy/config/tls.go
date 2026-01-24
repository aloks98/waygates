// Package config provides typed Go structs for generating Caddy JSON configuration.
package config

// TLSAppFull provides the complete TLS app configuration.
// This extends the basic TLSApp in types.go with all DNS provider configs.
// See: https://caddyserver.com/docs/json/apps/tls/

// DNSProviderCloudflare configures Cloudflare DNS-01 challenge.
type DNSProviderCloudflare struct {
	Name     string `json:"name"` // Must be "cloudflare"
	APIToken string `json:"api_token"`
}

// DNSProviderRoute53 configures AWS Route53 DNS-01 challenge.
// Uses AWS SDK credential chain (env vars, IAM roles, etc.)
type DNSProviderRoute53 struct {
	Name string `json:"name"` // Must be "route53"
	// Credentials are loaded via AWS SDK - no explicit config needed
	Region          string `json:"region,omitempty"`
	HostedZoneID    string `json:"hosted_zone_id,omitempty"`
	MaxRetries      int    `json:"max_retries,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

// DNSProviderDuckDNS configures DuckDNS DNS-01 challenge.
type DNSProviderDuckDNS struct {
	Name     string `json:"name"` // Must be "duckdns"
	APIToken string `json:"api_token"`
}

// DNSProviderDigitalOcean configures DigitalOcean DNS-01 challenge.
type DNSProviderDigitalOcean struct {
	Name      string `json:"name"` // Must be "digitalocean"
	AuthToken string `json:"auth_token"`
}

// DNSProviderHetzner configures Hetzner DNS-01 challenge.
type DNSProviderHetzner struct {
	Name     string `json:"name"` // Must be "hetzner"
	APIToken string `json:"api_token"`
}

// DNSProviderPorkbun configures Porkbun DNS-01 challenge.
type DNSProviderPorkbun struct {
	Name         string `json:"name"` // Must be "porkbun"
	APIKey       string `json:"api_key"`
	APISecretKey string `json:"api_secret_key"`
}

// DNSProviderAzure configures Azure DNS-01 challenge.
type DNSProviderAzure struct {
	Name              string `json:"name"` // Must be "azure"
	TenantID          string `json:"tenant_id"`
	ClientID          string `json:"client_id"`
	ClientSecret      string `json:"client_secret"`
	SubscriptionID    string `json:"subscription_id"`
	ResourceGroupName string `json:"resource_group_name"`
}

// DNSProviderVultr configures Vultr DNS-01 challenge.
type DNSProviderVultr struct {
	Name   string `json:"name"` // Must be "vultr"
	APIKey string `json:"api_key"`
}

// DNSProviderNamecheap configures Namecheap DNS-01 challenge.
type DNSProviderNamecheap struct {
	Name   string `json:"name"` // Must be "namecheap"
	APIKey string `json:"api_key"`
	User   string `json:"user"`
}

// DNSProviderOVH configures OVH DNS-01 challenge.
type DNSProviderOVH struct {
	Name              string `json:"name"` // Must be "ovh"
	Endpoint          string `json:"endpoint"`
	ApplicationKey    string `json:"application_key"`
	ApplicationSecret string `json:"application_secret"`
	ConsumerKey       string `json:"consumer_key"`
}

// ACME provider name constants.
const (
	ACMEProviderOff          = "off"
	ACMEProviderHTTP         = "http"
	ACMEProviderCloudflare   = "cloudflare"
	ACMEProviderRoute53      = "route53"
	ACMEProviderDuckDNS      = "duckdns"
	ACMEProviderDigitalOcean = "digitalocean"
	ACMEProviderHetzner      = "hetzner"
	ACMEProviderPorkbun      = "porkbun"
	ACMEProviderAzure        = "azure"
	ACMEProviderVultr        = "vultr"
	ACMEProviderNamecheap    = "namecheap"
	ACMEProviderOVH          = "ovh"
)

// NewTLSApp creates a basic TLS app configuration with automation for specified domains.
func NewTLSApp(domains []string) *TLSApp {
	if len(domains) == 0 {
		return nil
	}

	return &TLSApp{
		Certificates: &CertificatesConfig{
			Automate: domains,
		},
	}
}

// NewTLSAppWithACME creates a TLS app with ACME automation policy.
func NewTLSAppWithACME(domains []string, issuer *Issuer) *TLSApp {
	if len(domains) == 0 {
		return nil
	}

	app := &TLSApp{
		Certificates: &CertificatesConfig{
			Automate: domains,
		},
	}

	if issuer != nil {
		app.Automation = &AutomationConfig{
			Policies: []*AutomationPolicy{
				{
					Subjects: domains,
					Issuers:  []*Issuer{issuer},
				},
			},
		}
	}

	return app
}

// NewACMEIssuer creates an ACME issuer configuration.
func NewACMEIssuer(email string) *Issuer {
	return &Issuer{
		Module: "acme",
		Email:  email,
	}
}

// NewACMEIssuerWithDNS creates an ACME issuer with DNS-01 challenge.
func NewACMEIssuerWithDNS(email string, provider interface{}) *Issuer {
	return &Issuer{
		Module: "acme",
		Email:  email,
		Challenges: &ChallengesConfig{
			DNS: &DNSChallengeConfig{
				Provider: provider,
			},
		},
	}
}

// NewCloudflareProvider creates a Cloudflare DNS provider configuration.
func NewCloudflareProvider(apiToken string) *DNSProviderCloudflare {
	return &DNSProviderCloudflare{
		Name:     "cloudflare",
		APIToken: apiToken,
	}
}

// NewRoute53Provider creates a Route53 DNS provider configuration.
func NewRoute53Provider() *DNSProviderRoute53 {
	return &DNSProviderRoute53{
		Name: "route53",
	}
}

// NewRoute53ProviderWithCredentials creates a Route53 provider with explicit credentials.
func NewRoute53ProviderWithCredentials(accessKeyID, secretAccessKey, region string) *DNSProviderRoute53 {
	return &DNSProviderRoute53{
		Name:            "route53",
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}
}

// NewDuckDNSProvider creates a DuckDNS DNS provider configuration.
func NewDuckDNSProvider(apiToken string) *DNSProviderDuckDNS {
	return &DNSProviderDuckDNS{
		Name:     "duckdns",
		APIToken: apiToken,
	}
}

// NewDigitalOceanProvider creates a DigitalOcean DNS provider configuration.
func NewDigitalOceanProvider(authToken string) *DNSProviderDigitalOcean {
	return &DNSProviderDigitalOcean{
		Name:      "digitalocean",
		AuthToken: authToken,
	}
}

// NewHetznerProvider creates a Hetzner DNS provider configuration.
func NewHetznerProvider(apiToken string) *DNSProviderHetzner {
	return &DNSProviderHetzner{
		Name:     "hetzner",
		APIToken: apiToken,
	}
}

// NewPorkbunProvider creates a Porkbun DNS provider configuration.
func NewPorkbunProvider(apiKey, apiSecretKey string) *DNSProviderPorkbun {
	return &DNSProviderPorkbun{
		Name:         "porkbun",
		APIKey:       apiKey,
		APISecretKey: apiSecretKey,
	}
}

// NewAzureProvider creates an Azure DNS provider configuration.
func NewAzureProvider(tenantID, clientID, clientSecret, subscriptionID, resourceGroupName string) *DNSProviderAzure {
	return &DNSProviderAzure{
		Name:              "azure",
		TenantID:          tenantID,
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		SubscriptionID:    subscriptionID,
		ResourceGroupName: resourceGroupName,
	}
}

// NewVultrProvider creates a Vultr DNS provider configuration.
func NewVultrProvider(apiKey string) *DNSProviderVultr {
	return &DNSProviderVultr{
		Name:   "vultr",
		APIKey: apiKey,
	}
}

// NewNamecheapProvider creates a Namecheap DNS provider configuration.
func NewNamecheapProvider(apiKey, user string) *DNSProviderNamecheap {
	return &DNSProviderNamecheap{
		Name:   "namecheap",
		APIKey: apiKey,
		User:   user,
	}
}

// NewOVHProvider creates an OVH DNS provider configuration.
func NewOVHProvider(endpoint, applicationKey, applicationSecret, consumerKey string) *DNSProviderOVH {
	return &DNSProviderOVH{
		Name:              "ovh",
		Endpoint:          endpoint,
		ApplicationKey:    applicationKey,
		ApplicationSecret: applicationSecret,
		ConsumerKey:       consumerKey,
	}
}
