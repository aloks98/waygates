import { Tabs, TabsContent, TabsList, TabsTrigger } from '@e412/titanium';
import { AuditConfigPanel } from '@/components/audit-logs';
import { CatchallSettings } from '@/components/settings';

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      <Tabs defaultValue="catchall">
        <TabsList variant="line">
          <TabsTrigger value="catchall">Catchall</TabsTrigger>
          <TabsTrigger value="audit">Audit Logs</TabsTrigger>
        </TabsList>

        <TabsContent value="catchall" className="mt-6">
          <CatchallSettings />
        </TabsContent>

        <TabsContent value="audit" className="mt-6">
          <AuditConfigPanel />
        </TabsContent>
      </Tabs>
    </div>
  );
}
