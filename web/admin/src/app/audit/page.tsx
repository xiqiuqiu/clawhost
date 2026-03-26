"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAuth } from "@/components/auth-provider";
import { listAuditLogs, type AuditLog } from "@/lib/api";

function ResultBadge({ result }: { result: string }) {
  const variant = result === "success" ? "default" : "destructive";
  return <Badge variant={variant}>{result}</Badge>;
}

export default function AuditPage() {
  const { isAuthed } = useAuth();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({
    actor: "",
    action: "",
    app_id: "",
  });

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      const res = await listAuditLogs({
        actor: filters.actor || undefined,
        action: filters.action || undefined,
        app_id: filters.app_id || undefined,
        limit: 100,
      });
      setLogs(res.data || []);
    } catch (err) {
      toast.error("Failed to load audit logs: " + (err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [filters.action, filters.actor, filters.app_id]);

  useEffect(() => {
    if (isAuthed) {
      void fetchLogs();
    }
  }, [isAuthed, fetchLogs]);

  if (!isAuthed) return null;

  return (
    <div className="p-4 md:p-6 space-y-4 md:space-y-6">
      <div className="flex flex-col gap-2">
        <h1 className="text-xl md:text-2xl font-bold">Audit</h1>
        <p className="text-muted-foreground text-sm">
          Review recent admin actions across apps and bots.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="grid gap-4 md:grid-cols-4">
            <div className="space-y-2">
              <Label htmlFor="actor-filter">Actor</Label>
              <Input
                id="actor-filter"
                placeholder="admin@example.com"
                value={filters.actor}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, actor: event.target.value }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="action-filter">Action</Label>
              <Input
                id="action-filter"
                placeholder="app.create"
                value={filters.action}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, action: event.target.value }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="app-filter">App ID</Label>
              <Input
                id="app-filter"
                placeholder="app-123"
                value={filters.app_id}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, app_id: event.target.value }))
                }
              />
            </div>
            <div className="flex items-end">
              <Button onClick={() => void fetchLogs()} className="w-full">
                Refresh
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {loading ? (
        <div className="text-center py-8 text-muted-foreground">Loading...</div>
      ) : logs.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          No audit logs matched the current filters.
        </div>
      ) : (
        <div className="border rounded-lg overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Target</TableHead>
                <TableHead>App</TableHead>
                <TableHead>Result</TableHead>
                <TableHead>IP</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="text-sm text-muted-foreground">
                    {new Date(log.created_at).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="font-medium">{log.actor_email || "Unknown"}</span>
                      <span className="text-xs text-muted-foreground">{log.actor_admin_id}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                      {log.action}
                    </code>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="font-medium">{log.target_label || log.target_id || "-"}</span>
                      <span className="text-xs text-muted-foreground">{log.target_type}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {log.app_id || "-"}
                  </TableCell>
                  <TableCell>
                    <ResultBadge result={log.result} />
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {log.source_ip || "-"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
