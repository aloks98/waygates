const MESSAGES: Record<string, string> = {
  no_account: 'No Waygates account for this identity — contact an administrator.',
  disabled: 'Your account is disabled. Contact your administrator.',
  email_unverified: 'Your identity provider did not provide a verified email.',
  state_mismatch: 'Sign-in could not be verified. Please try again.',
  sso_disabled: 'Single sign-on is not enabled.',
  sso_failed: 'Single sign-on failed. Please try again or use a password.',
};

export function ssoErrorMessage(code: string): string {
  return MESSAGES[code] ?? MESSAGES.sso_failed;
}
