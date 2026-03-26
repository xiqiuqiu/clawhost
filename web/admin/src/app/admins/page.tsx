"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuth } from "@/components/auth-provider";
import {
  createManagedAdminUser,
  listAdminRoles,
  listApps,
  listManagedAdminUsers,
  updateManagedAdminUser,
  type AdminRole,
  type App,
  type ManagedAdminUser,
} from "@/lib/api";

const statusOptions = [
  { value: "active", label: "Active" },
  { value: "disabled", label: "Disabled" },
];

const statusFilterOptions = [
  { value: "active", label: "Active" },
  { value: "disabled", label: "Disabled" },
  { value: "all", label: "All Statuses" },
];

const scopeModeOptions = [
  { value: "platform", label: "All Apps" },
  { value: "selected_apps", label: "Selected Apps" },
];

const scopeFilterOptions = [
  { value: "all", label: "All Scopes" },
  { value: "platform", label: "All Apps" },
  { value: "selected_apps", label: "Selected Apps" },
];

type AdminFormState = {
  name: string;
  role: string;
  status: string;
  password: string;
  scope_mode: string;
  app_scope_ids: string[];
};

type ScopePresentation = {
  label: string;
  detail: string;
};

function formatRole(role: string, roles: AdminRole[]) {
  return roles.find((item) => item.key === role)?.name || role;
}

function formatStatus(status: string) {
  return statusOptions.find((item) => item.value === status)?.label || status;
}

function toggleSelection(items: string[], value: string) {
  return items.includes(value)
    ? items.filter((item) => item !== value)
    : [...items, value];
}

function getScopePresentation(admin: ManagedAdminUser, apps: App[]): ScopePresentation {
  if (admin.scope_mode !== "selected_apps") {
    return {
      label: "All Apps",
      detail: "Can access every app on the platform.",
    };
  }
  if (!admin.app_scope_ids.length) {
    return {
      label: "Selected Apps",
      detail: "No apps selected yet.",
    };
  }

  const resolvedNames = admin.app_scope_ids
    .map((appId) => apps.find((app) => app.id === appId)?.name || appId)
    .filter(Boolean);
  const previewNames = resolvedNames.slice(0, 3).join(", ");
  const remaining = resolvedNames.length > 3 ? ` +${resolvedNames.length - 3} more` : "";

  return {
    label: `${resolvedNames.length} selected ${resolvedNames.length === 1 ? "app" : "apps"}`,
    detail: `${previewNames}${remaining}`,
  };
}

