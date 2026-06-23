import { Button, Card, CardContent } from '@e412/rnui-react';
import { useSearch } from '@tanstack/react-router';
import { AlertCircle, ArrowLeft, Ban, LogOut, Mail, ShieldX } from 'lucide-react';

import { publicApi } from '@/lib/api';

// Search params type
interface ForbiddenSearchParams {
  reason?: string;
  email?: string;
  provider?: string;
  url?: string;
}

// Logout and clear session
async function handleLogout() {
  try {
    await publicApi.post('auth/acl/logout').json();
  } catch {
    // Ignore errors - cookie will be cleared anyway
  }
  // Redirect back to the original URL to trigger fresh login
  const search = new URLSearchParams(window.location.search);
  const originalUrl = search.get('url');
  if (originalUrl) {
    window.location.href = originalUrl;
  } else {
    window.location.href = '/';
  }
}

// Go back to previous page
function handleGoBack() {
  if (window.history.length > 1) {
    window.history.back();
  } else {
    window.location.href = '/';
  }
}

export function ACLForbiddenPage() {
  const search = useSearch({ strict: false }) as ForbiddenSearchParams;

  // Decode URL-encoded parameters
  const reason = search.reason ? decodeURIComponent(search.reason) : null;
  const email = search.email ? decodeURIComponent(search.email) : null;
  const provider = search.provider ? decodeURIComponent(search.provider) : null;
  const originalUrl = search.url ? decodeURIComponent(search.url) : null;

  // Determine if this is an OAuth authorization issue
  const isOAuthDenied = email && provider;

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-8">
      <Card className="w-full max-w-lg shadow-lg">
        <CardContent className="pt-8 pb-6">
          <div className="text-center space-y-6">
            {/* Icon */}
            <div className="mx-auto w-16 h-16 rounded-none bg-destructive/10 flex items-center justify-center">
              {isOAuthDenied ? (
                <ShieldX className="size-8 text-destructive" />
              ) : (
                <Ban className="size-8 text-destructive" />
              )}
            </div>

            {/* Title */}
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold tracking-tight text-foreground">
                Access Denied
              </h1>
              <p className="text-sm text-muted-foreground">
                {isOAuthDenied
                  ? "You're signed in, but your account is not authorized to access this resource."
                  : 'You do not have permission to access this resource.'}
              </p>
            </div>

            {/* User info badge (for OAuth users) */}
            {isOAuthDenied && (
              <div className="flex items-center justify-center gap-3 rounded-lg border bg-muted/50 px-4 py-3">
                <div className="flex items-center justify-center w-10 h-10 rounded-none bg-background border">
                  <Mail className="size-5 text-muted-foreground" />
                </div>
                <div className="text-left">
                  <p className="text-sm font-medium">{email}</p>
                  <p className="text-xs text-muted-foreground capitalize">
                    Signed in via {provider}
                  </p>
                </div>
              </div>
            )}

            {/* Reason box */}
            {reason && (
              <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-4 text-left">
                <div className="flex items-start gap-3">
                  <AlertCircle className="size-5 text-destructive mt-0.5 flex-shrink-0" />
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-destructive">Why was access denied?</p>
                    <p className="text-sm text-muted-foreground">{reason}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Original URL info */}
            {originalUrl && (
              <div className="text-xs text-muted-foreground">
                Attempted to access:{' '}
                <span className="font-mono bg-muted px-1.5 py-0.5 rounded">{originalUrl}</span>
              </div>
            )}

            {/* Actions */}
            <div className="flex flex-col sm:flex-row gap-3 pt-2">
              <Button variant="outline" onClick={handleGoBack} className="flex-1">
                <ArrowLeft className="size-4" />
                Go Back
              </Button>
              {isOAuthDenied && (
                <Button variant="default" onClick={handleLogout} className="flex-1">
                  <LogOut className="size-4" />
                  Sign in with different account
                </Button>
              )}
            </div>

            {/* Help text */}
            <p className="text-xs text-muted-foreground">
              If you believe you should have access, please contact your administrator.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
