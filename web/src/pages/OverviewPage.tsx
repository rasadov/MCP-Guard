import { useAuth } from "../auth";
import { useApiQuery } from "../useApiQuery";
import { RequestState } from "../RequestState";
import { client, displayToolName } from "../api";
import { Link } from "react-router-dom";

export default function OverviewPage() {
  const { status } = useAuth();
  const enabled = status === "authenticated";

  const statsQuery = useApiQuery("overview-stats", () => client.stats(), enabled);
  const auditQuery = useApiQuery("overview-audit", () => client.audit(), enabled);
  const sessionsQuery = useApiQuery(
    "overview-sessions",
    () => client.activeAgents(),
    enabled
  );

  const loading =
    status === "loading" ||
    statsQuery.loading ||
    auditQuery.loading ||
    sessionsQuery.loading;
  const unauthorized =
    status === "unauthenticated" ||
    statsQuery.unauthorized ||
    auditQuery.unauthorized ||
    sessionsQuery.unauthorized;
  const error = statsQuery.error || auditQuery.error || sessionsQuery.error;

  const stats = statsQuery.data;
  const audit = (auditQuery.data ?? []).slice(0, 8);
  const sessions = sessionsQuery.data ?? [];

  return (
    <RequestState loading={loading} unauthorized={unauthorized} error={error}>
      <div className="page">
        <header className="page-header">
          <div>
            <h2>Overview</h2>
            <p className="muted">Live gateway activity and recent tool calls</p>
          </div>
          <Link className="btn btn-secondary" to="/governance">
            Manage access
          </Link>
        </header>

        <div className="stat-grid">
          <div className="stat-card">
            <span className="stat-label">Total calls</span>
            <span className="stat-value">{stats?.total_calls ?? 0}</span>
          </div>
          <div className="stat-card allowed">
            <span className="stat-label">Allowed</span>
            <span className="stat-value">{stats?.allowed_calls ?? 0}</span>
          </div>
          <div className="stat-card denied">
            <span className="stat-label">Denied</span>
            <span className="stat-value">{stats?.denied_calls ?? 0}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">Active agents</span>
            <span className="stat-value">{sessions.length}</span>
          </div>
        </div>

        <div className="split-grid">
          <section className="card">
            <div className="section-head">
              <h3>Active agents</h3>
            </div>
            {sessions.length === 0 ? (
              <p className="muted">No active MCP sessions. Connect Gemini with the demo API key.</p>
            ) : (
              <table>
                <thead>
                  <tr>
                    <th>Agent</th>
                    <th>Last seen</th>
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
            )}
          </section>

          <section className="card">
            <div className="section-head">
              <h3>Top tools</h3>
            </div>
            {(stats?.top_tools ?? []).length === 0 ? (
              <p className="muted">No tool usage yet.</p>
            ) : (
              <ul className="rank-list">
                {(stats?.top_tools ?? []).map((t) => (
                  <li key={t.tool_name ?? "unknown"}>
                    <span>{displayToolName(t.tool_name)}</span>
                    <strong>{t.count}</strong>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>

        <section className="card">
          <div className="section-head">
            <h3>Recent audit</h3>
            <Link className="muted small" to="/audit">
              View all
            </Link>
          </div>
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
                  <td>{log.reason || "—"}</td>
                </tr>
              ))}
              {audit.length === 0 && (
                <tr>
                  <td colSpan={4} className="muted">
                    No audit events yet. Ask Gemini to read Slack via MCP Guard.
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
