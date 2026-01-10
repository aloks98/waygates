import { Alert, AlertDescription, Card, CardContent, Separator, Skeleton } from '@e412/titanium';
import { useQuery } from '@tanstack/react-query';
import { useSearch } from '@tanstack/react-router';
import { AlertCircle, Lock, RefreshCw } from 'lucide-react';
import { ACLLoginForm, OAuthProvidersList } from '@/components/acl';
import { publicApi } from '@/lib/api';
import { sanitizeCSS } from '@/lib/css-sanitizer';
import type { ACLBranding, OAuthProvider } from '@/types/acl';
import type { ApiResponse } from '@/types/api';

// Default branding values
const defaultBranding: ACLBranding = {
  id: 0,
  title: 'Waygates',
  subtitle: 'Sign in to continue',
  primary_color: '#3b82f6',
  background_color: '', // Empty to use default theme background
  updated_at: '',
};

// Query keys for public APIs
const PUBLIC_BRANDING_KEY = ['public', 'acl-branding'] as const;
const PUBLIC_OAUTH_PROVIDERS_KEY = ['public', 'oauth-providers'] as const;

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

function usePublicOAuthProviders() {
  return useQuery({
    queryKey: PUBLIC_OAUTH_PROVIDERS_KEY,
    queryFn: async () => {
      const response = await publicApi
        .get('auth/oauth/providers')
        .json<ApiResponse<OAuthProvider[]>>();
      return response.data;
    },
    staleTime: 5 * 60 * 1000,
    retry: 1,
  });
}

// Loading skeleton component
function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="text-center space-y-2">
        <Skeleton className="h-12 w-12 rounded-full mx-auto" />
        <Skeleton className="h-8 w-48 mx-auto" />
        <Skeleton className="h-5 w-64 mx-auto" />
      </div>
      <Skeleton className="h-16 w-full rounded-lg" />
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
    <div className="flex items-center justify-center gap-2 rounded-lg border bg-muted/50 px-4 py-3">
      <Lock className="size-4 text-muted-foreground" />
      <div className="text-center">
        <p className="text-xs text-muted-foreground">Accessing</p>
        <p className="text-sm font-medium truncate max-w-[280px]">{host}</p>
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
        <span className="bg-background px-2 text-muted-foreground">{text}</span>
      </div>
    </div>
  );
}

// Main ACL Login Page content
function ACLLoginContent({
  branding,
  providers,
  redirectUrl,
  host,
}: {
  branding: ACLBranding;
  providers: OAuthProvider[];
  redirectUrl?: string;
  host?: string;
}) {
  const enabledProviders = providers.filter((p) => p.enabled);
  const hasOAuthProviders = enabledProviders.length > 0;

  return (
    <div className="space-y-6">
      {/* Header with logo and title */}
      <div className="text-center space-y-2">
        {branding.logo_url ? (
          <img
            src={branding.logo_url}
            alt={branding.title}
            className="h-12 w-auto mx-auto object-contain"
          />
        ) : (
          <div className="h-12 w-12 mx-auto rounded-full bg-primary/10 flex items-center justify-center">
            <Lock className="size-6 text-primary" />
          </div>
        )}
        <h1 className="text-2xl font-semibold tracking-tight">{branding.title}</h1>
        {branding.subtitle && <p className="text-sm text-muted-foreground">{branding.subtitle}</p>}
      </div>

      {/* Host badge */}
      {host && <HostBadge host={host} />}

      {/* Login form */}
      <ACLLoginForm redirectUrl={redirectUrl} primaryColor={branding.primary_color} />

      {/* OAuth providers */}
      {hasOAuthProviders && (
        <>
          <Divider text="or" />
          <OAuthProvidersList providers={providers} redirectUrl={redirectUrl} />
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

export function ACLLoginPage() {
  // Get query parameters
  const search = useSearch({ strict: false }) as ACLLoginSearchParams;
  const redirectUrl = search.redirect;
  const host = search.host;
  const errorParam = search.error;

  // Fetch branding and providers
  const {
    data: branding,
    isLoading: isBrandingLoading,
    isError: isBrandingError,
    error: brandingError,
    refetch: refetchBranding,
  } = usePublicBranding();

  const {
    data: providers,
    isLoading: isProvidersLoading,
    isError: isProvidersError,
  } = usePublicOAuthProviders();

  const isLoading = isBrandingLoading || isProvidersLoading;
  const effectiveBranding = branding || defaultBranding;
  const effectiveProviders = providers || [];

  // Apply custom CSS if provided (sanitized to prevent XSS and data exfiltration)
  const customStyles = effectiveBranding.custom_css
    ? { __html: sanitizeCSS(effectiveBranding.custom_css) }
    : null;

  // Background style from branding (only apply if explicitly set)
  const backgroundStyle = effectiveBranding.background_color
    ? { backgroundColor: effectiveBranding.background_color }
    : undefined;

  return (
    <>
      {/* Inject custom CSS - sanitized to prevent XSS and data exfiltration */}
      {/* biome-ignore lint/security/noDangerouslySetInnerHtml: CSS is sanitized via sanitizeCSS() */}
      {customStyles && <style dangerouslySetInnerHTML={customStyles} />}

      <div
        className="flex min-h-screen items-center justify-center bg-background px-4 py-8"
        style={backgroundStyle}
      >
        <Card className="w-full max-w-md shadow-lg">
          <CardContent className="pt-6">
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
                providers={effectiveProviders}
                redirectUrl={redirectUrl}
                host={host}
              />
            )}

            {/* Show warning if providers failed to load but branding is OK */}
            {!isLoading && !isBrandingError && isProvidersError && (
              <Alert variant="warning" className="mt-4">
                <AlertCircle className="size-4" />
                <AlertDescription>
                  Some sign-in options may be unavailable. Please try again later.
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      </div>
    </>
  );
}
