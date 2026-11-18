import { SlidersHorizontal } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import {
  useForm,
  type FieldPath,
  type FieldValues,
  type Resolver,
  type UseFormReturn,
} from 'react-hook-form';

import { Button } from '@/components/ui/button';
import { Form } from '@/components/ui/form';
import { Sheet, SheetContent, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet';

export function AdvancedSettingsSheet<T extends FieldValues>({
  form,
  resolver,
  fields,
  children,
  title = 'Advanced settings',
}: {
  form: UseFormReturn<T>;
  resolver: Resolver<T>;
  fields: FieldPath<T>[];
  children: ReactNode;
  title?: string;
}) {
  const [open, setOpen] = useState(false);
  const draft = useForm<T>({ resolver, mode: 'onTouched' });
  const apply = async () => {
    if (!(await draft.trigger(fields, { shouldFocus: true }))) return;
    for (const name of fields)
      form.setValue(name, structuredClone(draft.getValues(name)), {
        shouldDirty: true,
        shouldValidate: true,
      });
    setOpen(false);
  };
  return (
    <>
      <Button
        type="button"
        variant="outline"
        className="self-start justify-start"
        onClick={() => {
          draft.reset(structuredClone(form.getValues()));
          setOpen(true);
        }}
      >
        <SlidersHorizontal data-icon="inline-start" />
        Configure {title.replace(/^./, (letter) => letter.toLowerCase())}
      </Button>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent className="sm:max-w-xl" aria-describedby={undefined}>
          <SheetHeader>
            <SheetTitle>{title}</SheetTitle>
          </SheetHeader>
          <Form {...draft}>
            <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 py-2">
              {children}
            </div>
          </Form>
          <SheetFooter className="flex-row justify-end border-t">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="button" onClick={apply}>
              Apply
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  );
}
