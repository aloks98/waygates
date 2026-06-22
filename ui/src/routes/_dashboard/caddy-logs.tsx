import { Tabs, TabsList, TabsTrigger } from '@e412/rnui-react';
import { useState } from 'react';

import { LogViewer } from '@/components/caddy-logs';
import { useCaddyLogs } from '@/hooks/use-caddy-logs';
import type { CaddyLogSource } from '@/types/caddy-logs';

export function CaddyLogsPage() {
  const [source, setSource] = useState<CaddyLogSource>('runtime');
  const { lines, isStreaming, error, pause, resume, clear } = useCaddyLogs(source);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Caddy Logs</h1>

      <Tabs value={source} onValueChange={(v) => setSource(v as CaddyLogSource)}>
        <TabsList variant="line">
          <TabsTrigger value="runtime">Runtime</TabsTrigger>
          <TabsTrigger value="access">Access</TabsTrigger>
        </TabsList>
      </Tabs>

      <LogViewer
        source={source}
        lines={lines}
        isStreaming={isStreaming}
        error={error}
        onPause={pause}
        onResume={resume}
        onClear={clear}
      />
    </div>
  );
}
