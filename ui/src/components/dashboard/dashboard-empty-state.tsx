import { Button, Card, CardContent, EmptyState } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { Globe, Network, Rocket } from 'lucide-react';

export function DashboardEmptyState() {
  return (
    <Card>
      <CardContent className="py-10">
        <EmptyState
          icon={<Rocket className="size-8" />}
          title="Welcome to Waygates"
          description="You don't have any proxies yet. Create your first one to start routing traffic — Waygates handles HTTPS automatically."
          action={
            <div className="flex flex-wrap justify-center gap-2">
              <Button render={<Link to="/dashboard/proxies/new" />}>
                <Globe className="size-4" />
                Create HTTP proxy
              </Button>
              <Button variant="outline" render={<Link to="/dashboard/proxies/tcp-udp/new" />}>
                <Network className="size-4" />
                Create TCP/UDP proxy
              </Button>
            </div>
          }
        />
      </CardContent>
    </Card>
  );
}
