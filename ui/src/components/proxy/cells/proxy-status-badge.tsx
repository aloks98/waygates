import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';

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
        <Switch
          checked={isActive}
          onCheckedChange={onToggle}
          disabled={isToggling}
          size="sm"
          aria-label={isActive ? 'Disable proxy' : 'Enable proxy'}
        />
        <span className="text-sm text-muted-foreground">{isActive ? 'Active' : 'Inactive'}</span>
      </div>
    );
  }

  return (
    <Badge
      className={isActive ? 'border-success/25 bg-success/5 text-success' : 'text-muted-foreground'}
      variant={'outline'}
    >
      {isActive ? 'Active' : 'Inactive'}
    </Badge>
  );
}
