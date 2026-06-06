import { NavLink, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./auth";
import { ProtectedRoute } from "./ProtectedRoute";
import { AdminRoute } from "./AdminRoute";
import OverviewPage from "./pages/OverviewPage";
import GovernancePage from "./pages/GovernancePage";
import AuditPage from "./pages/AuditPage";
import AgentsPage from "./pages/AgentsPage";
import LoginPage from "./pages/LoginPage";

function AppShell() {
  const { user, status } = useAuth();
  const isAdmin = user?.role === "admin";

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
          <NavLink to="/agents">Agents</NavLink>
          <NavLink to="/audit">Audit Log</NavLink>
          {isAdmin && (
            <>
              <div className="nav-divider">Admin</div>
              <NavLink to="/governance">Governance</NavLink>
            </>
          )}
        </nav>
        <div className="sidebar-footer">
          {status === "loading" ? (
            <p className="muted">Checking session...</p>
          ) : user ? (
            <div className="user-chip">
              <strong>{user.name}</strong>
              <span className="muted">{user.role}</span>
              <a className="sign-out-link" href="/auth/logout">
                Sign out
              </a>
            </div>
          ) : null}
        </div>
      </aside>
      <main className="content-wrap">
        <div className="content">
          <Routes>
            <Route path="/" element={<OverviewPage />} />
            <Route path="/agents" element={<AgentsPage />} />
            <Route path="/audit" element={<AuditPage />} />
            <Route
              path="/governance"
              element={
                <AdminRoute>
                  <GovernancePage />
                </AdminRoute>
              }
            />
          </Routes>
        </div>
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/*"
          element={
            <ProtectedRoute>
              <AppShell />
            </ProtectedRoute>
          }
        />
      </Routes>
    </AuthProvider>
  );
}
