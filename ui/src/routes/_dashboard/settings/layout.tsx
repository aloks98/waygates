import { Outlet } from '@tanstack/react-router';

import { SettingsNav } from '@/components/settings';

export function SettingsLayout() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-[28px] font-semibold">Settings</h1>
      <div className="flex flex-col gap-6 md:flex-row">
        <SettingsNav />
        <div className="min-w-0 flex-1">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
