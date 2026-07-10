interface ProxySslCellProps {
  enabled: boolean | null;
}

export function ProxySslCell({ enabled }: ProxySslCellProps) {
  // null = inherit. The list row doesn't carry the group's resolved value,
  // so this can't guess On/Off without risking a wrong answer — show a
  // neutral "Inherited" rather than silently defaulting to Disabled.
  if (enabled === null) {
    return <span className="text-muted-foreground">Inherited</span>;
  }

  if (enabled) {
    return <span className="text-green-600">Enabled</span>;
  }

  return <span className="text-muted-foreground">Disabled</span>;
}
