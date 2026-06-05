import { useEffect, useState } from "react";
import { ActiveSession, AuditLog, Stats, client } from "../api";

export default function OverviewPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [audit, setAudit] = useState<AuditLog[]>([]);
  const [sessions, setSessions] = useState<ActiveSession[]>([]);

  useEffect(() => {
    Promise.all([client.stats(), client.audit(), client.activeAgents()])
      .then(([s, a, sess]) => {
        setStats(s);
        setAudit(a);
        setSessions(sess);
      })
      .catch(console.error);
  }, []);

  return (
    <div>
      <h2>Overview</h2>
      <div className="grid">
        <div className="card">
          <div className="muted">Total Calls</div>
          <div className="stat">{stats?.total_calls ?? "-"}</div>
        </div>
        <div className="card">
          <div className="muted">Allowed</div>
          <div className="stat">{stats?.allowed_calls ?? "-"}</div>
        </div>
        <div className="card">
          <div className="muted">Denied</div>
          <div className="stat">{stats?.denied_calls ?? "-"}</div>
        </div>
        <div className="card">
          <div className="muted">Active Agents</div>
          <div className="stat">{sessions.length}</div>
        </div>
      </div>

      <div className="card">
        <h3>Active Agents</h3>
        <table>
          <thead>
            <tr>
              <th>Agent</th>
              <th>Last Seen</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map((s) => (
              <tr key={s.id}>
                <td>{s.agent?.name || s.agent_id}</td>
                <td>{new Date(s.last_seen).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="card">
        <h3>Recent Audit Feed</h3>
        <table>
          <thead>
            <tr>
              <th>Time</th>
              <th>Tool</th>
              <th>Outcome</th>
              <th>Reason</th>
            </tr>
          </thead>
          <tbody>
            {audit.map((log) => (
              <tr key={log.id}>
                <td>{new Date(log.created_at).toLocaleString()}</td>
                <td>{log.tool_name}</td>
                <td>
                  <span className={`badge ${log.outcome}`}>{log.outcome}</span>
                </td>
                <td>{log.reason || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