function AdminActions({ onEdit }: { onEdit: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium h-8 px-2 hover:bg-accent hover:text-accent-foreground">
        ...
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AppScopeFields({
  apps,
  disabled,
  form,
  onChange,
}: {
  apps: App[];
  disabled: boolean;
  form: AdminFormState;
  onChange: (next: AdminFormState) => void;
}) {
  const [search, setSearch] = useState("");
  const forcePlatformScope = form.role === "platform_admin";
  const selectedScopeMode = forcePlatformScope ? "platform" : form.scope_mode;
  const filteredApps = apps.filter((app) =>
    app.name.toLowerCase().includes(search.trim().toLowerCase())
  );
  const selectedApps = apps.filter((app) => form.app_scope_ids.includes(app.id));

  return (
    <>
      <div className="space-y-2">
        <Label>Scope</Label>
        <Select
          value={selectedScopeMode}
          onValueChange={(value) => {
            if (!value) return;
            onChange({
              ...form,
              scope_mode: value,
              app_scope_ids: value === "selected_apps" ? form.app_scope_ids : [],
            });
          }}
          disabled={disabled || forcePlatformScope}
        >
          <SelectTrigger className="w-full h-10">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {scopeModeOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">
          {forcePlatformScope
            ? "Platform admins always have access to every app."
            : selectedScopeMode === "selected_apps"
              ? "This admin will only see and operate the apps selected below."
              : "This admin can access every app allowed by their role."}
        </p>
      </div>

      {!forcePlatformScope && selectedScopeMode === "selected_apps" ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <Label>Selected Apps</Label>
            <Badge variant="outline">
              {form.app_scope_ids.length} / {apps.length}
            </Badge>
          </div>
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search apps"
            disabled={disabled || apps.length === 0}
          />
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() =>
                onChange({
                  ...form,
                  app_scope_ids: Array.from(
                    new Set([...form.app_scope_ids, ...filteredApps.map((app) => app.id)])
                  ),
                })
              }
              disabled={disabled || filteredApps.length === 0}
            >
              Select Visible
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() =>
                onChange({
                  ...form,
                  app_scope_ids: [],
                })
              }
              disabled={disabled || form.app_scope_ids.length === 0}
            >
              Clear
            </Button>
          </div>
          <div className="max-h-40 overflow-y-auto rounded-md border p-3 space-y-3">
            {apps.length === 0 ? (
              <p className="text-sm text-muted-foreground">No apps available</p>
            ) : filteredApps.length === 0 ? (
              <p className="text-sm text-muted-foreground">No apps match this search</p>
            ) : (
              filteredApps.map((app) => (
                <label key={app.id} className="flex items-center gap-3 text-sm">
                  <Checkbox
                    checked={form.app_scope_ids.includes(app.id)}
                    onCheckedChange={() =>
                      onChange({
                        ...form,
                        app_scope_ids: toggleSelection(form.app_scope_ids, app.id),
                      })
                    }
                    disabled={disabled}
                  />
                  <span>{app.name}</span>
                </label>
              ))
            )}
          </div>
          <div className="rounded-md border border-dashed p-3">
            <p className="text-xs font-medium text-muted-foreground">Current access</p>
            {selectedApps.length === 0 ? (
              <p className="mt-2 text-sm text-muted-foreground">No apps selected yet.</p>
            ) : (
              <div className="mt-2 flex flex-wrap gap-2">
                {selectedApps.map((app) => (
                  <Badge key={app.id} variant="secondary">
                    {app.name}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : null}
    </>
  );
}

export default function AdminsPage() {
  const { isAuthed, hasPermission, admin: currentAdmin } = useAuth();
  const canManageAdmins = hasPermission("admins:manage");
  const [admins, setAdmins] = useState<ManagedAdminUser[]>([]);
  const [roles, setRoles] = useState<AdminRole[]>([]);
  const [apps, setApps] = useState<App[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [editAdmin, setEditAdmin] = useState<ManagedAdminUser | null>(null);
  const [createForm, setCreateForm] = useState({
    email: "",
    name: "",
    password: "",
    role: "viewer",
    scope_mode: "platform",
    app_scope_ids: [] as string[],
  });
  const [editForm, setEditForm] = useState<AdminFormState>({
    name: "",
    role: "viewer",
    status: "active",
    password: "",
    scope_mode: "platform",
    app_scope_ids: [],
  });
  const [filters, setFilters] = useState({
    query: "",
    scope: "all",
    status: "active",
  });

  const fetchAdmins = useCallback(async () => {
    if (!canManageAdmins) {
      setAdmins([]);
      setRoles([]);
      setApps([]);
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const [adminsRes, rolesRes, appsRes] = await Promise.all([
        listManagedAdminUsers(),
        listAdminRoles(),
        listApps(),
      ]);
      setAdmins(adminsRes.data || []);
      setRoles(rolesRes.data || []);
      setApps(appsRes.data || []);
    } catch (err) {
      toast.error("Failed to load admins: " + (err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [canManageAdmins]);

  useEffect(() => {
    if (isAuthed) {
      void fetchAdmins();
    }
  }, [fetchAdmins, isAuthed]);

  const handleCreate = async () => {
    try {
      await createManagedAdminUser({
        ...createForm,
        scope_mode: createForm.role === "platform_admin" ? "platform" : createForm.scope_mode,
        app_scope_ids:
          createForm.role === "platform_admin" || createForm.scope_mode !== "selected_apps"
            ? []
            : createForm.app_scope_ids,
      });
      toast.success("Admin created");
      setShowCreate(false);
      setCreateForm({
        email: "",
        name: "",
        password: "",
        role: "viewer",
        scope_mode: "platform",
        app_scope_ids: [],
      });
      await fetchAdmins();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleEdit = async () => {
    if (!editAdmin) return;

    try {
      await updateManagedAdminUser(editAdmin.id, {
        name: editForm.name,
        role: editForm.role,
        status: editForm.status,
        password: editForm.password || undefined,
        scope_mode: editForm.role === "platform_admin" ? "platform" : editForm.scope_mode,
        app_scope_ids:
          editForm.role === "platform_admin" || editForm.scope_mode !== "selected_apps"
            ? []
            : editForm.app_scope_ids,
      });
      toast.success("Admin updated");
      setEditAdmin(null);
      setEditForm({
        name: "",
        role: "viewer",
        status: "active",
        password: "",
        scope_mode: "platform",
        app_scope_ids: [],
      });
      await fetchAdmins();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const openEdit = (admin: ManagedAdminUser) => {
    setEditAdmin(admin);
    setEditForm({
      name: admin.name,
      role: admin.roles[0] || "viewer",
      status: admin.status,
      password: "",
      scope_mode: admin.scope_mode || "platform",
      app_scope_ids: admin.app_scope_ids || [],
    });
  };

  const roleOptions = roles.map((role) => ({
    value: role.key,
    label: role.name,
  }));
  const normalizedQuery = filters.query.trim().toLowerCase();
  const filteredAdmins = admins.filter((admin) => {
    const matchesQuery =
      normalizedQuery === "" ||
      admin.name.toLowerCase().includes(normalizedQuery) ||
      admin.email.toLowerCase().includes(normalizedQuery);
    const matchesScope =
      filters.scope === "all" || admin.scope_mode === filters.scope;
    const matchesStatus =
      filters.status === "all" || admin.status === filters.status;

    return matchesQuery && matchesScope && matchesStatus;
  });
  const createRoleDisabled = roleOptions.length === 0;
  const createNeedsApps =
    createForm.role !== "platform_admin" && createForm.scope_mode === "selected_apps";
  const editNeedsApps = editForm.role !== "platform_admin" && editForm.scope_mode === "selected_apps";
  const createFormDisabled =
    !createForm.name ||
    !createForm.email ||
    !createForm.password ||
    createRoleDisabled ||
    (createNeedsApps && createForm.app_scope_ids.length === 0);
  const editFormDisabled = editNeedsApps && editForm.app_scope_ids.length === 0;

  if (!isAuthed) return null;

  if (!canManageAdmins) {
    return (
      <div className="p-4 md:p-6">
        <Card>
          <CardContent className="p-6">
            <h1 className="text-xl font-bold">Admins</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Only platform admins can manage admin accounts.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-4 md:p-6 space-y-4 md:space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-xl md:text-2xl font-bold">Admins</h1>
          <p className="text-muted-foreground text-sm">
            Manage admin access, roles, account status, and scope
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)} className="shrink-0">
          Add Admin
        </Button>
      </div>

      <div className="rounded-lg border p-4 space-y-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end">
          <div className="space-y-2 flex-1">
            <Label htmlFor="admin-filter-query">Name or Email</Label>
            <Input
              id="admin-filter-query"
              value={filters.query}
              onChange={(e) => setFilters({ ...filters, query: e.target.value })}
              placeholder="Search admins"
            />
          </div>
          <div className="space-y-2 lg:w-56">
            <Label>Scope</Label>
            <Select
              value={filters.scope}
              onValueChange={(value) => {
                if (!value) return;
                setFilters({ ...filters, scope: value });
              }}
            >
              <SelectTrigger className="w-full h-10">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {scopeFilterOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2 lg:w-56">
            <Label>Status</Label>
            <Select
              value={filters.status}
              onValueChange={(value) => {
                if (!value) return;
                setFilters({ ...filters, status: value });
              }}
            >
              <SelectTrigger className="w-full h-10">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {statusFilterOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              setFilters({
                query: "",
                scope: "all",
                status: "active",
              })
            }
            className="lg:w-auto"
          >
            Reset Filters
          </Button>
        </div>
        <p className="text-sm text-muted-foreground">
          Showing {filteredAdmins.length} of {admins.length} admins
        </p>
      </div>

      {loading ? (
        <div className="text-center py-8 text-muted-foreground">Loading...</div>
      ) : admins.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">No admin users yet</div>
      ) : filteredAdmins.length === 0 ? (
        <div className="rounded-lg border border-dashed p-8 text-center">
          <p className="text-sm text-muted-foreground">No admins match the current filters.</p>
          <Button
            type="button"
            variant="outline"
            className="mt-4"
            onClick={() =>
              setFilters({
                query: "",
                scope: "all",
                status: "active",
              })
            }
          >
            Clear Filters
          </Button>
        </div>
      ) : (
        <>
          <div className="hidden md:block border rounded-lg">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="w-[80px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredAdmins.map((admin) => {
                  const scope = getScopePresentation(admin, apps);
                  return (
                    <TableRow key={admin.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{admin.name}</span>
                          {admin.is_bootstrap ? <Badge variant="secondary">Bootstrap</Badge> : null}
                          {currentAdmin?.id === admin.id ? <Badge variant="outline">You</Badge> : null}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm">{admin.email}</TableCell>
                      <TableCell>{formatRole(admin.roles[0] || "viewer", roles)}</TableCell>
                      <TableCell className="text-sm">
                        <div className="space-y-1">
                          <div className="font-medium">{scope.label}</div>
                          <div className="text-muted-foreground">{scope.detail}</div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={admin.status === "active" ? "default" : "secondary"}>
                          {formatStatus(admin.status)}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(admin.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell>
                        <AdminActions onEdit={() => openEdit(admin)} />
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>

          <div className="md:hidden space-y-3">
            {filteredAdmins.map((admin) => (
              <Card key={admin.id}>
                <CardContent className="p-4">
                  {(() => {
                    const scope = getScopePresentation(admin, apps);
                    return (
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-medium truncate">{admin.name}</span>
                            <Badge variant={admin.status === "active" ? "default" : "secondary"}>
                              {formatStatus(admin.status)}
                            </Badge>
                            {admin.is_bootstrap ? <Badge variant="secondary">Bootstrap</Badge> : null}
                          </div>
                          <p className="text-xs text-muted-foreground mt-1 truncate">{admin.email}</p>
                          <p className="text-xs text-muted-foreground mt-2">
                            {formatRole(admin.roles[0] || "viewer", roles)}
                          </p>
                          <p className="text-xs text-muted-foreground mt-1">{scope.label}</p>
                          <p className="text-xs text-muted-foreground mt-1">{scope.detail}</p>
                        </div>
                        <AdminActions onEdit={() => openEdit(admin)} />
                      </div>
                    );
                  })()}
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="max-w-[95vw] sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Add Admin</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Name *</Label>
              <Input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="Admin name"
              />
            </div>
            <div className="space-y-2">
              <Label>Email *</Label>
              <Input
                type="email"
                value={createForm.email}
                onChange={(e) => setCreateForm({ ...createForm, email: e.target.value })}
                placeholder="admin@example.com"
              />
            </div>
            <div className="space-y-2">
              <Label>Password *</Label>
              <Input
                type="password"
                value={createForm.password}
                onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
                placeholder="Temporary password"
              />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <Select
                value={createForm.role}
                onValueChange={(value) => {
                  if (!value) return;
                  setCreateForm({
                    ...createForm,
                    role: value,
                    scope_mode: value === "platform_admin" ? "platform" : createForm.scope_mode,
                    app_scope_ids: value === "platform_admin" ? [] : createForm.app_scope_ids,
                  });
                }}
              >
                <SelectTrigger className="w-full h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roleOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <AppScopeFields
              apps={apps}
              disabled={false}
              form={{
                name: createForm.name,
                role: createForm.role,
                status: "active",
                password: createForm.password,
                scope_mode: createForm.scope_mode,
                app_scope_ids: createForm.app_scope_ids,
              }}
              onChange={(next) =>
                setCreateForm({
                  ...createForm,
                  role: next.role,
                  scope_mode: next.scope_mode,
                  app_scope_ids: next.app_scope_ids,
                })
              }
            />
          </div>
          <DialogFooter className="flex-col sm:flex-row gap-2">
            <Button variant="outline" onClick={() => setShowCreate(false)} className="w-full sm:w-auto">
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createFormDisabled} className="w-full sm:w-auto">
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editAdmin} onOpenChange={() => setEditAdmin(null)}>
        <DialogContent className="max-w-[95vw] sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Admin</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Name</Label>
              <Input
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <Select
                value={editForm.role}
                onValueChange={(value) => {
                  if (!value) return;
                  setEditForm({
                    ...editForm,
                    role: value,
                    scope_mode: value === "platform_admin" ? "platform" : editForm.scope_mode,
                    app_scope_ids: value === "platform_admin" ? [] : editForm.app_scope_ids,
                  });
                }}
              >
                <SelectTrigger className="w-full h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roleOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <AppScopeFields apps={apps} disabled={false} form={editForm} onChange={setEditForm} />

            <div className="space-y-2">
              <Label>Status</Label>
              <Select
                value={editForm.status}
                onValueChange={(value) => {
                  if (!value) return;
                  setEditForm({ ...editForm, status: value });
                }}
              >
                <SelectTrigger className="w-full h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {statusOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>New Password</Label>
              <Input
                type="password"
                value={editForm.password}
                onChange={(e) => setEditForm({ ...editForm, password: e.target.value })}
                placeholder="Leave blank to keep current password"
              />
            </div>
          </div>
          <DialogFooter className="flex-col sm:flex-row gap-2">
            <Button variant="outline" onClick={() => setEditAdmin(null)} className="w-full sm:w-auto">
              Cancel
            </Button>
            <Button onClick={handleEdit} disabled={editFormDisabled} className="w-full sm:w-auto">
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
