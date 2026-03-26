"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
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
  listManagedAdminUsers,
  updateManagedAdminUser,
  type AdminRole,
  type ManagedAdminUser,
} from "@/lib/api";

const statusOptions = [
  { value: "active", label: "Active" },
  { value: "disabled", label: "Disabled" },
];

function formatRole(role: string, roles: AdminRole[]) {
  return roles.find((item) => item.key === role)?.name || role;
}

function formatStatus(status: string) {
  return statusOptions.find((item) => item.value === status)?.label || status;
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

export default function AdminsPage() {
  const { isAuthed, hasPermission, admin: currentAdmin } = useAuth();
  const canManageAdmins = hasPermission("admins:manage");
  const [admins, setAdmins] = useState<ManagedAdminUser[]>([]);
  const [roles, setRoles] = useState<AdminRole[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [editAdmin, setEditAdmin] = useState<ManagedAdminUser | null>(null);
  const [createForm, setCreateForm] = useState({
    email: "",
    name: "",
    password: "",
    role: "viewer",
  });
  const [editForm, setEditForm] = useState({
    name: "",
    role: "viewer",
    status: "active",
    password: "",
  });

  const fetchAdmins = useCallback(async () => {
    if (!canManageAdmins) {
      setAdmins([]);
      setRoles([]);
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const [adminsRes, rolesRes] = await Promise.all([
        listManagedAdminUsers(),
        listAdminRoles(),
      ]);
      const platformRoles = (rolesRes.data || []).filter(
        (role) => role.scope_type === "platform"
      );
      setAdmins(adminsRes.data || []);
      setRoles(platformRoles);
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
      await createManagedAdminUser(createForm);
      toast.success("Admin created");
      setShowCreate(false);
      setCreateForm({
        email: "",
        name: "",
        password: "",
        role: "viewer",
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
      });
      toast.success("Admin updated");
      setEditAdmin(null);
      setEditForm({
        name: "",
        role: "viewer",
        status: "active",
        password: "",
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
    });
  };

  const roleOptions = roles.map((role) => ({
    value: role.key,
    label: role.name,
  }));
  const createRoleDisabled = roleOptions.length === 0;

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
            Manage admin access, roles, and account status
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)} className="shrink-0">
          Add Admin
        </Button>
      </div>

      {loading ? (
        <div className="text-center py-8 text-muted-foreground">Loading...</div>
      ) : admins.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">No admin users yet</div>
      ) : (
        <>
          <div className="hidden md:block border rounded-lg">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="w-[80px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {admins.map((admin) => (
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
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="md:hidden space-y-3">
            {admins.map((admin) => (
              <Card key={admin.id}>
                <CardContent className="p-4">
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
                    </div>
                    <AdminActions onEdit={() => openEdit(admin)} />
                  </div>
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
                  setCreateForm({ ...createForm, role: value });
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
          </div>
          <DialogFooter className="flex-col sm:flex-row gap-2">
            <Button variant="outline" onClick={() => setShowCreate(false)} className="w-full sm:w-auto">
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!createForm.name || !createForm.email || !createForm.password || createRoleDisabled}
              className="w-full sm:w-auto"
            >
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
                  setEditForm({ ...editForm, role: value });
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
            <Button onClick={handleEdit} className="w-full sm:w-auto">
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
