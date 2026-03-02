import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  CardToolbar,
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
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Textarea,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { format } from 'date-fns';
import { Network, Pencil, Plus, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { z } from 'zod';

import { useAddIPRule, useDeleteIPRule, useIPRules, useUpdateIPRule } from '@/hooks';
import type { ACLIPRule, IPRuleType } from '@/types/acl';

const ipRuleSchema = z.object({
  cidr: z
    .string()
    .min(1, 'CIDR is required')
    .regex(
      /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$|^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}(\/\d{1,3})?$/,
      'Must be a valid IP address or CIDR notation',
    ),
  rule_type: z.enum(['allow', 'deny', 'bypass']),
  description: z.string().max(200, 'Description must be at most 200 characters').optional(),
  priority: z
    .number()
    .min(0, 'Priority must be at least 0')
    .max(1000, 'Priority must be at most 1000'),
});

type IPRuleFormValues = z.infer<typeof ipRuleSchema>;

function getRuleTypeBadgeVariant(
  type: IPRuleType,
): 'primary' | 'secondary' | 'outline' | 'destructive' {
  switch (type) {
    case 'allow':
      return 'primary';
    case 'deny':
      return 'destructive';
    case 'bypass':
      return 'secondary';
    default:
      return 'outline';
  }
}

function getRuleTypeLabel(type: IPRuleType): string {
  switch (type) {
    case 'allow':
      return 'Allow';
    case 'deny':
      return 'Deny';
    case 'bypass':
      return 'Bypass';
    default:
      return type;
  }
}

interface IPRuleFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groupId: number;
  mode: 'create' | 'edit';
  initialData?: ACLIPRule | null;
}

