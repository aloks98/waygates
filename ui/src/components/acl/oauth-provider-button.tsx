import { Button } from '@e412/titanium';
import type { ReactNode } from 'react';
import type { OAuthProvider } from '@/types/acl';

// Provider icons as inline SVGs for better control
// Using aria-hidden since these are decorative icons next to text labels
const providerIcons: Record<string, ReactNode> = {
  google: (
    <svg className="size-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
        fill="#4285F4"
      />
      <path
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
        fill="#34A853"
      />
      <path
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
        fill="#FBBC05"
      />
      <path
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
        fill="#EA4335"
      />
    </svg>
  ),
  github: (
    <svg className="size-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0112 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z" />
    </svg>
  ),
  microsoft: (
    <svg className="size-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M11.4 24H0V12.6h11.4V24z" fill="#00A4EF" />
      <path d="M24 24H12.6V12.6H24V24z" fill="#FFB900" />
      <path d="M11.4 11.4H0V0h11.4v11.4z" fill="#F25022" />
      <path d="M24 11.4H12.6V0H24v11.4z" fill="#7FBA00" />
    </svg>
  ),
  gitlab: (
    <svg className="size-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path
        d="M23.955 13.587l-1.342-4.135-2.664-8.189a.455.455 0 00-.867 0L16.418 9.45H7.582L4.918 1.263a.455.455 0 00-.867 0L1.386 9.452.044 13.587a.924.924 0 00.331 1.023L12 23.054l11.625-8.443a.92.92 0 00.33-1.024"
        fill="#FC6D26"
      />
      <path d="M12 23.054L7.582 9.452h8.836L12 23.054z" fill="#E24329" />
      <path d="M12 23.054l-4.418-13.6H1.386L12 23.053z" fill="#FC6D26" />
      <path
        d="M1.386 9.452L.044 13.587a.924.924 0 00.331 1.023L12 23.054 1.386 9.452z"
        fill="#FCA326"
      />
      <path d="M1.386 9.452h6.196L4.918 1.263a.455.455 0 00-.867 0L1.386 9.452z" fill="#E24329" />
      <path d="M12 23.054l4.418-13.6h6.196L12 23.053z" fill="#FC6D26" />
      <path
        d="M22.614 9.452l1.342 4.135a.924.924 0 01-.331 1.023L12 23.054l10.614-13.602z"
        fill="#FCA326"
      />
      <path d="M22.614 9.452h-6.196l2.664-8.189a.455.455 0 01.867 0l2.665 8.189z" fill="#E24329" />
    </svg>
  ),
  okta: (
    <svg className="size-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path
        d="M12 0C5.389 0 0 5.389 0 12s5.389 12 12 12 12-5.389 12-12S18.611 0 12 0zm0 18c-3.314 0-6-2.686-6-6s2.686-6 6-6 6 2.686 6 6-2.686 6-6 6z"
        fill="#007DC1"
      />
    </svg>
  ),
  auth0: (
    <svg className="size-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path
        d="M21.98 7.448L19.62 0H4.347L2.02 7.448c-1.352 4.312.03 9.206 3.815 12.015L12.007 24l6.157-4.552c3.755-2.81 5.182-7.688 3.815-12.015l-6.16 4.58 2.343 7.45-6.157-4.597-6.158 4.58 2.358-7.433-6.188-4.55 7.63-.045L12.008 0l2.356 7.404 7.615.044z"
        fill="#EB5424"
      />
    </svg>
  ),
};

// Default icon for unknown providers
const defaultIcon = (
  <svg
    className="size-5"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="10" />
    <path d="M12 16v-4M12 8h.01" />
  </svg>
);

interface OAuthProviderButtonProps {
  provider: OAuthProvider;
  redirectUrl?: string;
  className?: string;
}

export function OAuthProviderButton({
  provider,
  redirectUrl,
  className,
}: OAuthProviderButtonProps) {
  const handleClick = () => {
    const params = new URLSearchParams();
    if (redirectUrl) {
      params.set('redirect', redirectUrl);
    }
    const queryString = params.toString();
    window.location.href = `/auth/oauth/${provider.id}${queryString ? `?${queryString}` : ''}`;
  };

  const icon = providerIcons[provider.id.toLowerCase()] || defaultIcon;

  return (
    <Button
      type="button"
      variant="outline"
      className={`w-full justify-center gap-3 ${className || ''}`}
      onClick={handleClick}
    >
      {icon}
      <span>Continue with {provider.name}</span>
    </Button>
  );
}

interface OAuthProvidersListProps {
  providers: OAuthProvider[];
  redirectUrl?: string;
}

export function OAuthProvidersList({ providers, redirectUrl }: OAuthProvidersListProps) {
  const enabledProviders = providers.filter((p) => p.enabled);

  if (enabledProviders.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      {enabledProviders.map((provider) => (
        <OAuthProviderButton key={provider.id} provider={provider} redirectUrl={redirectUrl} />
      ))}
    </div>
  );
}
