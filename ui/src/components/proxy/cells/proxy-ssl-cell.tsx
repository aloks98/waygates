interface ProxySslCellProps {
  enabled: boolean;
}

export function ProxySslCell({ enabled }: ProxySslCellProps) {
  if (enabled) {
    return <span className="text-green-600">Enabled</span>;
  }

  return <span className="text-muted-foreground">Disabled</span>;
}