function IPRuleFormModal({ open, onOpenChange, groupId, mode, initialData }: IPRuleFormModalProps) {
  const { addRule, isAdding } = useAddIPRule();
  const { updateRule, isUpdating } = useUpdateIPRule();

  const isLoading = isAdding || isUpdating;
  const isEditMode = mode === 'edit' && initialData;

  const form = useForm({
    defaultValues: {
      cidr: '',
      rule_type: 'allow' as IPRuleType,
      description: '',
      priority: 100,
    } as IPRuleFormValues,
    validators: {
      onSubmit: ipRuleSchema,
    },
    onSubmit: async ({ value }) => {
      if (isEditMode) {
        await updateRule({
          id: initialData.id,
          groupId,
          data: {
            cidr: value.cidr,
            rule_type: value.rule_type,
            description: value.description || undefined,
            priority: value.priority,
          },
        });
      } else {
        await addRule({
          groupId,
          data: {
            cidr: value.cidr,
            rule_type: value.rule_type,
            description: value.description || undefined,
            priority: value.priority,
          },
        });
      }
      onOpenChange(false);
    },
  });

  useEffect(() => {
    if (isEditMode) {
      form.setFieldValue('cidr', initialData.cidr);
      form.setFieldValue('rule_type', initialData.rule_type);
      form.setFieldValue('description', initialData.description || '');
      form.setFieldValue('priority', initialData.priority);
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
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Network className="size-5" />
            {isEditMode ? 'Edit IP Rule' : 'Add IP Rule'}
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
            <FieldGroup>
              <form.Field name="cidr">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>IP Address / CIDR</FieldLabel>
                      <Input
                        id={field.name}
                        placeholder="e.g., 192.168.1.0/24 or 10.0.0.1"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                      />
                      <FieldDescription>
                        Single IP or CIDR notation (e.g., 192.168.1.0/24)
                      </FieldDescription>
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <form.Field name="rule_type">
                {(field) => (
                  <Field>
                    <FieldLabel>Rule Type</FieldLabel>
                    <Select
                      value={field.state.value}
                      onValueChange={(val) => field.handleChange(val as IPRuleType)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="allow">Allow - Grant access to this IP</SelectItem>
                        <SelectItem value="deny">Deny - Block access from this IP</SelectItem>
                        <SelectItem value="bypass">Bypass - Skip other auth methods</SelectItem>
                      </SelectContent>
                    </Select>
                  </Field>
                )}
              </form.Field>

              <form.Field name="priority">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Priority</FieldLabel>
                      <Input
                        id={field.name}
                        type="number"
                        min={0}
                        max={1000}
                        value={field.state.value}
                        onChange={(e) => field.handleChange(parseInt(e.target.value, 10) || 0)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                      />
                      <FieldDescription>
                        Lower numbers are evaluated first (0-1000)
                      </FieldDescription>
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
                        placeholder="e.g., Office network"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        rows={2}
                        aria-invalid={hasError}
                      />
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>
            </FieldGroup>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Add Rule'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface IPRulesTabProps {
  groupId: number;
}

export function IPRulesTab({ groupId }: IPRulesTabProps) {
  const { rules, isLoading } = useIPRules(groupId);
  const { deleteRule, isDeleting } = useDeleteIPRule();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<ACLIPRule | null>(null);
  const [deletingRule, setDeletingRule] = useState<ACLIPRule | null>(null);

  const handleDelete = async () => {
    if (!deletingRule) return;
    await deleteRule({ id: deletingRule.id, groupId });
    setDeletingRule(null);
  };

  // Sort rules by priority
  const sortedRules = [...rules].sort((a, b) => a.priority - b.priority);

  return (
    <>
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle className="flex items-center gap-2">
              <Network className="size-5" />
              IP Rules
            </CardTitle>
            <CardDescription>
              Control access based on client IP addresses. Rules are evaluated in priority order.
            </CardDescription>
          </CardHeading>
          <CardToolbar>
            <Button onClick={() => setCreateModalOpen(true)}>
              <Plus className="size-4" />
              Add Rule
            </Button>
          </CardToolbar>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="h-5 w-32" />
                  <Skeleton className="h-6 w-16" />
                  <Skeleton className="h-5 w-48 flex-1" />
                  <Skeleton className="h-8 w-16" />
                </div>
              ))}
            </div>
          ) : sortedRules.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Network className="size-12 mx-auto mb-4 opacity-50" />
              <p>No IP rules configured</p>
              <p className="text-sm mt-1">
                Add IP rules to control access based on client addresses
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">Priority</TableHead>
                  <TableHead>CIDR</TableHead>
                  <TableHead className="w-24">Type</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="w-32">Created</TableHead>
                  <TableHead className="w-20 text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedRules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell>
                      <Badge variant="outline" className="font-mono">
                        {rule.priority}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <code className="text-sm bg-muted px-2 py-0.5 rounded">{rule.cidr}</code>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getRuleTypeBadgeVariant(rule.rule_type)}>
                        {getRuleTypeLabel(rule.rule_type)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {rule.description ? (
                        <span className="text-sm text-muted-foreground">{rule.description}</span>
                      ) : (
                        <span className="text-sm text-muted-foreground/50">-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="text-sm text-muted-foreground cursor-default">
                            {format(new Date(rule.created_at), 'MMM d, yyyy')}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>{format(new Date(rule.created_at), 'PPpp')}</TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8 p-0"
                              onClick={() => setEditingRule(rule)}
                            >
                              <Pencil className="size-4" />
                              <span className="sr-only">Edit</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Edit rule</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8 p-0 text-destructive hover:text-destructive"
                              onClick={() => setDeletingRule(rule)}
                            >
                              <Trash2 className="size-4" />
                              <span className="sr-only">Delete</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Delete rule</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <IPRuleFormModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        groupId={groupId}
        mode="create"
      />

      {editingRule && (
        <IPRuleFormModal
          open={!!editingRule}
          onOpenChange={(open) => !open && setEditingRule(null)}
          groupId={groupId}
          mode="edit"
          initialData={editingRule}
        />
      )}

      <AlertDialog open={!!deletingRule} onOpenChange={(open) => !open && setDeletingRule(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete IP Rule</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the rule for{' '}
              <code className="bg-muted px-1 rounded">{deletingRule?.cidr}</code>? This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
