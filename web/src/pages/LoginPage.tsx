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
      .catch((err) => setConfigError(err instanceof Error ? err.message : "Failed to load auth options"));
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

        {config?.google_enabled && (
          <a className="btn btn-google" href="/auth/google">
            Sign in with Google
          </a>
        )}

        {config?.dev_login_enabled && (
          <form className="dev-login-form" onSubmit={handleDevLogin}>
            {config?.google_enabled && <div className="login-divider">or</div>}
            <label className="field-label" htmlFor="dev-email">
              Dev login email
            </label>
            <input
              id="dev-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@mcpguard.local"
              required
            />
            <button type="submit" className="btn btn-secondary">
              Continue with dev login
            </button>
          </form>
        )}

        {!config && !configError && <p className="muted">Loading sign-in options...</p>}

        {config && !config.google_enabled && !config.dev_login_enabled && (
          <p className="muted">
            No sign-in methods are configured. Set Google OAuth credentials or enable{" "}
            <code>AUTH_DEV_MODE</code>.
          </p>
        )}
      </div>
    </div>
  );
}
