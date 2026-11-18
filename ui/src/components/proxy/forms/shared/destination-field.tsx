import { Plus, Trash2 } from 'lucide-react';
import { useId, useState } from 'react';
import { useFieldArray, useFormContext, useWatch } from 'react-hook-form';

import { Button } from '@/components/ui/button';
import { Field, FieldError, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import type { ReverseProxyFormValues } from '@/lib/form-validation';
import { formatUpstream, parseUpstream } from '@/lib/upstream-url';

function DestinationInput({
  index,
  removable,
  onRemove,
}: {
  index: number;
  removable: boolean;
  onRemove: () => void;
}) {
  const form = useFormContext<ReverseProxyFormValues>();
  const upstream = useWatch({ control: form.control, name: `upstreams.${index}` });
  const formatted = formatUpstream(upstream);
  // Keep incomplete URLs while typing; external resets replace the local draft.
  const [draft, setDraft] = useState<{ value: string; persisted: string }>();
  const value = draft?.persisted === formatted ? draft.value : formatted;
  const id = useId();
  const errors = form.formState.errors.upstreams?.[index];
  const error = errors?.host?.message || errors?.port?.message || errors?.scheme?.message;
  return (
    <Field data-invalid={!!error}>
      <FieldLabel htmlFor={id}>
        {index === 0 ? 'Destination URL' : `Destination URL ${index + 1}`}
      </FieldLabel>
      <div className="flex items-center gap-2">
        <Input
          id={id}
          className="min-w-0 font-mono"
          placeholder="http://analytics:3000"
          value={value}
          aria-invalid={!!error}
          aria-describedby={error ? `${id}-error` : undefined}
          onChange={(e) => {
            const raw = e.target.value;
            const parsed = parseUpstream(raw);
            const next = parsed ?? { ...upstream, host: '' };
            setDraft({ value: raw, persisted: formatUpstream(next) });
            form.setValue(`upstreams.${index}.host`, next.host, { shouldDirty: true });
            if (parsed) {
              form.setValue(`upstreams.${index}.port`, parsed.port, { shouldDirty: true });
              form.setValue(`upstreams.${index}.scheme`, parsed.scheme, { shouldDirty: true });
              void form.trigger(`upstreams.${index}`);
            }
          }}
          onBlur={() => {
            if (parseUpstream(value)) void form.trigger(`upstreams.${index}`);
            else
              form.setError(`upstreams.${index}.host`, {
                message: 'Enter an HTTP or HTTPS URL with a host and optional port.',
              });
          }}
        />
        {removable && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Remove destination ${index + 1}`}
            onClick={onRemove}
          >
            <Trash2 />
          </Button>
        )}
      </div>
      <FieldError id={`${id}-error`}>{error}</FieldError>
    </Field>
  );
}

export function DestinationField() {
  const { control } = useFormContext<ReverseProxyFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: 'upstreams' });
  return (
    <div className="flex flex-col gap-4">
      {fields.map((field, index) => (
        <DestinationInput
          key={field.id}
          index={index}
          removable={fields.length > 1}
          onRemove={() => remove(index)}
        />
      ))}
      <Button
        type="button"
        variant="outline"
        className="self-start"
        size="sm"
        onClick={() => append({ host: '', port: 8080, scheme: 'http' }, { shouldFocus: false })}
      >
        <Plus data-icon="inline-start" />
        Add destination
      </Button>
    </div>
  );
}
