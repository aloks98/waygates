import { describe, expect, it } from 'vitest';

import { PROXY_SYSTEM_DEFAULTS } from './inheritable-switch';

// Drift guard: PROXY_SYSTEM_DEFAULTS is a hand-maintained copy of the
// canonical system defaults in backend/internal/proxygroup/resolve.go
// (DefaultSSLEnabled, DefaultSSLForced, DefaultBlockExploits,
// DefaultTLSInsecureSkipVerify). Nothing wires these two together at
// compile time, so this test pins the expected literal values here — if a
// future change updates the Go constants without updating this file, this
// is the test that must be touched (and will fail if it isn't).
describe('PROXY_SYSTEM_DEFAULTS', () => {
  it('matches proxygroup.Default* in backend/internal/proxygroup/resolve.go', () => {
    expect(PROXY_SYSTEM_DEFAULTS).toEqual({
      ssl_enabled: true,
      ssl_forced: true,
      block_exploits: true,
      tls_insecure_skip_verify: false,
    });
  });
});
