import { useQuery } from '@tanstack/react-query';
import { useSearch } from '@tanstack/react-router';
import { AlertCircle, CheckCircle, Lock, RefreshCw } from 'lucide-react';
import { useEffect } from 'react';

import { ACLLoginForm, OAuthProvidersList } from '@/components/acl';
import { WaygatesBrand } from '@/components/layout/waygate-logo';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card, CardContent } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { publicApi } from '@/lib/api';
import { sanitizeCSS } from '@/lib/css-sanitizer';
import type { ACLBranding, AuthOptionsResponse, OAuthProvider } from '@/types/acl';
import type { ApiResponse } from '@/types/api';

// Default branding values
const defaultBranding: ACLBranding = {
  id: 0,
  title: 'Waygates',
  subtitle: 'Sign in to continue',
  primary_color: '',
  background_color: '', // Empty to use default theme background
  updated_at: '',
};

// Query keys for public APIs
const PUBLIC_BRANDING_KEY = ['public', 'acl-branding'] as const;
const AUTH_OPTIONS_KEY = ['public', 'auth-options'] as const;

// Custom hooks for public API endpoints (no auth required)
function usePublicBranding() {
  return useQuery({
    queryKey: PUBLIC_BRANDING_KEY,
    queryFn: async () => {
      const response = await publicApi.get('acl/branding').json<ApiResponse<ACLBranding>>();
      return response.data;
    },
    staleTime: 5 * 60 * 1000, // Branding doesn't change often
    retry: 1,
  });
}

// Fetch auth options for a specific hostname (union of all ACL groups)
function useAuthOptions(hostname?: string) {
  return useQuery({
    queryKey: [...AUTH_OPTIONS_KEY, hostname],
    queryFn: async () => {
      if (!hostname) return null;
      const response = await publicApi
        .get('acl/options', { searchParams: { hostname } })
        .json<ApiResponse<AuthOptionsResponse>>();
      return response.data;
    },
    staleTime: 5 * 60 * 1000,
    retry: 1,
    enabled: !!hostname,
  });
}

// Loading skeleton component
function LoadingSkeleton() {
  return (
    <div className="flex flex-col gap-5">
      <div className="text-center space-y-2">
        <Skeleton className="h-12 w-12 rounded mx-auto" />
        <Skeleton className="h-8 w-48 mx-auto" />
        <Skeleton className="h-5 w-64 mx-auto" />
      </div>
      <Skeleton className="h-16 w-full rounded" />
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    </div>
  );
}

// Error state component
function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <AlertCircle className="size-12 text-destructive/50" />
      <h3 className="mt-4 text-lg font-medium">Something went wrong</h3>
      <p className="mt-2 text-sm text-muted-foreground max-w-sm">{message}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="mt-4 inline-flex items-center gap-2 text-sm text-primary hover:underline"
        >
          <RefreshCw className="size-4" />
          Try again
        </button>
      )}
    </div>
  );
}

// Host display component
function HostBadge({ host }: { host: string }) {
  return (
    <div className="flex items-center gap-2 rounded-md bg-muted px-3 py-3">
      <Lock className="size-4 text-muted-foreground" />
      <div className="text-center">
        <p className="text-sm font-mono break-all">{host}</p>
      </div>
    </div>
  );
}

// Divider with text
function Divider({ text }: { text: string }) {
  return (
    <div className="relative">
      <div className="absolute inset-0 flex items-center">
        <Separator className="w-full" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-card px-2 text-muted-foreground">{text}</span>
      </div>
    </div>
  );
}

// Main ACL Login Page content
function ACLLoginContent({
  branding,
  authOptions,
  redirectUrl,
  host,
}: {
  branding: ACLBranding;
  authOptions: AuthOptionsResponse | null;
  redirectUrl?: string;
  host?: string;
}) {
  // Check what auth methods are available from auth options
  // If no authOptions (e.g., no host param), default to showing Waygates form
  const showWaygatesAuth = authOptions?.waygates_auth?.enabled ?? !authOptions;
  const hasBasicAuth = authOptions?.basic_auth_enabled ?? false;
  const oauthProviders: OAuthProvider[] = (authOptions?.oauth_providers ?? []).map((p) => ({
    id: p.id,
    name: p.name,
    available: true,
    enabled: p.enabled,
  }));
  // Only consider OAuth providers that are actually enabled
  const hasOAuthProviders = oauthProviders.some((p) => p.enabled);

  // Check if auth is required (default to true if no authOptions yet)
  const requiresAuth = authOptions?.requires_auth ?? true;

  return (
    <div className="flex flex-col gap-5">
      {/* Header - only show login header when auth is required */}
      {requiresAuth ? (
        <div className="flex flex-col gap-2">
          <h1 className="text-2xl font-semibold tracking-tight">
            {branding.title && branding.title !== 'Waygates'
              ? branding.title
              : 'Sign in to continue'}
          </h1>
          {branding.subtitle && branding.subtitle !== 'Sign in to continue' && (
            <p className="text-sm text-muted-foreground">{branding.subtitle}</p>
          )}
        </div>
      ) : (
        /* Show success state when no auth required */
        <div className="flex flex-col items-center py-4 text-center">
          <div className="rounded bg-green-500/10 p-4 mb-4">
            <CheckCircle className="size-8 text-green-500" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">No Authentication Required</h1>
          <p className="text-sm text-muted-foreground max-w-xs mt-2">
            {redirectUrl
              ? 'This resource is publicly accessible. Redirecting you now...'
              : 'This resource is publicly accessible. You can access it directly.'}
          </p>
        </div>
      )}

      {/* Host badge - always show if host is available */}
      {host && <HostBadge host={host} />}

      {/* Auth methods - only show when auth is required */}
      {requiresAuth && (
        <>
          {/* Basic auth info - show when only basic auth is available */}
          {hasBasicAuth && !showWaygatesAuth && !hasOAuthProviders && (
            <Alert>
              <Lock className="size-4" />
              <AlertDescription>
                This resource uses HTTP Basic Authentication. Your browser will prompt you for
                credentials when you access the protected resource.
              </AlertDescription>
            </Alert>
          )}

          {/* Waygates login form */}
          {showWaygatesAuth && (
            <ACLLoginForm redirectUrl={redirectUrl} primaryColor={branding.primary_color} />
          )}

          {/* OAuth providers */}
          {hasOAuthProviders && (
            <>
              {showWaygatesAuth && <Divider text="or" />}
              <OAuthProvidersList providers={oauthProviders} redirectUrl={redirectUrl} />
            </>
          )}
        </>
      )}

      {/* Footer */}
      {branding.footer_text && (
        <p className="text-center text-xs text-muted-foreground">{branding.footer_text}</p>
      )}
    </div>
  );
}

