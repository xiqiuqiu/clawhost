"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import {
  type AdminSessionPayload,
  type AdminUser,
  clearAdminSession,
  getCurrentAdmin,
  hasStoredSessionToken,
  loginAdmin,
  logoutAdmin,
} from "@/lib/api";

interface AuthContextType {
  admin: AdminUser | null;
  isAuthed: boolean;
  verifying: boolean;
  expiresAt: string | null;
  hasPermission: (permission: string) => boolean;
  roleSummary: string;
  login: (email: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType>({
  admin: null,
  isAuthed: false,
  verifying: true,
  expiresAt: null,
  hasPermission: () => false,
  roleSummary: "",
  login: async () => false,
  logout: async () => {},
});

function applySessionPayload(
  payload: AdminSessionPayload,
  setAdmin: (admin: AdminUser | null) => void,
  setExpiresAt: (expiresAt: string | null) => void,
  setIsAuthed: (value: boolean) => void
) {
  setAdmin(payload.admin);
  setExpiresAt(payload.expires_at);
  setIsAuthed(true);
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [admin, setAdmin] = useState<AdminUser | null>(null);
  const [isAuthed, setIsAuthed] = useState(false);
  const [verifying, setVerifying] = useState(true);
  const [expiresAt, setExpiresAt] = useState<string | null>(null);

  useEffect(() => {
    if (!hasStoredSessionToken()) {
      setVerifying(false);
      return;
    }

    getCurrentAdmin()
      .then((res) => {
        applySessionPayload(res.data, setAdmin, setExpiresAt, setIsAuthed);
      })
      .catch(() => {
        clearAdminSession();
        setAdmin(null);
        setExpiresAt(null);
        setIsAuthed(false);
      })
      .finally(() => {
        setVerifying(false);
      });
  }, []);

  const login = useCallback(
    async (email: string, password: string): Promise<boolean> => {
      try {
        const res = await loginAdmin(email, password);
        applySessionPayload(res.data, setAdmin, setExpiresAt, setIsAuthed);
        return true;
      } catch {
        return false;
      }
    },
    []
  );

  const logout = useCallback(async () => {
    await logoutAdmin();
    setAdmin(null);
    setExpiresAt(null);
    setIsAuthed(false);
  }, []);

  const hasPermission = useCallback(
    (permission: string) => admin?.permissions.includes(permission) ?? false,
    [admin]
  );

  const roleSummary = admin?.roles.length
    ? admin.roles.map((role) => role.replaceAll("_", " ")).join(", ")
    : "";

  return (
    <AuthContext.Provider
      value={{
        admin,
        isAuthed,
        verifying,
        expiresAt,
        hasPermission,
        roleSummary,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
