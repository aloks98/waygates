import { useSearch } from '@tanstack/react-router';
import { Lock } from 'lucide-react';

import { WaygatesBrand } from '@/components/layout/waygate-logo';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
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

  let host = originalUrl;
  try {
    if (originalUrl) host = new URL(originalUrl).hostname;
  } catch {
    /* Display the provided resource when it is not a URL. */
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-7 bg-background px-4 py-8">
      <WaygatesBrand />
      <Card className="w-full max-w-md rounded-xl [--card-spacing:28px]">
        <CardContent className="flex flex-col gap-5">
          <div className="flex flex-col gap-2">
            <h1 className="text-2xl font-semibold">Access denied</h1>
            <p className="text-sm text-muted-foreground">
              This account doesn’t have access to the app.
            </p>
          </div>
          {host && (
            <div className="flex items-center gap-2 rounded-md bg-muted p-3">
              <Lock className="size-4 shrink-0 text-muted-foreground" />
              <span className="break-all font-mono text-sm">{host}</span>
            </div>
          )}
          {email && (
            <p className="text-sm text-muted-foreground">
              <span className="font-mono">{email}</span>
              {provider && ` · ${provider}`}
            </p>
          )}
          {reason && <p className="text-sm text-muted-foreground">{reason}</p>}
          <div className="flex flex-col gap-2">
            <Button onClick={handleLogout}>Use another account</Button>
            <Button variant="ghost" onClick={handleGoBack}>
              Go back
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
