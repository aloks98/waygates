import { Tabs, TabsContent, TabsList, TabsTrigger } from '@e412/titanium';
import { AuditConfigPanel } from '@/components/audit-logs';
import { ACLBrandingSettings, ACLOAuthSettings, CatchallSettings } from '@/components/settings';

export function SettingsPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      <Tabs defaultValue="catchall">
        <TabsList variant="line">
          <TabsTrigger value="catchall">Catchall</TabsTrigger>
          <TabsTrigger value="audit">Audit Logs</TabsTrigger>
          <TabsTrigger value="acl-branding">ACL Branding</TabsTrigger>
          <TabsTrigger value="oauth-providers">OAuth Providers</TabsTrigger>
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

        <TabsContent value="oauth-providers" className="mt-6">
          <ACLOAuthSettings />
        </TabsContent>
      </Tabs>
    </div>
  );
}
