import type { ACLAssignment } from '@/components/proxy/forms/acl-selector';

/**
 * The subset of a persisted ACL assignment `diffAclAssignments` needs to
 * detect adds/updates/removes against a fresh `ACLAssignment[]` from
 * ACLSelector. `ProxyACLAssignment` (types/acl.ts) and
 * `ProxyGroupAclAssignment` (types/proxy-group.ts) both satisfy this shape
 * structurally — no adapter needed at either call site.
 */
export interface ExistingAclAssignment {
  id: number;
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
}

export interface AclAssignmentUpdate {
  assignmentId: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
}

export interface AclAssignmentDiff {
  added: ACLAssignment[];
  updated: AclAssignmentUpdate[];
  /** acl_group_id of assignments to remove — both the proxy and proxy-group
   * ACL DELETE endpoints key off acl_group_id, not the assignment id. */
  removed: number[];
}

/**
 * Diffs a persisted set of ACL assignments (proxy- or group-scoped) against
 * a fresh ACLSelector value, keyed on acl_group_id (a proxy/group can only
 * have one assignment per ACL group).
 *
 * `enabled = false` on a matching acl_group_id is the one documented way a
 * proxy opts out of an ACL inherited from its group — without the `updated`
 * branch here, toggling `enabled` on an existing assignment would silently
 * not persist (it would look identical to "no change" to an add/remove-only
 * diff, since the id is present on both sides).
 */
export function diffAclAssignments(
  prev: ExistingAclAssignment[],
  next: ACLAssignment[],
): AclAssignmentDiff {
  const nextIds = new Set(next.map((a) => a.acl_group_id));
  const removed = prev.filter((p) => !nextIds.has(p.acl_group_id)).map((p) => p.acl_group_id);

  const added: ACLAssignment[] = [];
  const updated: AclAssignmentUpdate[] = [];

  for (const assignment of next) {
    const match = prev.find((p) => p.acl_group_id === assignment.acl_group_id);
    if (!match) {
      added.push(assignment);
      continue;
    }
    if (
      match.enabled !== assignment.enabled ||
      match.priority !== assignment.priority ||
      match.path_pattern !== assignment.path_pattern
    ) {
      updated.push({
        assignmentId: match.id,
        path_pattern: assignment.path_pattern,
        priority: assignment.priority,
        enabled: assignment.enabled,
      });
    }
  }

  return { added, updated, removed };
}
