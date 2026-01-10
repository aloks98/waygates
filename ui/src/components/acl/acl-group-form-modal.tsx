import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { useNavigate } from '@tanstack/react-router';
import { Info, Shield } from 'lucide-react';
import { useEffect } from 'react';
import { z } from 'zod';
import { useCreateACLGroup, useUpdateACLGroup } from '@/hooks';
import type { ACLGroup, CombinationMode } from '@/types/acl';

const aclGroupSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  combination_mode: z.enum(['any', 'all', 'ip_bypass']),
});

type ACLGroupFormValues = z.infer<typeof aclGroupSchema>;

interface ACLGroupFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: 'create' | 'edit';
  initialData?: ACLGroup | null;
}

const combinationModeOptions = [
  {
    value: 'any',
    label: 'Any Match',
    description: 'Access granted if any single auth method passes (OR logic)',
  },
  {
    value: 'all',
    label: 'All Required',
    description: 'All configured auth methods must pass (AND logic)',
  },
  {
    value: 'ip_bypass',
    label: 'IP Bypass',
    description: 'Matching IPs skip authentication entirely, others must authenticate',
  },
];

export function ACLGroupFormModal({
  open,
  onOpenChange,
  mode,
  initialData,
}: ACLGroupFormModalProps) {
  const navigate = useNavigate();
  const { createGroup, isCreating } = useCreateACLGroup();
  const { updateGroup, isUpdating } = useUpdateACLGroup();

  const isLoading = isCreating || isUpdating;
  const isEditMode = mode === 'edit' && initialData;

  const form = useForm({
    defaultValues: {
      name: '',
      description: '',
      combination_mode: 'any' as CombinationMode,
    } as ACLGroupFormValues,
    validators: {
      onSubmit: aclGroupSchema,
    },
    onSubmit: async ({ value }) => {
      if (isEditMode) {
        await updateGroup({
          id: initialData.id,
          data: {
            name: value.name,
            description: value.description || undefined,
            combination_mode: value.combination_mode,
          },
        });
        onOpenChange(false);
      } else {
        const response = await createGroup({
          name: value.name,
          description: value.description || undefined,
          combination_mode: value.combination_mode,
        });
        onOpenChange(false);
        // Navigate to the newly created group's detail page to configure rules
        if (response.data?.id) {
          navigate({
            to: '/dashboard/acl/$groupId',
            params: { groupId: String(response.data.id) },
          });
        }
      }
    },
  });

  useEffect(() => {
    if (isEditMode) {
      form.setFieldValue('name', initialData.name);
      form.setFieldValue('description', initialData.description || '');
      form.setFieldValue('combination_mode', initialData.combination_mode);
    } else {
      form.reset();
    }
  }, [initialData, isEditMode, form.setFieldValue, form.reset]);

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="size-5" />
            {isEditMode ? 'Edit ACL Group' : 'Create ACL Group'}
          </DialogTitle>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            e.stopPropagation();
            form.handleSubmit();
          }}
        >
          <DialogBody>
            {!isEditMode && (
              <Alert className="mb-4">
                <Info className="size-4" />
                <AlertDescription>
                  After creating the group, you will be redirected to configure access rules
                  including IP restrictions, basic authentication, Waygates auth, and external
                  providers.
                </AlertDescription>
              </Alert>
            )}
            <FieldGroup>
              <form.Field name="name">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Name</FieldLabel>
                      <Input
                        id={field.name}
                        placeholder="e.g., Engineering Team"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                      />
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <form.Field name="description">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Description (optional)</FieldLabel>
                      <Textarea
                        id={field.name}
                        placeholder="A brief description of what this group is for..."
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        rows={3}
                        aria-invalid={hasError}
                      />
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <form.Field name="combination_mode">
                {(field) => (
                  <Field>
                    <FieldLabel>Combination Mode</FieldLabel>
                    <Select
                      value={field.state.value}
                      onValueChange={(val) => field.handleChange(val as CombinationMode)}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Select mode...">
                          {combinationModeOptions.find((o) => o.value === field.state.value)?.label}
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {combinationModeOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            <div className="flex flex-col items-start">
                              <span className="font-medium">{option.label}</span>
                              <span className="text-xs text-muted-foreground">
                                {option.description}
                              </span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FieldDescription>
                      {
                        combinationModeOptions.find((o) => o.value === field.state.value)
                          ?.description
                      }
                    </FieldDescription>
                  </Field>
                )}
              </form.Field>
            </FieldGroup>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Create & Configure'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
