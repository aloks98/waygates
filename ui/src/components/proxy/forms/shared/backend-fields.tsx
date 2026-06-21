import {
  Button,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@e412/rnui-react';
import { Plus, Trash2 } from 'lucide-react';
import { useFieldArray, useFormContext } from 'react-hook-form';

import type { ReverseProxyFormValues } from '@/lib/form-validation';

export function BackendFields() {
  const form = useFormContext<ReverseProxyFormValues>();
  const { fields, append, remove } = useFieldArray({ control: form.control, name: 'upstreams' });

  return (
    <div className="space-y-3">
      {fields.map((item, index) => (
        <div key={item.id} className="flex items-start gap-2">
          <FormField
            control={form.control}
            name={`upstreams.${index}.scheme`}
            render={({ field }) => (
              <FormItem className="w-28">
                <FormControl>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">http</SelectItem>
                      <SelectItem value="https">https</SelectItem>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`upstreams.${index}.host`}
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input placeholder="10.0.0.5 or backend.internal" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={`upstreams.${index}.port`}
            render={({ field }) => (
              <FormItem className="w-28">
                <FormControl>
                  <Input
                    type="number"
                    placeholder="8080"
                    {...field}
                    value={field.value ?? ''}
                    onChange={(e) =>
                      field.onChange(e.target.value === '' ? undefined : e.target.valueAsNumber)
                    }
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          {fields.length > 1 && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Remove server"
              onClick={() => remove(index)}
            >
              <Trash2 className="size-4" />
            </Button>
          )}
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => append({ host: '', port: 8080, scheme: 'http' })}
      >
        <Plus className="size-4" /> Add Server
      </Button>
      <FormField
        control={form.control}
        name="upstreams"
        render={() => (
          <FormItem>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
