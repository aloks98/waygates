import { ActivityTimeline } from '@/components/dashboard/activity-timeline';
import { DashboardEmptyState } from '@/components/dashboard/dashboard-empty-state';
import { DashboardQuickActions } from '@/components/dashboard/dashboard-quick-actions';
import { DashboardStatCards } from '@/components/dashboard/dashboard-stat-cards';
import { FleetComposition } from '@/components/dashboard/fleet-composition';
import { SystemStatusBar } from '@/components/dashboard/system-status-bar';
import { TrafficCharts } from '@/components/dashboard/traffic-charts';
import { useAppStatus, useDashboardData } from '@/hooks';
import { useAuthStore } from '@/stores/auth';

export function DashboardIndex() {
  const { user } = useAuthStore();
  const { proxyStats, l4ProxyStats, recentActivity, isLoading } = useDashboardData();
  const { appStatus } = useAppStatus();

  const totalProxies = (proxyStats?.total ?? 0) + (l4ProxyStats?.total_proxies ?? 0);
  const isFirstRun = !isLoading && (totalProxies === 0 || appStatus?.user_setup_complete === false);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Welcome back, {user?.name || 'User'}</h1>
        <div className="mt-3">
          <SystemStatusBar />
        </div>
      </div>

      {isFirstRun ? (
        <DashboardEmptyState />
      ) : (
        <>
          <DashboardStatCards
            proxyStats={proxyStats}
            l4ProxyStats={l4ProxyStats}
            isLoading={isLoading}
          />
          <TrafficCharts />
          <div className="grid gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <FleetComposition
                proxyStats={proxyStats}
                l4ProxyStats={l4ProxyStats}
                isLoading={isLoading}
              />
            </div>
            <div>
              <DashboardQuickActions />
            </div>
          </div>
        </>
      )}

      <ActivityTimeline activity={recentActivity} isLoading={isLoading} />
    </div>
  );
}
