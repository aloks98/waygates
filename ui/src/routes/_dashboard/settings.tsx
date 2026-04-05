import { Tabs, TabsContent, TabsList, TabsTrigger } from '@e412/titanium';
import { AuditConfigPanel } from '@/components/audit-logs';
import { ACLBrandingSettings, CatchallSettings } from '@/components/settings';

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      <Tabs defaultValue="catchall">
        <TabsList variant="line">
          <TabsTrigger value="catchall">Default Page</TabsTrigger>
          <TabsTrigger value="audit">Audit Logs</TabsTrigger>
          <TabsTrigger value="acl-branding">Login Branding</TabsTrigger>
        </TabsList>

        <TabsContent value="catchall" className="mt-6">
          <CatchallSettings />
        </TabsContent>

        <TabsContent value="audit" className="mt-6">
          <AuditConfigPanel />
        </TabsContent>

        <TabsContent value="acl-branding" className="mt-6">
          <ACLBrandingSettings />
        </TabsContent>
      </Tabs>
    </div>
  );
}
