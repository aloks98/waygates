import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/titanium';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { ExternalLink, Globe, Link2, Shield } from 'lucide-react';
import { api } from '@/lib/api';
import type { ProxyACLAssignment } from '@/types/acl';
import type { ApiResponse } from '@/types/api';

interface GroupUsageTabProps {
  groupId: number;
}

export function GroupUsageTab({ groupId }: GroupUsageTabProps) {
  const navigate = useNavigate();

  // Fetch proxy ACL assignments for this group using the dedicated endpoint
  const { data: assignments, isLoading } = useQuery({
    queryKey: ['acl-groups', groupId, 'usage'],
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/usage`)
        .json<ApiResponse<ProxyACLAssignment[]>>();
      return response.data ?? [];
    },
    staleTime: 30000, // Cache for 30 seconds
  });

  const handleProxyClick = (_proxyId: number) => {
    navigate({ to: '/dashboard/proxies' });
    // In a more complete implementation, this would navigate to a proxy detail page
  };

  return (
    <Card>
      <CardHeader>
        <CardHeading>
          <CardTitle className="flex items-center gap-2">
            <Link2 className="size-5" />
            Proxy Usage
          </CardTitle>
          <CardDescription>
            Proxies that are protected by this ACL group. Changes to this group will affect all
            listed proxies.
          </CardDescription>
        </CardHeading>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="flex items-center gap-4">
                <Skeleton className="size-10 rounded" />
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-5 w-48 flex-1" />
                <Skeleton className="h-6 w-20" />
              </div>
            ))}
          </div>
        ) : !assignments || assignments.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <Shield className="size-12 mx-auto mb-4 opacity-50" />
            <p>No proxies using this group</p>
            <p className="text-sm mt-1">
              Assign this ACL group to a proxy to protect it with these access controls
            </p>
            <Button
              variant="outline"
              className="mt-4"
              onClick={() => navigate({ to: '/dashboard/proxies' })}
            >
              <Globe className="size-4" />
              Go to Proxies
            </Button>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Proxy</TableHead>
                <TableHead>Hostname</TableHead>
                <TableHead>Path Pattern</TableHead>
                <TableHead className="w-24">Priority</TableHead>
                <TableHead className="w-24">Status</TableHead>
                <TableHead className="w-20 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {assignments.map((assignment) => {
                const proxy = assignment.proxy;
                if (!proxy) return null;

                return (
                  <TableRow key={assignment.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="flex items-center justify-center size-8 rounded bg-muted">
                          <Globe className="size-4 text-muted-foreground" />
                        </div>
                        <span className="font-medium">{proxy.name || proxy.hostname}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <a
                        href={`https://${proxy.hostname}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
                      >
                        <code className="bg-muted px-1.5 py-0.5 rounded text-xs">
                          {proxy.hostname}
                        </code>
                        <ExternalLink className="size-3" />
                      </a>
                    </TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted px-2 py-0.5 rounded">
                        {assignment.path_pattern || '/*'}
                      </code>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="font-mono">
                        {assignment.priority}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {assignment.enabled ? (
                        <Badge variant="default">Active</Badge>
                      ) : (
                        <Badge variant="secondary">Disabled</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8 p-0"
                              onClick={() => handleProxyClick(proxy.id)}
                            >
                              <ExternalLink className="size-4" />
                              <span className="sr-only">View proxy</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>View proxy</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
