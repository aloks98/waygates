import { Spinner } from '@e412/rnui-react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { useEffect, useRef } from 'react';

import { publicApi } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import type { ApiResponse, TokenPair } from '@/types/api';

export function SSOCallbackPage() {
  const navigate = useNavigate();
  const setTokens = useAuthStore((s) => s.setTokens);
  const search = useSearch({ strict: false }) as { code?: string };
  const ran = useRef(false);

  useEffect(() => {
    if (ran.current) return;
    ran.current = true;
    const code = search.code;
    if (!code) {
      navigate({ to: '/login', search: { sso_error: 'sso_failed' } });
      return;
    }
    void (async () => {
      try {
        const res = await publicApi
          .post('auth/sso/exchange', { json: { code } })
          .json<ApiResponse<TokenPair>>();
        if (res.success && res.data) {
          setTokens(res.data);
          navigate({ to: '/' });
        } else {
          navigate({ to: '/login', search: { sso_error: 'sso_failed' } });
        }
      } catch {
        navigate({ to: '/login', search: { sso_error: 'sso_failed' } });
      }
    })();
  }, [search.code, navigate, setTokens]);

  return (
    <div className="flex min-h-screen items-center justify-center gap-3 text-muted-foreground">
      <Spinner variant="circle" />
      Signing you in…
    </div>
  );
}