// Search params type
interface ACLLoginSearchParams {
  redirect?: string;
  host?: string;
  error?: string;
}

// Extract hostname from a URL string
function extractHostname(url?: string): string | undefined {
  if (!url) return undefined;
  try {
    // Handle both full URLs (https://example.com/path) and partial (example.com/path)
    const urlToParse = url.startsWith('http') ? url : `https://${url}`;
    const parsed = new URL(urlToParse);
    return parsed.hostname;
  } catch {
    return undefined;
  }
}

export function ACLLoginPage() {
  // Get query parameters
  const search = useSearch({ strict: false }) as ACLLoginSearchParams;
  const redirectUrl = search.redirect;
  const errorParam = search.error;

  // Extract hostname from redirect URL or use explicit host param
  const host = search.host || extractHostname(redirectUrl);

  // Fetch branding and auth options for the hostname
  const {
    data: branding,
    isLoading: isBrandingLoading,
    isError: isBrandingError,
    error: brandingError,
    refetch: refetchBranding,
  } = usePublicBranding();

  const {
    data: authOptions,
    isLoading: isAuthOptionsLoading,
    isError: isAuthOptionsError,
  } = useAuthOptions(host);

  // Redirect if no auth is required
  useEffect(() => {
    if (authOptions && !authOptions.requires_auth && redirectUrl) {
      // Small delay to show feedback before redirect
      const timer = setTimeout(() => {
        window.location.href = redirectUrl;
      }, 1500);
      return () => clearTimeout(timer);
    }
  }, [authOptions, redirectUrl]);

  const isLoading = isBrandingLoading || isAuthOptionsLoading;
  const effectiveBranding = branding || defaultBranding;

  // Apply custom CSS if provided (sanitized to prevent XSS and data exfiltration)
  const customStyles = effectiveBranding.custom_css
    ? { __html: sanitizeCSS(effectiveBranding.custom_css) }
    : null;

  // Background style from branding — applied directly when set by admin
  const backgroundStyle = effectiveBranding.background_color
    ? { backgroundColor: effectiveBranding.background_color }
    : undefined;

  return (
    <>
      {/* Inject custom CSS - sanitized to prevent XSS and data exfiltration */}
      {/* biome-ignore lint/security/noDangerouslySetInnerHtml: CSS is sanitized via sanitizeCSS() */}
      {customStyles && <style dangerouslySetInnerHTML={customStyles} />}

      <div
        className="flex min-h-screen flex-col items-center justify-center gap-7 bg-background px-4 py-8"
        style={backgroundStyle}
      >
        {effectiveBranding.logo_url ? (
          <img
            src={effectiveBranding.logo_url}
            alt={effectiveBranding.title}
            className="h-9 max-w-48 object-contain"
          />
        ) : (
          <WaygatesBrand />
        )}
        <Card className="w-full max-w-md rounded-xl [--card-spacing:28px]">
          <CardContent>
            {/* Show error from URL param (e.g., OAuth failure) */}
            {errorParam && (
              <Alert variant="destructive" className="mb-6">
                <AlertCircle className="size-4" />
                <AlertDescription>{decodeURIComponent(errorParam)}</AlertDescription>
              </Alert>
            )}

            {isLoading ? (
              <LoadingSkeleton />
            ) : isBrandingError ? (
              <ErrorState
                message={brandingError?.message || 'Failed to load login page'}
                onRetry={() => refetchBranding()}
              />
            ) : (
              <ACLLoginContent
                branding={effectiveBranding}
                authOptions={authOptions ?? null}
                redirectUrl={redirectUrl}
                host={host}
              />
            )}

            {/* Show warning if auth options failed to load but branding is OK */}
            {!isLoading && !isBrandingError && isAuthOptionsError && (
              <Alert variant="warning" className="mt-4">
                <AlertCircle className="size-4" />
                <AlertDescription>
                  Could not load authentication options. Some sign-in methods may be unavailable.
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
