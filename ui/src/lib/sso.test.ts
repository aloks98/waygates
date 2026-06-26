import { expect, test } from 'vitest';

import { ssoErrorMessage } from './sso';

test('ssoErrorMessage maps known codes', () => {
  expect(ssoErrorMessage('no_account')).toMatch(/account/i);
  expect(ssoErrorMessage('disabled')).toMatch(/disabled/i);
  expect(ssoErrorMessage('email_unverified')).toMatch(/verified/i);
  expect(ssoErrorMessage('state_mismatch')).toMatch(/verified|try again/i);
  expect(ssoErrorMessage('sso_disabled')).toMatch(/not enabled/i);
  expect(ssoErrorMessage('sso_failed')).toMatch(/single sign-on/i);
});

test('ssoErrorMessage falls back for unknown codes', () => {
  expect(ssoErrorMessage('whatever')).toMatch(/single sign-on/i);
});
