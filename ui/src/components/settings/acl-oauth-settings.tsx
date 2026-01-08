import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  Skeleton,
  Switch,
} from '@e412/titanium';
import { AlertCircle, CheckCircle2, Github, KeyRound, XCircle } from 'lucide-react';
import { useAdminOAuthProviders, useUpdateOAuthProvider } from '@/hooks';
import type { OAuthProvider } from '@/types/acl';

// Provider icons mapping
const providerIcons: Record<string, React.ReactNode> = {
  google: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor" aria-hidden="true">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
    </svg>
  ),
  github: <Github className="size-5" />,
  microsoft: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor" aria-hidden="true">
      <path d="M11.4 24H0V12.6h11.4V24zM24 24H12.6V12.6H24V24zM11.4 11.4H0V0h11.4v11.4zm12.6 0H12.6V0H24v11.4z" />
    </svg>
  ),
  gitlab: (
    <svg viewBox="0 0 24 24" className="size-5" fill="currentColor" aria-hidden="true">
      <path d="M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 0 1-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 0 1 4.82 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0 1 18.6 2a.43.43 0 0 1 .58 0 .42.42 0 0 1 .11.18l2.44 7.51L23 13.45a.84.84 0 0 1-.35.94z" />
    </svg>
  ),
};

function ProviderIcon({ providerId }: { providerId: string }) {
  return providerIcons[providerId] || <KeyRound className="size-5" />;
}

interface ProviderRowProps {
  provider: OAuthProvider;
  onToggle: (id: string, enabled: boolean) => void;
  isUpdating: boolean;
}

function ProviderRow({ provider, onToggle, isUpdating }: ProviderRowProps) {
  const handleToggle = (checked: boolean) => {
    onToggle(provider.id, checked);
  };

  return (
    <div className="flex items-center justify-between py-4 border-b border-border last:border-b-0">
      <div className="flex items-center gap-4">
        <div className="flex items-center justify-center size-10 rounded-lg bg-muted">
          <ProviderIcon providerId={provider.id} />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-medium">{provider.name}</span>
            {provider.available ? (
              <Badge variant="success" className="text-xs">
                <CheckCircle2 className="size-3 mr-1" />
                Available
              </Badge>
            ) : (
              <Badge variant="secondary" className="text-xs">
                <XCircle className="size-3 mr-1" />
                Not Configured
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground">
            {provider.available
              ? 'Environment variables are configured'
              : 'Environment variables not set on server'}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-4">
        <Switch
          checked={provider.enabled}
          onCheckedChange={handleToggle}
          disabled={!provider.available || isUpdating}
          aria-label={`${provider.enabled ? 'Disable' : 'Enable'} ${provider.name} OAuth`}
        />
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="flex items-center justify-between py-4 border-b last:border-b-0">
          <div className="flex items-center gap-4">
            <Skeleton className="size-10 rounded-lg" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-40" />
            </div>
          </div>
          <Skeleton className="h-6 w-10 rounded-full" />
        </div>
      ))}
    </div>
  );
}

export function ACLOAuthSettings() {
  const { providers, isLoading, isError } = useAdminOAuthProviders();
  const { updateProvider, isUpdating } = useUpdateOAuthProvider();

  const handleToggle = async (id: string, enabled: boolean) => {
    await updateProvider({ id, enabled });
  };

  const availableCount = providers.filter((p) => p.available).length;
  const enabledCount = providers.filter((p) => p.enabled).length;

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>OAuth Providers</CardTitle>
          <CardDescription>
            Configure which OAuth providers are available for login.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <LoadingSkeleton />
        </CardContent>
      </Card>
    );
  }

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>OAuth Providers</CardTitle>
          <CardDescription>
            Configure which OAuth providers are available for login.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 text-destructive">
            <AlertCircle className="size-4" />
            <span>Failed to load OAuth providers</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardHeading>
          <CardTitle>OAuth Providers</CardTitle>
          <CardDescription>
            Configure which OAuth providers are available for login. Only providers with server-side
            environment variables configured can be enabled.
          </CardDescription>
        </CardHeading>
        <div className="flex gap-4 text-sm text-muted-foreground mt-2">
          <span>
            <strong>{availableCount}</strong> of {providers.length} providers configured
          </span>
          <span>
            <strong>{enabledCount}</strong> enabled
          </span>
        </div>
      </CardHeader>
      <CardContent>
        {providers.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <KeyRound className="size-12 mx-auto mb-4 opacity-50" />
            <p>No OAuth providers found.</p>
          </div>
        ) : (
          <div>
            {providers.map((provider) => (
              <ProviderRow
                key={provider.id}
                provider={provider}
                onToggle={handleToggle}
                isUpdating={isUpdating}
              />
            ))}
          </div>
        )}

        {availableCount === 0 && providers.length > 0 && (
          <div className="mt-4 p-4 bg-muted rounded-lg">
            <div className="flex items-start gap-3">
              <AlertCircle className="size-5 text-amber-500 mt-0.5" />
              <div>
                <p className="font-medium text-sm">No OAuth providers configured</p>
                <p className="text-sm text-muted-foreground mt-1">
                  To enable OAuth login, configure the environment variables for your desired
                  providers on the server. For example, set{' '}
                  <code className="text-xs bg-background px-1 py-0.5 rounded">
                    GOOGLE_CLIENT_ID
                  </code>{' '}
                  and{' '}
                  <code className="text-xs bg-background px-1 py-0.5 rounded">
                    GOOGLE_CLIENT_SECRET
                  </code>{' '}
                  for Google OAuth.
                </p>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
