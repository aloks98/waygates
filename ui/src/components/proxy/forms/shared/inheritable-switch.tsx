import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
} from '@e412/rnui-react';

// System defaults applied when neither the proxy nor its group has an
// opinion — must match backend/internal/proxygroup/resolve.go exactly.
export const PROXY_SYSTEM_DEFAULTS = {
  ssl_enabled: true,
  ssl_forced: true,
  block_exploits: true,
  tls_insecure_skip_verify: false,
} as const;

export interface InheritableSwitchProps {
  value: boolean | null;
  onChange: (value: boolean | null) => void;
  /** The selected group's value for this field, or null if the group is silent. */
  groupValue: boolean | null;
  /** Applied when neither the proxy nor the group has an opinion. */
  systemDefault: boolean;
  /** When there is no group, collapse to a plain switch. */
  hasGroup: boolean;
  label: string;
  /**
   * Standard a11y props RHF's FormControl injects onto its single child
   * (id, aria-describedby, aria-invalid) when this is wired up via
   * FormField/FormControl, same as a plain <Switch> or <SelectTrigger>
   * would receive directly. Forwarded to whichever control actually
   * renders (the Select trigger with a group, the Switch without one).
   */
  id?: string;
  'aria-describedby'?: string;
  'aria-invalid'?: boolean;
}

/**
 * Three states — Inherit / On / Off — backed by `boolean | null`. The
 * Inherit option's label resolves live against the selected group's value,
 * so it reads "Inherit (on)" or "Inherit (off)" instead of leaving the user
 * to guess what inheriting actually does.
 *
 * Without a group, "inherit" resolves to the system default and there is
 * nothing group-specific to show — so this collapses to a plain switch
 * rather than offering a third state the user can't reason about. `null`
 * still round-trips to the server, which is what keeps the proxy inheriting
 * if it is later added to a group.
 */
export function InheritableSwitch({
  value,
  onChange,
  groupValue,
  systemDefault,
  hasGroup,
  label,
  id,
  'aria-describedby': ariaDescribedBy,
  'aria-invalid': ariaInvalid,
}: InheritableSwitchProps) {
  if (!hasGroup) {
    return (
      <Switch
        checked={value ?? systemDefault}
        onCheckedChange={(next) => onChange(next)}
        aria-label={label}
        id={id}
        aria-describedby={ariaDescribedBy}
        aria-invalid={ariaInvalid}
      />
    );
  }

  const inherited = groupValue ?? systemDefault;
  const inheritLabel = `Inherit (${inherited ? 'on' : 'off'})`;

  return (
    <Select
      value={value === null ? 'inherit' : value ? 'on' : 'off'}
      onValueChange={(next) => onChange(next === 'inherit' ? null : next === 'on')}
    >
      <SelectTrigger
        aria-label={label}
        id={id}
        aria-describedby={ariaDescribedBy}
        aria-invalid={ariaInvalid}
        className="w-40"
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="inherit">{inheritLabel}</SelectItem>
        <SelectItem value="on">On</SelectItem>
        <SelectItem value="off">Off</SelectItem>
      </SelectContent>
    </Select>
  );
}
