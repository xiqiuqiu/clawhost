"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/components/auth-provider";
import {
  listAdminPermissions,
  listAdminRoles,
  updateAdminRolePermissions,
  type AdminPermission,
  type AdminRole,
} from "@/lib/api";

type PermissionGroup = {
  resource: string;
  permissions: AdminPermission[];
};

function groupPermissions(permissions: AdminPermission[]): PermissionGroup[] {
  const grouped = new Map<string, AdminPermission[]>();
  for (const permission of permissions) {
    const current = grouped.get(permission.resource_type) || [];
    current.push(permission);
    grouped.set(permission.resource_type, current);
  }

  return Array.from(grouped.entries())
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([resource, items]) => ({
      resource,
      permissions: items.sort((a, b) => a.key.localeCompare(b.key)),
    }));
}

export default function RolesPage() {
  const { isAuthed, hasPermission } = useAuth();
  const canManageAdmins = hasPermission("admins:manage");
  const [roles, setRoles] = useState<AdminRole[]>([]);
  const [permissions, setPermissions] = useState<AdminPermission[]>([]);
  const [selectedRoleKey, setSelectedRoleKey] = useState("");
  const [draftPermissions, setDraftPermissions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const fetchRolePolicy = useCallback(async () => {
    if (!canManageAdmins) {
      setRoles([]);
      setPermissions([]);
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const [rolesRes, permissionsRes] = await Promise.all([
        listAdminRoles(),
        listAdminPermissions(),
      ]);
      const nextRoles = rolesRes.data || [];
      const nextPermissions = permissionsRes.data || [];
      setRoles(nextRoles);
      setPermissions(nextPermissions);

      const nextSelectedRole =
        nextRoles.find((role) => role.key === selectedRoleKey)?.key ||
        nextRoles[0]?.key ||
        "";
      setSelectedRoleKey(nextSelectedRole);
      const role = nextRoles.find((item) => item.key === nextSelectedRole);
      setDraftPermissions(role?.permissions || []);
    } catch (err) {
      toast.error("Failed to load roles: " + (err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [canManageAdmins, selectedRoleKey]);

  useEffect(() => {
    if (isAuthed) {
      void fetchRolePolicy();
    }
  }, [fetchRolePolicy, isAuthed]);

  const selectedRole = roles.find((role) => role.key === selectedRoleKey) || null;
  const groupedPermissions = groupPermissions(permissions);
  const isDirty =
    selectedRole !== null &&
    JSON.stringify([...draftPermissions].sort()) !==
      JSON.stringify([...(selectedRole.permissions || [])].sort());

  const togglePermission = (permissionKey: string) => {
    setDraftPermissions((current) => {
      if (current.includes(permissionKey)) {
        return current.filter((key) => key !== permissionKey);
      }
      return [...current, permissionKey].sort();
    });
  };

  const handleSave = async () => {
    if (!selectedRole) return;

    try {
      setSaving(true);
      await updateAdminRolePermissions(selectedRole.key, draftPermissions);
      toast.success("Role permissions updated");
      await fetchRolePolicy();
    } catch (err) {
      toast.error((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (!isAuthed) return null;

  if (!canManageAdmins) {
    return (
      <div className="p-4 md:p-6">
        <Card>
          <CardContent className="p-6">
            <h1 className="text-xl font-bold">Roles</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Only platform admins can manage role permissions.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-4 md:p-6 space-y-4 md:space-y-6">
      <div>
        <h1 className="text-xl md:text-2xl font-bold">Roles</h1>
        <p className="text-muted-foreground text-sm">
          Configure which actions each built-in admin role can perform
        </p>
      </div>

      {loading ? (
        <div className="text-center py-8 text-muted-foreground">Loading...</div>
      ) : roles.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">No roles available</div>
      ) : (
        <div className="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Built-in Roles</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {roles.map((role) => (
                <button
                  key={role.key}
                  type="button"
                  onClick={() => {
                    setSelectedRoleKey(role.key);
                    setDraftPermissions(role.permissions || []);
                  }}
                  className={`w-full rounded-lg border px-3 py-3 text-left transition ${
                    selectedRoleKey === role.key
                      ? "border-primary bg-primary/5"
                      : "border-border hover:bg-accent/40"
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium">{role.name}</span>
                    <Badge variant="outline">{role.scope_type}</Badge>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {role.description || role.key}
                  </p>
                </button>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between gap-3">
                <div>
                  <CardTitle className="text-base">
                    {selectedRole?.name || "Role Permissions"}
                  </CardTitle>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {selectedRole?.description || "Select a role to edit its actions."}
                  </p>
                </div>
                <Button onClick={handleSave} disabled={!selectedRole || !isDirty || saving}>
                  {saving ? "Saving..." : "Save"}
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {groupedPermissions.map((group) => (
                <div key={group.resource} className="space-y-3">
                  <div>
                    <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
                      {group.resource}
                    </h2>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    {group.permissions.map((permission) => {
                      const checked = draftPermissions.includes(permission.key);
                      const locked =
                        selectedRole?.key === "platform_admin" &&
                        permission.key === "admins:manage";

                      return (
                        <label
                          key={permission.key}
                          className={`flex items-start gap-3 rounded-lg border p-3 ${
                            checked ? "border-primary/50 bg-primary/5" : "border-border"
                          } ${locked ? "opacity-80" : ""}`}
                        >
                          <Checkbox
                            checked={checked}
                            disabled={locked}
                            onCheckedChange={() => togglePermission(permission.key)}
                          />
                          <div className="min-w-0">
                            <div className="font-medium">{permission.name}</div>
                            <div className="text-xs text-muted-foreground">
                              {permission.description || permission.key}
                            </div>
                            <div className="mt-1 text-[11px] text-muted-foreground">
                              {permission.key}
                            </div>
                          </div>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
