import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  CardToolbar,
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
} from '@e412/titanium';
import { Link } from '@tanstack/react-router';
import { ExternalLink, Plus, Shield, Trash2, X } from 'lucide-react';
import { useState } from 'react';

import { useACLGroups } from '@/hooks';
import type { ACLGroup } from '@/types/acl';

export interface ACLAssignment {
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
}

interface ACLSelectorProps {
  /** Current ACL assignments */
  value: ACLAssignment[];
  /** Called when assignments change */
  onChange: (assignments: ACLAssignment[]) => void;
  /** Whether the selector is disabled */
  disabled?: boolean;
}

function getCombinationModeLabel(mode: string): string {
  switch (mode) {
    case 'any':
      return 'Any Match';
    case 'all':
      return 'All Required';
    case 'ip_bypass':
      return 'IP Bypass';
    default:
      return mode;
  }
}

export function ACLSelector({ value, onChange, disabled }: ACLSelectorProps) {
  const { groups, isLoading } = useACLGroups({ limit: 100 });
  const [showAddForm, setShowAddForm] = useState(false);
  const [selectedGroupId, setSelectedGroupId] = useState<string>('');
  const [pathPattern, setPathPattern] = useState('/*');
  const [priority, setPriority] = useState(100);

  // Get group details by ID
  const getGroup = (groupId: number): ACLGroup | undefined => {
    return groups.find((g) => g.id === groupId);
  };

  // Get groups that haven't been assigned yet (or show all if path patterns differ)
  const availableGroups = groups;

  const handleAddAssignment = () => {
    if (!selectedGroupId) return;

    const newAssignment: ACLAssignment = {
      acl_group_id: parseInt(selectedGroupId, 10),
      path_pattern: pathPattern || '/*',
      priority,
      enabled: true,
    };

    onChange([...value, newAssignment]);

    // Reset form
    setSelectedGroupId('');
    setPathPattern('/*');
    setPriority(100);
    setShowAddForm(false);
  };

  const handleRemoveAssignment = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };

  const handleToggleEnabled = (index: number) => {
    const updated = [...value];
    updated[index] = { ...updated[index], enabled: !updated[index].enabled };
    onChange(updated);
  };

  return (
    <Card>
      <CardHeader>
        <CardHeading>
          <CardTitle className="flex items-center gap-2">
            <Shield className="size-5" />
            Access Control
          </CardTitle>
          <CardDescription>
            Assign ACL groups to protect this proxy with authentication rules
          </CardDescription>
        </CardHeading>
        <CardToolbar>
          {!showAddForm && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setShowAddForm(true)}
              disabled={disabled || groups.length === 0}
            >
              <Plus className="size-4" />
              Add ACL
            </Button>
          )}
        </CardToolbar>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : groups.length === 0 ? (
          <div className="text-center py-6 text-muted-foreground">
            <Shield className="size-10 mx-auto mb-3 opacity-50" />
            <p className="text-sm">No ACL groups available</p>
            <p className="text-xs mt-1">
              <Link
                to="/dashboard/acl"
                className="text-primary hover:underline inline-flex items-center gap-1"
              >
                Create an ACL group first
                <ExternalLink className="size-3" />
              </Link>
            </p>
          </div>
        ) : (
          <>
            {/* Current Assignments */}
            {value.length > 0 && (
              <div className="space-y-2">
                {value.map((assignment, index) => {
                  const group = getGroup(assignment.acl_group_id);
                  if (!group) return null;

                  return (
                    <div
                      key={`${assignment.acl_group_id}-${assignment.path_pattern}-${index}`}
                      className={`flex items-center justify-between p-3 rounded border ${
                        assignment.enabled ? 'bg-muted/30' : 'bg-muted/10 opacity-60'
                      }`}
                    >
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="flex items-center justify-center size-8 rounded bg-primary/10 flex-shrink-0">
                          <Shield className="size-4 text-primary" />
                        </div>
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="font-medium truncate">{group.name}</span>
                            <Badge variant="outline" className="text-xs flex-shrink-0">
                              {getCombinationModeLabel(group.combination_mode)}
                            </Badge>
                            {!assignment.enabled && (
                              <Badge variant="secondary" className="text-xs flex-shrink-0">
                                Disabled
                              </Badge>
                            )}
                          </div>
                          <div className="flex items-center gap-2 text-xs text-muted-foreground mt-0.5">
                            <code className="bg-muted px-1 rounded">{assignment.path_pattern}</code>
                            <span>Priority: {assignment.priority}</span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1 flex-shrink-0">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="size-8 p-0"
                          onClick={() => handleToggleEnabled(index)}
                          disabled={disabled}
                        >
                          {assignment.enabled ? (
                            <X className="size-4 text-muted-foreground" />
                          ) : (
                            <Shield className="size-4 text-muted-foreground" />
                          )}
                          <span className="sr-only">
                            {assignment.enabled ? 'Disable' : 'Enable'}
                          </span>
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="size-8 p-0 text-destructive hover:text-destructive"
                          onClick={() => handleRemoveAssignment(index)}
                          disabled={disabled}
                        >
                          <Trash2 className="size-4" />
                          <span className="sr-only">Remove</span>
                        </Button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Add Form */}
            {showAddForm && (
              <div className="p-4 rounded border border-dashed bg-muted/20 space-y-4">
                <Field>
                  <FieldLabel>ACL Group</FieldLabel>
                  <Select value={selectedGroupId} onValueChange={setSelectedGroupId}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select an ACL group..." />
                    </SelectTrigger>
                    <SelectContent>
                      {availableGroups.map((group) => (
                        <SelectItem key={group.id} value={String(group.id)}>
                          <div className="flex items-center gap-2">
                            <Shield className="size-4 text-muted-foreground" />
                            <span>{group.name}</span>
                            <Badge variant="outline" className="text-xs ml-auto">
                              {getCombinationModeLabel(group.combination_mode)}
                            </Badge>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FieldDescription>
                    The ACL group containing authentication rules to apply
                  </FieldDescription>
                </Field>

                <div className="grid gap-4 sm:grid-cols-2">
                  <Field>
                    <FieldLabel>Path Pattern</FieldLabel>
                    <Input
                      value={pathPattern}
                      onChange={(e) => setPathPattern(e.target.value)}
                      placeholder="/*"
                    />
                    <FieldDescription>
                      URL path pattern to protect (e.g., /*, /api/*, /admin)
                    </FieldDescription>
                  </Field>

                  <Field>
                    <FieldLabel>Priority</FieldLabel>
                    <Input
                      type="number"
                      value={priority}
                      onChange={(e) => setPriority(parseInt(e.target.value, 10) || 100)}
                      min={0}
                      max={1000}
                    />
                    <FieldDescription>Lower numbers are evaluated first (0-1000)</FieldDescription>
                  </Field>
                </div>

                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setShowAddForm(false);
                      setSelectedGroupId('');
                      setPathPattern('/*');
                      setPriority(100);
                    }}
                  >
                    Cancel
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    onClick={handleAddAssignment}
                    disabled={!selectedGroupId}
                  >
                    Add Assignment
                  </Button>
                </div>
              </div>
            )}

            {/* Empty state when no assignments */}
            {value.length === 0 && !showAddForm && (
              <div className="text-center py-4 text-muted-foreground">
                <p className="text-sm">No access control configured</p>
                <p className="text-xs mt-1">This proxy will be publicly accessible</p>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
