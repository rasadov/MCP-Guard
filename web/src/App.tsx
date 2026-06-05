import { NavLink, Route, Routes } from "react-router-dom";
import { useEffect, useState } from "react";
import { client, User } from "./api";
import OverviewPage from "./pages/OverviewPage";
import AuditPage from "./pages/AuditPage";
import ShadowPage from "./pages/ShadowPage";
import SkillsPage from "./pages/SkillsPage";
import PoliciesPage from "./pages/PoliciesPage";
import ToolsPage from "./pages/ToolsPage";

export default function App() {
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    client
      .me()
      .then(setUser)
      .catch(() => setError("Not authenticated. Use /auth/dev-login?email=admin@mcpguard.local"));
  }, []);

  return (
    <div className="layout">
      <aside className="sidebar">
        <h1>MCP Guard</h1>
        <nav>
          <NavLink to="/" end>Overview</NavLink>
          <NavLink to="/audit">Audit</NavLink>
          <NavLink to="/shadow">Shadow AI</NavLink>
          <NavLink to="/skills">Skills</NavLink>
          <NavLink to="/policies">Policies</NavLink>
          <NavLink to="/tools">Tools</NavLink>
        </nav>
        <div style={{ marginTop: "2rem", fontSize: "0.85rem" }}>
          {user ? (
            <p>
              {user.name} <span className="muted">({user.role})</span>
            </p>
          ) : (
            <p className="muted">{error}</p>
          )}
        </div>
      </aside>
      <main className="content">
        <Routes>
          <Route path="/" element={<OverviewPage />} />
          <Route path="/audit" element={<AuditPage />} />
          <Route path="/shadow" element={<ShadowPage />} />
          <Route path="/skills" element={<SkillsPage user={user} />} />
          <Route path="/policies" element={<PoliciesPage user={user} />} />
          <Route path="/tools" element={<ToolsPage />} />
        </Routes>
      </main>
    </div>
  );
}
