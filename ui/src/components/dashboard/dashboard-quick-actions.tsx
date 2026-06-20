import { Button, Card, CardContent, CardHeader, CardTitle } from '@e412/rnui-react';
import { Link } from '@tanstack/react-router';
import { Globe, Network, Shield } from 'lucide-react';

export function DashboardQuickActions() {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Quick actions</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <Button
          variant="outline"
          className="justify-start"
          render={<Link to="/dashboard/proxies/new" />}
        >
          <Globe className="size-4" />
          New HTTP proxy
        </Button>
        <Button
          variant="outline"
          className="justify-start"
          render={<Link to="/dashboard/proxies/tcp-udp/new" />}
        >
          <Network className="size-4" />
          New TCP/UDP proxy
        </Button>
        <Button
          variant="outline"
          className="justify-start"
          render={<Link to="/dashboard/access" />}
        >
          <Shield className="size-4" />
          New access group
        </Button>
      </CardContent>
    </Card>
  );
}
