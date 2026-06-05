import { NavLink, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./auth";
import { DEV_LOGIN_URL } from "./api";
import OverviewPage from "./pages/OverviewPage";
import GovernancePage from "./pages/GovernancePage";
import AuditPage from "./pages/AuditPage";

function AppShell() {
  const { user, status } = useAuth();

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="brand">
          <h1>MCP Guard</h1>
          <p className="brand-tag">Secure MCP Gateway</p>
        </div>
        <nav>
          <NavLink to="/" end>
            Overview
          </NavLink>
          <NavLink to="/governance">Governance</NavLink>
          <NavLink to="/audit">Audit Log</NavLink>
        </nav>
        <div className="sidebar-footer">
          {status === "loading" ? (
            <p className="muted">Checking session...</p>
          ) : user ? (
            <div className="user-chip">
              <strong>{user.name}</strong>
              <span className="muted">{user.role}</span>
            </div>
          ) : (
            <a className="btn btn-sm" href={DEV_LOGIN_URL}>
              Dev login
            </a>
          )}
        </div>
      </aside>
      <main className="content-wrap">
        <div className="content">
          <Routes>
            <Route path="/" element={<OverviewPage />} />
            <Route path="/governance" element={<GovernancePage />} />
            <Route path="/audit" element={<AuditPage />} />
          </Routes>
        </div>
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <AppShell />
    </AuthProvider>
  );
}
