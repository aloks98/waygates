import { Button, FormControl, FormField, FormItem, FormMessage, Input } from '@e412/rnui-react';
import { Plus, Trash2 } from 'lucide-react';
import { useFieldArray, useFormContext } from 'react-hook-form';

import type { StaticFormValues } from '@/lib/form-validation';

export function TryFilesFields() {
  const form = useFormContext<StaticFormValues>();
  const { fields, append, remove } = useFieldArray({ control: form.control, name: 'try_files' });

  return (
    <div className="space-y-3">
      {fields.length === 0 && (
        <p className="text-sm text-muted-foreground">No try_files configured.</p>
      )}
      {fields.map((item, index) => (
        <div key={item.id} className="flex items-start gap-2">
          <FormField
            control={form.control}
            name={`try_files.${index}.value`}
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormControl>
                  <Input placeholder="{path}" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Remove file"
            onClick={() => remove(index)}
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => append({ value: '' })}>
        <Plus className="size-4" /> Add File
      </Button>
    </div>
  );
}
