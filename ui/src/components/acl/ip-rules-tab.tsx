import { zodResolver } from '@hookform/resolvers/zod';
import { format } from 'date-fns';
import { Network, Pencil, Plus, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { getRuleTypeLabel } from '@/components/acl/access-labels';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardAction, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectGroup,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useAddIPRule, useDeleteIPRule, useIPRules, useUpdateIPRule } from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';
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

const RULE_TYPES = [
  { value: 'allow', label: 'Allow', description: 'grant access to this IP' },
  { value: 'deny', label: 'Block', description: 'deny access from this IP' },
  { value: 'bypass', label: 'Trusted', description: 'skips other auth for matching IPs' },
] as const;

function getRuleTypeBadgeVariant(
  type: IPRuleType,
): 'default' | 'secondary' | 'outline' | 'destructive' {
  switch (type) {
    case 'allow':
      return 'default';
    case 'deny':
      return 'destructive';
    case 'bypass':
      return 'secondary';
    default:
      return 'outline';
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

  const form = useForm<IPRuleFormValues>({
    resolver: zodResolver(ipRuleSchema),
    defaultValues: {
      cidr: '',
      rule_type: 'allow' as IPRuleType,
      description: '',
      priority: 100,
    },
  });

  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      form.reset({
        cidr: initialData.cidr,
        rule_type: initialData.rule_type,
        description: initialData.description ?? '',
        priority: initialData.priority,
      });
    } else {
      form.reset({
        cidr: '',
        rule_type: 'allow',
        description: '',
        priority: 100,
      });
    }
  }, [open, isEditMode, initialData, form]);

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
    }
    onOpenChange(isOpen);
  };

  const onSubmit = async (value: IPRuleFormValues) => {
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
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              <div className="grid gap-4">
                <FormField
                  control={form.control}
                  name="cidr"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>IP Address / CIDR</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g., 192.168.1.0/24 or 10.0.0.1" {...field} />
                      </FormControl>

                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="rule_type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Action</FormLabel>
                      <Select items={RULE_TYPES} value={field.value} onValueChange={field.onChange}>
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectGroup>
                            {RULE_TYPES.map((option) => (
                              <SelectItem key={option.value} value={option.value}>
                                {`${option.label} — ${option.description}`}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>

                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="priority"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Priority</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          min={0}
                          max={1000}
                          value={field.value ?? ''}
                          onChange={(e) =>
                            field.onChange(e.target.value === '' ? 0 : e.target.valueAsNumber || 0)
                          }
                          onBlur={field.onBlur}
                        />
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
                        <Textarea placeholder="e.g., Office network" rows={2} {...field} />
                      </FormControl>
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
                {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Add Rule'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
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
  const { canUpdateAccess, canDeleteAccess } = usePermissions();

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
          <CardTitle className="flex items-center gap-2">
            <Network className="size-5" />
            IP Rules
          </CardTitle>

          {canUpdateAccess && (
            <CardAction>
              <Button onClick={() => setCreateModalOpen(true)}>
                <Plus className="size-4" />
                Add Rule
              </Button>
            </CardAction>
          )}
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
                  {(canUpdateAccess || canDeleteAccess) && (
                    <TableHead className="w-20 text-right">Actions</TableHead>
                  )}
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
                        <TooltipTrigger
                          render={<span className="text-sm text-muted-foreground cursor-default" />}
                        >
                          {format(new Date(rule.created_at), 'MMM d, yyyy')}
                        </TooltipTrigger>
                        <TooltipContent>{format(new Date(rule.created_at), 'PPpp')}</TooltipContent>
                      </Tooltip>
                    </TableCell>
                    {(canUpdateAccess || canDeleteAccess) && (
                      <TableCell>
                        <div className="flex justify-end gap-1">
                          {canUpdateAccess && (
                            <Tooltip>
                              <TooltipTrigger
                                render={
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="size-8 p-0"
                                    onClick={() => setEditingRule(rule)}
                                  />
                                }
                              >
                                <Pencil className="size-4" />
                                <span className="sr-only">Edit</span>
                              </TooltipTrigger>
                              <TooltipContent>Edit rule</TooltipContent>
                            </Tooltip>
                          )}
                          {canDeleteAccess && (
                            <Tooltip>
                              <TooltipTrigger
                                render={
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="size-8 p-0 text-destructive hover:text-destructive"
                                    onClick={() => setDeletingRule(rule)}
                                  />
                                }
                              >
                                <Trash2 className="size-4" />
                                <span className="sr-only">Delete</span>
                              </TooltipTrigger>
                              <TooltipContent>Delete rule</TooltipContent>
                            </Tooltip>
                          )}
                        </div>
                      </TableCell>
                    )}
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
