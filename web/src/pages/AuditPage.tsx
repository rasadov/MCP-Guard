import { useMemo, useState, useEffect } from "react";
import { useAuth } from "../auth";
import { useApiQuery } from "../useApiQuery";
import { RequestState } from "../RequestState";
import { AuditLog, client } from "../api";

export default function AuditPage() {
  const { status } = useAuth();
  const enabled = status === "authenticated";
  const [outcome, setOutcome] = useState<"all" | "allowed" | "denied" | "error">("all");

  const { data: logs, loading, error, unauthorized, refetch } = useApiQuery(
    "audit",
    () => client.audit(),
    enabled
  );

  useEffect(() => {
    if (!enabled) return;
    const id = window.setInterval(() => refetch(), 5000);
    return () => window.clearInterval(id);
  }, [enabled, refetch]);

  const filtered = useMemo(() => {
    const items = logs ?? [];
    if (outcome === "all") return items;
    return items.filter((log) => log.outcome === outcome);
  }, [logs, outcome]);

  return (
    <RequestState
      loading={status === "loading" || loading}
      unauthorized={status === "unauthenticated" || unauthorized}
      error={error}
    >
      <div className="page">
        <header className="page-header">
          <div>
            <h2>Audit Log</h2>
            <p className="muted">Every MCP tool call through the gateway</p>
          </div>
          <div className="btn-row">
            <a className="btn btn-secondary" href="/api/v1/audit/export?format=json">
              Export JSON
            </a>
            <a className="btn btn-secondary" href="/api/v1/audit/export?format=csv">
              Export CSV
            </a>
          </div>
        </header>

        <div className="filter-row">
          {(["all", "allowed", "denied", "error"] as const).map((value) => (
            <button
              key={value}
              className={`filter-chip ${outcome === value ? "active" : ""}`}
              onClick={() => setOutcome(value)}
            >
              {value}
            </button>
          ))}
        </div>

        <section className="card">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Tool</th>
                <th>Action</th>
                <th>Outcome</th>
                <th>Reason</th>
                <th>Latency</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((log: AuditLog) => (
                <tr key={log.id}>
                  <td>{new Date(log.created_at).toLocaleString()}</td>
                  <td>{log.tool_name}</td>
                  <td>{log.action}</td>
                  <td>
                    <span className={`badge ${log.outcome}`}>{log.outcome}</span>
                  </td>
                  <td>{log.reason || "—"}</td>
                  <td>{log.latency_ms} ms</td>
                </tr>
              ))}
              {filtered.length === 0 && (
                <tr>
                  <td colSpan={6} className="muted">
                    No audit events match this filter.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </section>
      </div>
    </RequestState>
  );
}
