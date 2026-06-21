import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Textarea,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate } from '@tanstack/react-router';
import { Info, Shield } from 'lucide-react';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { getModeLabel } from '@/components/acl/access-labels';
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
    label: getModeLabel('any'),
    description: 'Access granted if any single auth method passes (OR logic)',
  },
  {
    value: 'all',
    label: getModeLabel('all'),
    description: 'All configured auth methods must pass (AND logic)',
  },
  {
    value: 'ip_bypass',
    label: getModeLabel('ip_bypass'),
    description: 'Trusted IPs skip authentication entirely; others must authenticate normally',
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

  const form = useForm<ACLGroupFormValues>({
    resolver: zodResolver(aclGroupSchema),
    defaultValues: {
      name: '',
      description: '',
      combination_mode: 'any' as CombinationMode,
    },
  });

  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      form.reset({
        name: initialData.name,
        description: initialData.description ?? '',
        combination_mode: initialData.combination_mode,
      });
    } else {
      form.reset({
        name: '',
        description: '',
        combination_mode: 'any',
      });
    }
  }, [open, isEditMode, initialData, form]);

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
    }
    onOpenChange(isOpen);
  };

  const onSubmit = async (value: ACLGroupFormValues) => {
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
          to: '/dashboard/access/$groupId',
          params: { groupId: String(response.data.id) },
        });
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Shield className="size-5" />
            {isEditMode ? 'Edit Access Group' : 'Create Access Group'}
          </DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              {!isEditMode && (
                <Alert className="mb-4">
                  <Info className="size-4" />
                  <AlertDescription>
                    After creating the group, you will be redirected to configure access rules
                    including IP restrictions, basic authentication, OAuth / SSO, Waygates Account,
                    and Forward Auth.
                  </AlertDescription>
                </Alert>
              )}
              <div className="grid gap-4">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Name</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g., Engineering Team" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="description"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Description (optional)</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder="A brief description of what this group is for..."
                          rows={3}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="combination_mode"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>How methods combine</FormLabel>
                      <Select value={field.value} onValueChange={field.onChange}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="Select mode...">
                              {combinationModeOptions.find((o) => o.value === field.value)?.label}
                            </SelectValue>
                          </SelectTrigger>
                        </FormControl>
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
                      <FormDescription>
                        {combinationModeOptions.find((o) => o.value === field.value)?.description}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Create & Configure'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
