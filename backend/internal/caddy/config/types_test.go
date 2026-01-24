package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaddyConfig_ToJSON(t *testing.T) {
	tests := []struct {
		name     string
		config   *CaddyConfig
		wantKeys []string
	}{
		{
			name:     "empty config",
			config:   &CaddyConfig{},
			wantKeys: []string{},
		},
		{
			name: "admin only",
			config: &CaddyConfig{
				Admin: &AdminConfig{
					Listen: "localhost:2019",
				},
			},
			wantKeys: []string{"admin"},
		},
		{
			name: "admin disabled",
			config: &CaddyConfig{
				Admin: &AdminConfig{
					Disabled: true,
				},
			},
			wantKeys: []string{"admin"},
		},
		{
			name: "with storage",
			config: &CaddyConfig{
				Storage: &StorageConfig{
					Module: "file_system",
					Root:   "/data/caddy",
				},
			},
			wantKeys: []string{"storage"},
		},
		{
			name: "full config structure",
			config: &CaddyConfig{
				Admin: &AdminConfig{
					Listen: "localhost:2019",
				},
				Storage: &StorageConfig{
					Module: "file_system",
					Root:   "/data/caddy",
				},
				Apps: &AppsConfig{},
			},
			wantKeys: []string{"admin", "storage", "apps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := tt.config.ToJSON()
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(jsonBytes, &result)
			require.NoError(t, err)

			for _, key := range tt.wantKeys {
				assert.Contains(t, result, key, "expected key %s in JSON output", key)
			}
		})
	}
}

func TestCaddyConfig_ToCompactJSON(t *testing.T) {
	config := &CaddyConfig{
		Admin: &AdminConfig{
			Listen: "localhost:2019",
		},
	}

	jsonBytes, err := config.ToCompactJSON()
	require.NoError(t, err)

	// Compact JSON should not have newlines or extra spaces
	assert.NotContains(t, string(jsonBytes), "\n")
	assert.Contains(t, string(jsonBytes), `"admin"`)
	assert.Contains(t, string(jsonBytes), `"listen":"localhost:2019"`)
}

func TestCaddyConfig_OmitEmpty(t *testing.T) {
	// Test that nil fields are omitted
	config := &CaddyConfig{
		Admin: &AdminConfig{
			Listen: "localhost:2019",
		},
		// Storage, Logging, Apps are nil
	}

	jsonBytes, err := config.ToJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "admin")
	assert.NotContains(t, result, "storage")
	assert.NotContains(t, result, "logging")
	assert.NotContains(t, result, "apps")
}

func TestAdminConfig_JSON(t *testing.T) {
	tests := []struct {
		name  string
		admin *AdminConfig
		want  string
	}{
		{
			name: "with listen",
			admin: &AdminConfig{
				Listen: "localhost:2019",
			},
			want: `{"listen":"localhost:2019"}`,
		},
		{
			name: "disabled",
			admin: &AdminConfig{
				Disabled: true,
			},
			want: `{"disabled":true}`,
		},
		{
			name: "disabled with listen",
			admin: &AdminConfig{
				Disabled: true,
				Listen:   "localhost:2019",
			},
			want: `{"disabled":true,"listen":"localhost:2019"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.admin)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(jsonBytes))
		})
	}
}

func TestStorageConfig_JSON(t *testing.T) {
	storage := &StorageConfig{
		Module: "file_system",
		Root:   "/data/caddy",
	}

	jsonBytes, err := json.Marshal(storage)
	require.NoError(t, err)

	want := `{"module":"file_system","root":"/data/caddy"}`
	assert.JSONEq(t, want, string(jsonBytes))
}

func TestLoggingConfig_JSON(t *testing.T) {
	logging := &LoggingConfig{
		Logs: map[string]*LogConfig{
			"default": {
				Writer: &LogWriter{
					Output:   "file",
					Filename: "/var/log/caddy/access.log",
				},
				Level: "INFO",
			},
		},
	}

	jsonBytes, err := json.Marshal(logging)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "logs")
}

func TestAppsConfig_JSON(t *testing.T) {
	apps := &AppsConfig{
		// All nil - should marshal to empty object
	}

	jsonBytes, err := json.Marshal(apps)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(jsonBytes))
}

func TestL4Server_JSON(t *testing.T) {
	// Placeholder test for L4 structs
	server := &L4Server{
		Listen: []string{"tcp/:5432"},
	}

	jsonBytes, err := json.Marshal(server)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "listen")
}
