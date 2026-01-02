import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, Badge } from '@e412/titanium';
import { Globe, Activity, Pause } from 'lucide-react';
import { api } from '../../lib/api';
import { useAuthStore } from '../../stores/auth';
import type { ApiResponse, PaginatedResponse } from '../../types/api';
import type { Proxy } from '../../types/proxy';

export function DashboardIndex() {
  const { user } = useAuthStore();

  const { data: proxiesData } = useQuery({
    queryKey: ['proxies', 'summary'],
    queryFn: async () => {
      const response = await api
        .get('proxies', { searchParams: { limit: '1000' } })
        .json<ApiResponse<PaginatedResponse<Proxy>>>();
      return response.data;
    },
  });

  const proxies = proxiesData?.items || [];
  const activeProxies = proxies.filter((p) => p.is_active).length;
  const inactiveProxies = proxies.filter((p) => !p.is_active).length;

  const stats = [
    { label: 'Total Proxies', value: proxies.length, icon: Globe, color: 'text-primary' },
    { label: 'Active', value: activeProxies, icon: Activity, color: 'text-green-500' },
    { label: 'Inactive', value: inactiveProxies, icon: Pause, color: 'text-muted-foreground' },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">
          Welcome back, {user?.name || 'User'}
        </h1>
        <p className="mt-1 text-muted-foreground">
          Here&apos;s an overview of your proxy configuration.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {stats.map((stat) => (
          <Card key={stat.label}>
            <CardContent className="flex items-center gap-4 p-6">
              <div className={`rounded-full bg-secondary p-3 ${stat.color}`}>
                <stat.icon className="size-6" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{stat.label}</p>
                <p className="text-2xl font-bold">{stat.value}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Recent Proxies</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Hostname</TableHead>
                <TableHead>Target</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {proxies.slice(0, 5).map((proxy) => (
                <TableRow key={proxy.id}>
                  <TableCell className="font-medium">{proxy.hostname}</TableCell>
                  <TableCell className="text-muted-foreground">{proxy.target_url}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{proxy.proxy_type}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                      {proxy.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
              {proxies.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="h-24 text-center text-muted-foreground">
                    No proxies configured yet. Create your first proxy to get started.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
