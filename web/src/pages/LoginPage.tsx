import { FormEvent, useEffect, useState } from "react";
import { Navigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../auth";
import { AuthConfig, DEV_LOGIN_EMAIL, client, devLoginUrl } from "../api";

export default function LoginPage() {
  const { status } = useAuth();
  const [searchParams] = useSearchParams();
  const next = searchParams.get("next") || "/";
  const [config, setConfig] = useState<AuthConfig | null>(null);
  const [configError, setConfigError] = useState<string | null>(null);
  const [email, setEmail] = useState(DEV_LOGIN_EMAIL);

  useEffect(() => {
    client
      .authConfig()
      .then(setConfig)
      .catch((err) => setConfigError(err instanceof Error ? err.message : "Failed to load sign-in options"));
  }, []);

  if (status === "loading") {
    return (
      <div className="login-shell">
        <p className="muted">Checking session...</p>
      </div>
    );
  }

  if (status === "authenticated") {
    return <Navigate to={next} replace />;
  }

  function handleDevLogin(e: FormEvent) {
    e.preventDefault();
    window.location.href = devLoginUrl(email.trim() || DEV_LOGIN_EMAIL);
  }

  return (
    <div className="login-shell">
      <div className="login-card card">
        <div className="login-brand">
          <h1>MCP Guard</h1>
          <p className="muted">Sign in to manage your MCP gateway</p>
        </div>

        {configError && (
          <div className="error-banner">
            <strong>Could not load sign-in options</strong>
            <p className="muted">{configError}</p>
          </div>
        )}

        {config?.dev_login_enabled && (
          <form className="dev-login-form" onSubmit={handleDevLogin}>
            <label className="field-label" htmlFor="dev-email">
              Email
            </label>
            <input
              id="dev-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@mcpguard.local"
              required
            />
            <button type="submit" className="btn">
              Sign in
            </button>
          </form>
        )}

        {!config && !configError && <p className="muted">Loading...</p>}

        {config && !config.dev_login_enabled && (
          <p className="muted">
            Sign-in is disabled. Enable <code>AUTH_DEV_MODE</code> on the server.
          </p>
        )}
      </div>
    </div>
  );
}
