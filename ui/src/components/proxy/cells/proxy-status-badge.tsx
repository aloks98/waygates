import { Button, Badge } from '@e412/titanium';

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
  const badge = (
    <Badge variant={isActive ? 'primary' : 'secondary'}>
      {isActive ? 'Active' : 'Inactive'}
    </Badge>
  );

  if (canToggle && onToggle) {
    return (
      <Button
        variant="ghost"
        size="sm"
        onClick={onToggle}
        disabled={isToggling}
      >
        {badge}
      </Button>
    );
  }

  return badge;
}
