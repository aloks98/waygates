import { Children, cloneElement, createContext, isValidElement, useContext, useId } from 'react';
import type { ComponentProps, ReactElement } from 'react';
import {
  Controller,
  FormProvider,
  useFormContext,
  useFormState,
  type ControllerProps,
  type FieldPath,
  type FieldValues,
} from 'react-hook-form';

import { Field, FieldDescription, FieldError, FieldLabel } from '@/components/ui/field';

const Form = FormProvider;
const FieldContext = createContext<{ name: string } | null>(null);
const ItemContext = createContext<string | null>(null);

function FormField<T extends FieldValues, N extends FieldPath<T>>(props: ControllerProps<T, N>) {
  return (
    <FieldContext.Provider value={{ name: props.name }}>
      <Controller {...props} />
    </FieldContext.Provider>
  );
}

function useFormField() {
  const field = useContext(FieldContext);
  const id = useContext(ItemContext);
  const { getFieldState } = useFormContext();
  const state = useFormState({ name: field?.name });
  if (!field || !id) throw new Error('Form controls must be inside FormField and FormItem.');
  return {
    id,
    ...getFieldState(field.name, state),
    descriptionId: `${id}-description`,
    errorId: `${id}-error`,
  };
}

function FormItem(props: ComponentProps<typeof Field>) {
  const id = useId();
  return (
    <ItemContext.Provider value={id}>
      <Field {...props} />
    </ItemContext.Provider>
  );
}

function FormLabel(props: ComponentProps<typeof FieldLabel>) {
  const { id, error } = useFormField();
  return <FieldLabel htmlFor={id} data-invalid={!!error || undefined} {...props} />;
}

function FormControl({ children }: { children: ReactElement }) {
  const { id, error, errorId } = useFormField();
  const child = Children.only(children);
  return isValidElement(child)
    ? cloneElement(child as ReactElement<Record<string, unknown>>, {
        id,
        'aria-invalid': !!error,
        'aria-describedby': error ? errorId : undefined,
      })
    : child;
}

function FormDescription(props: ComponentProps<typeof FieldDescription>) {
  const { descriptionId } = useFormField();
  return <FieldDescription id={descriptionId} {...props} />;
}

function FormMessage({ children, ...props }: ComponentProps<typeof FieldError>) {
  const { error, errorId } = useFormField();
  const message = error?.message ? String(error.message) : children;
  return message ? (
    <FieldError id={errorId} {...props}>
      {message}
    </FieldError>
  ) : null;
}

export { Form, FormField, FormItem, FormLabel, FormControl, FormDescription, FormMessage };
