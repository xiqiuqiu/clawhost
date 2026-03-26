const API_BASE = "/bot/api/v1/admin";
const SESSION_STORAGE_KEY = "admin_session_token";

export interface AdminUser {
  id: string;
  email: string;
  name: string;
  status: string;
  is_bootstrap: boolean;
  roles: string[];
  permissions: string[];
  memberships: AdminMembership[];
  scope_mode: string;
  app_scope_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface AdminMembership {
  role: string;
  scope_type: string;
  scope_id?: string;
}

export interface AdminSessionPayload {
  admin: AdminUser;
  expires_at: string;
}

export interface ManagedAdminUser extends AdminUser {}

export interface AdminRole {
  id: string;
  key: string;
  name: string;
  description: string;
  scope_type: string;
  is_system: boolean;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface AdminPermission {
  id: string;
  key: string;
  name: string;
  description: string;
  resource_type: string;
  created_at: string;
  updated_at: string;
}

export interface AuditLog {
  id: string;
  actor_admin_id: string;
  actor_email: string;
  app_id: string;
  action: string;
  target_type: string;
  target_id: string;
  target_label: string;
  result: string;
  source_ip: string;
  user_agent: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

function getStoredSessionToken(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(SESSION_STORAGE_KEY) || "";
}

function setStoredSessionToken(token: string) {
  localStorage.setItem(SESSION_STORAGE_KEY, token);
}

function clearStoredSessionToken() {
  localStorage.removeItem(SESSION_STORAGE_KEY);
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<{ code: number; message: string; data: T }> {
  const token = getStoredSessionToken();
  const headers = new Headers(options.headers);
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  const json = await res.json();
  if (!res.ok) {
    throw new Error(json.message || `Request failed: ${res.status}`);
  }
  return json;
}

export async function loginAdmin(email: string, password: string) {
  const res = await request<
    Omit<AdminSessionPayload, "admin"> & {
      admin: Omit<AdminUser, "roles" | "permissions" | "memberships">;
      roles: string[];
      permissions: string[];
      memberships: AdminMembership[];
      session_token: string;
    }
  >(
    "/login",
    {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }
  );
  const normalized = normalizeAdminSessionPayload(res.data);
  setStoredSessionToken(res.data.session_token);
  return {
    ...res,
    data: {
      ...normalized,
      session_token: res.data.session_token,
    },
  };
}

export async function logoutAdmin() {
  try {
    await request<{ revoked: boolean }>("/logout", {
      method: "POST",
    });
  } finally {
    clearStoredSessionToken();
  }
}

export async function getCurrentAdmin() {
  const res = await request<
    Omit<AdminSessionPayload, "admin"> & {
      admin: Omit<AdminUser, "roles" | "permissions" | "memberships">;
      roles: string[];
      permissions: string[];
      memberships: AdminMembership[];
    }
  >("/me");
  return {
    ...res,
    data: normalizeAdminSessionPayload(res.data),
  };
}

export async function listManagedAdminUsers() {
  return request<ManagedAdminUser[]>("/admin-users");
}

export async function createManagedAdminUser(data: {
  email: string;
  name: string;
  password: string;
  role: string;
  scope_mode?: string;
  app_scope_ids?: string[];
}) {
  return request<ManagedAdminUser>("/admin-users", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateManagedAdminUser(
  id: string,
  data: {
    name?: string;
    status?: string;
    role?: string;
    password?: string;
    scope_mode?: string;
    app_scope_ids?: string[];
  }
) {
  return request<ManagedAdminUser>(`/admin-users/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function listAdminRoles() {
  return request<AdminRole[]>("/roles");
}

export async function listAdminPermissions() {
  return request<AdminPermission[]>("/permissions");
}

export async function updateAdminRolePermissions(
  key: string,
  permissions: string[]
) {
  return request<AdminRole>(`/roles/${key}/permissions`, {
    method: "PUT",
    body: JSON.stringify({ permissions }),
  });
}

export function hasStoredSessionToken(): boolean {
  return Boolean(getStoredSessionToken());
}

export function clearAdminSession() {
  clearStoredSessionToken();
}

// App types
export interface App {
  id: string;
  name: string;
  url: string;
  description: string;
  owner_email: string;
  api_token: string;
  bot_domain_template: string;
  status: string;
  created_at: string;
  updated_at: string;
}

// Bot types
export interface Bot {
  id: string;
  app_id: string;
  user_id: string;
  name: string;
  slug: string;
  access_token: string;
  status: "created" | "starting" | "running" | "stopped" | "error";
  config: Record<string, unknown>;
  endpoint: string;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
}

// App APIs
export async function listApps() {
  return request<App[]>("/apps");
}

export async function getApp(id: string) {
  return request<App>(`/apps/${id}`);
}

export async function createApp(data: {
  name: string;
  url?: string;
  description?: string;
  owner_email?: string;
  bot_domain_template?: string;
}) {
  return request<App>("/apps", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateApp(
  id: string,
  data: {
    name?: string;
    url?: string;
    description?: string;
    owner_email?: string;
    bot_domain_template?: string;
    status?: string;
  }
) {
  return request<App>(`/apps/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteApp(id: string) {
  return request<{ message: string }>(`/apps/${id}`, {
    method: "DELETE",
  });
}

export async function resetAppToken(id: string) {
  return request<{ api_token: string }>(`/apps/${id}/reset-token`, {
    method: "POST",
  });
}

// Bot APIs
export async function createBot(data: {
  app_id: string;
  user_id?: string;
  name: string;
  slug?: string;
}) {
  return request<Bot>("/bots", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function listBots() {
  return request<Bot[]>("/bots");
}

export async function startBot(id: string) {
  return request<Bot>(`/bots/${id}/start`, { method: "POST" });
}

export async function stopBot(id: string) {
  return request<Bot>(`/bots/${id}/stop`, { method: "POST" });
}

export async function deleteBot(id: string) {
  return request<{ message: string }>(`/bots/${id}`, { method: "DELETE" });
}

export async function upgradeBot(id: string) {
  return request<Bot>(`/bots/${id}/upgrade`, { method: "POST" });
}

export async function upgradeAllBots() {
  return request<{ message: string }>("/bots/upgrade", { method: "POST" });
}

export async function restartAllBots() {
  return request<{ message: string }>("/bots/restart", { method: "POST" });
}

// Config APIs
export async function getAdminConfig() {
  return request<{ bot_domain_template: string }>("/config");
}

export async function listAuditLogs(filters?: {
  actor?: string;
  app_id?: string;
  action?: string;
  result?: string;
  limit?: number;
}) {
  const params = new URLSearchParams();
  if (filters?.actor) params.set("actor", filters.actor);
  if (filters?.app_id) params.set("app_id", filters.app_id);
  if (filters?.action) params.set("action", filters.action);
  if (filters?.result) params.set("result", filters.result);
  if (filters?.limit) params.set("limit", String(filters.limit));

  const query = params.toString();
  return request<AuditLog[]>(`/audit${query ? `?${query}` : ""}`);
}

function normalizeAdminSessionPayload(data: {
  admin: Omit<AdminUser, "roles" | "permissions" | "memberships">;
  roles?: string[];
  permissions?: string[];
  memberships?: AdminMembership[];
  expires_at: string;
}) {
  return {
    admin: {
      ...data.admin,
      roles: data.roles || [],
      permissions: data.permissions || [],
      memberships: data.memberships || [],
      scope_mode: data.admin.scope_mode || "platform",
      app_scope_ids: data.admin.app_scope_ids || [],
    },
    expires_at: data.expires_at,
  };
}
