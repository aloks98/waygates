import { Badge, Switch } from '@e412/titanium';

interface ProxyStatusBadgeProps {
  isActive: boolean;
  canToggle?: boolean;
  onToggle?: () => void;
  isToggling?: boolean;
}

export function ProxyStatusBadge({
  isActive,
  canToggle = false,
  onToggle,
  isToggling = false,
}: ProxyStatusBadgeProps) {
  if (canToggle && onToggle) {
    return (
      <div className="flex items-center gap-2">
        <Switch checked={isActive} onCheckedChange={onToggle} disabled={isToggling} size="sm" />
        <span className="text-sm text-muted-foreground">{isActive ? 'Active' : 'Inactive'}</span>
      </div>
    );
  }

  return (
    <Badge variant={isActive ? 'primary' : 'secondary'}>{isActive ? 'Active' : 'Inactive'}</Badge>
  );
}
