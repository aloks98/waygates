import { useNavigate, useSearch } from '@tanstack/react-router';

import { LogViewer } from '@/components/caddy-logs';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useCaddyLogs } from '@/hooks/use-caddy-logs';
import type { CaddyLogSource } from '@/types/caddy-logs';

export function CaddyLogsPage() {
  const { source = 'runtime' } = useSearch({ strict: false }) as { source?: CaddyLogSource };
  const navigate = useNavigate();
  const { lines, isStreaming, error, paused, pause, resume, clear } = useCaddyLogs(source);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-[28px] font-semibold">
        {source === 'access' ? 'Access logs' : 'Runtime logs'}
      </h1>

      <Tabs
        value={source}
        onValueChange={(v) =>
          navigate({ to: '/caddy-logs', search: { source: v as CaddyLogSource } })
        }
      >
        <TabsList variant="line">
          <TabsTrigger value="runtime">Runtime</TabsTrigger>
          <TabsTrigger value="access">Access</TabsTrigger>
        </TabsList>
      </Tabs>

      <LogViewer
        source={source}
        lines={lines}
        isStreaming={isStreaming}
        paused={paused}
        error={error}
        onPause={pause}
        onResume={resume}
        onClear={clear}
      />
    </div>
  );
}
