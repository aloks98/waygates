package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dnsProviderCredentials lists the credentials each DNS-01 provider needs before
// it will emit an issuer at all.
var dnsProviderCredentials = map[string]map[string]string{
	ACMEProviderCloudflare: {"CLOUDFLARE_API_TOKEN": "token"},
	ACMEProviderRoute53:    {},
	ACMEProviderDuckDNS:    {"DUCKDNS_API_TOKEN": "token"},
	ACMEProviderDigitalOcean: {
		"DO_AUTH_TOKEN": "token",
	},
	ACMEProviderHetzner: {"HETZNER_API_TOKEN": "token"},
	ACMEProviderPorkbun: {
		"PORKBUN_API_KEY":        "key",
		"PORKBUN_API_SECRET_KEY": "secret",
	},
	ACMEProviderAzure: {
		"AZURE_TENANT_ID":       "tenant",
		"AZURE_CLIENT_ID":       "client",
		"AZURE_CLIENT_SECRET":   "secret",
		"AZURE_SUBSCRIPTION_ID": "sub",
		"AZURE_RESOURCE_GROUP":  "rg",
	},
	ACMEProviderVultr: {"VULTR_API_KEY": "key"},
	ACMEProviderNamecheap: {
		"NAMECHEAP_API_KEY":  "key",
		"NAMECHEAP_API_USER": "user",
	},
	ACMEProviderOVH: {
		"OVH_ENDPOINT":           "ovh-eu",
		"OVH_APPLICATION_KEY":    "key",
		"OVH_APPLICATION_SECRET": "secret",
		"OVH_CONSUMER_KEY":       "consumer",
	},
}

func builderFor(provider string, resolvers []string) *TLSBuilder {
	return NewTLSBuilder(nil).SetSettings(&Settings{
		AdminEmail:     "admin@example.com",
		ACMEProvider:   provider,
		ACMEResolvers:  resolvers,
		DNSCredentials: dnsProviderCredentials[provider],
	})
}

// Every DNS-01 provider must set resolvers. The propagation pre-check queries the
// zone's authoritative nameservers; when the host resolver is authoritative for
// the zone internally (split-horizon) it answers with a nameserver name that does
// not resolve, and the challenge dies before validation is ever requested.
func TestTLSBuilder_AllDNSProvidersEmitResolvers(t *testing.T) {
	for provider := range dnsProviderCredentials {
		t.Run(provider, func(t *testing.T) {
			issuer, err := builderFor(provider, nil).buildIssuer()
			require.NoError(t, err)
			require.NotNil(t, issuer, "provider should produce an issuer")
			require.NotNil(t, issuer.Challenges)
			require.NotNil(t, issuer.Challenges.DNS)

			assert.Equal(t, DefaultACMEResolvers, issuer.Challenges.DNS.Resolvers)
			assert.NotNil(t, issuer.Challenges.DNS.Provider)
		})
	}
}

func TestTLSBuilder_ConfiguredResolversOverrideDefault(t *testing.T) {
	issuer, err := builderFor(ACMEProviderCloudflare, []string{"9.9.9.9", "149.112.112.112"}).buildIssuer()
	require.NoError(t, err)
	require.NotNil(t, issuer)

	assert.Equal(t, []string{"9.9.9.9", "149.112.112.112"}, issuer.Challenges.DNS.Resolvers)
}

func TestTLSBuilder_SystemSentinelRestoresHostResolver(t *testing.T) {
	issuer, err := builderFor(ACMEProviderCloudflare, []string{ACMEResolversSystem}).buildIssuer()
	require.NoError(t, err)
	require.NotNil(t, issuer)

	assert.Nil(t, issuer.Challenges.DNS.Resolvers, "sentinel should fall back to Caddy's default")

	// The key omitted entirely, not serialized as null or [].
	encoded, err := json.Marshal(issuer)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "resolvers")
}

func TestTLSBuilder_DefaultResolversSerializeIntoChallengeJSON(t *testing.T) {
	issuer, err := builderFor(ACMEProviderCloudflare, nil).buildIssuer()
	require.NoError(t, err)

	encoded, err := json.Marshal(issuer)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"resolvers":["1.1.1.1","8.8.8.8"]`)
}

// The HTTP-01 challenge has no propagation check, so resolvers are meaningless.
func TestTLSBuilder_HTTPChallengeHasNoDNSConfig(t *testing.T) {
	issuer, err := builderFor(ACMEProviderHTTP, nil).buildIssuer()
	require.NoError(t, err)
	require.NotNil(t, issuer)

	assert.Nil(t, issuer.Challenges)
}

// Callers must not be able to mutate the package-level default through a built issuer.
func TestTLSBuilder_DefaultResolversNotAliased(t *testing.T) {
	issuer, err := builderFor(ACMEProviderCloudflare, nil).buildIssuer()
	require.NoError(t, err)

	issuer.Challenges.DNS.Resolvers[0] = "mutated"

	assert.Equal(t, "1.1.1.1", DefaultACMEResolvers[0], "package default must not be aliased")
}
