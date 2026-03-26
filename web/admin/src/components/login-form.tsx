"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/components/auth-provider";

export function LoginForm() {
  const { login } = useAuth();
  const [form, setForm] = useState({ email: "", password: "" });
  const [logging, setLogging] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async () => {
    if (!form.email.trim() || !form.password) return;

    setLogging(true);
    setError("");
    const ok = await login(form.email.trim(), form.password);
    setLogging(false);

    if (!ok) {
      setError("Invalid email or password");
      toast.error("Invalid email or password");
      return;
    }

    setForm({ email: "", password: "" });
  };

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="w-full max-w-sm space-y-6 p-8">
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">ClawHost Admin</h1>
          <p className="text-muted-foreground text-sm">
            Sign in with your administrator account
          </p>
        </div>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="admin-email">Email</Label>
            <Input
              id="admin-email"
              type="email"
              placeholder="admin@example.com"
              value={form.email}
              onChange={(e) => {
                setForm((current) => ({ ...current, email: e.target.value }));
                setError("");
              }}
              disabled={logging}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin-password">Password</Label>
            <Input
              id="admin-password"
              type="password"
              placeholder="Password"
              value={form.password}
              onChange={(e) => {
                setForm((current) => ({
                  ...current,
                  password: e.target.value,
                }));
                setError("");
              }}
              onKeyDown={(e) => e.key === "Enter" && handleLogin()}
              disabled={logging}
            />
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
          </div>
          <Button
            className="w-full"
            onClick={handleLogin}
            disabled={logging || !form.email.trim() || !form.password}
          >
            {logging ? "Signing In..." : "Sign In"}
          </Button>
        </div>
      </div>
    </div>
  );
}
