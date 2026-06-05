import { useEffect, useState } from "react";
import { AuditLog, client } from "../api";

export default function AuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);

  useEffect(() => {
    client.audit().then(setLogs).catch(console.error);
  }, []);

  return (
    <div>
      <h2>Audit Log</h2>
      <div style={{ marginBottom: "1rem", display: "flex", gap: "0.5rem" }}>
        <a className="btn" href="/api/v1/audit/export?format=json">
          Export JSON
        </a>
        <a className="btn" href="/api/v1/audit/export?format=csv">
          Export CSV
        </a>
      </div>
      <div className="card">
        <table>
          <thead>
            <tr>
              <th>Time</th>
              <th>Tool</th>
              <th>Action</th>
              <th>Outcome</th>
              <th>Latency (ms)</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((log) => (
              <tr key={log.id}>
                <td>{new Date(log.created_at).toLocaleString()}</td>
                <td>{log.tool_name}</td>
                <td>{log.action}</td>
                <td>
                  <span className={`badge ${log.outcome}`}>{log.outcome}</span>
                </td>
                <td>{log.latency_ms}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
