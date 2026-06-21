import { Button, FormControl, FormField, FormItem, FormMessage, Input } from '@e412/rnui-react';
import { Plus, Trash2 } from 'lucide-react';
import { useFieldArray, useFormContext } from 'react-hook-form';

import type { ReverseProxyFormValues } from '@/lib/form-validation';

export function CustomHeadersFields() {
  const form = useFormContext<ReverseProxyFormValues>();

  const requestHeaders = useFieldArray({ control: form.control, name: 'request_headers' });
  const responseHeaders = useFieldArray({ control: form.control, name: 'response_headers' });

  const sections = [
    {
      label: 'Request headers (to upstream)',
      fieldArray: requestHeaders,
      name: 'request_headers' as const,
    },
    {
      label: 'Response headers (to client)',
      fieldArray: responseHeaders,
      name: 'response_headers' as const,
    },
  ];

  return (
    <div className="space-y-6">
      {sections.map((section) => (
        <div key={section.name} className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">{section.label}</span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => section.fieldArray.append({ name: '', value: '' })}
            >
              <Plus className="mr-1 size-4" />
              Add Header
            </Button>
          </div>
          {section.fieldArray.fields.map((item, index) => (
            <div key={item.id} className="flex items-start gap-2">
              <FormField
                control={form.control}
                name={`${section.name}.${index}.name`}
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input placeholder="Header-Name" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`${section.name}.${index}.value`}
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormControl>
                      <Input placeholder="value" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Remove header"
                onClick={() => section.fieldArray.remove(index)}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
