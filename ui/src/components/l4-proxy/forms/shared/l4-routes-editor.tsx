import { Button, FormField, FormItem, FormMessage } from '@e412/rnui-react';
import { Plus } from 'lucide-react';
import { useFieldArray, useFormContext } from 'react-hook-form';

import type { L4ProxyFormValues } from '@/lib/form-validation';

import { createDefaultRoute } from './l4-proxy-form-mappers';
import { L4RouteCard } from './l4-route-card';

export function L4RoutesEditor() {
  const form = useFormContext<L4ProxyFormValues>();
  const { fields, append, remove } = useFieldArray({ control: form.control, name: 'routes' });

  return (
    <div className="space-y-4">
      {fields.map((item, i) => (
        <L4RouteCard
          key={item.id}
          routeIndex={i}
          totalRoutes={fields.length}
          onRemove={() => remove(i)}
          defaultOpen={i === 0}
        />
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => append(createDefaultRoute())}
      >
        <Plus className="size-4" /> Add Route
      </Button>
      <FormField
        control={form.control}
        name="routes"
        render={() => (
          <FormItem>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  );
}
