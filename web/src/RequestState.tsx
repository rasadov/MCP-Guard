import { type ReactNode } from "react";
import { ApiError, DEV_LOGIN_URL } from "./api";

type RequestStateProps = {
  loading?: boolean;
  error?: ApiError | null;
  unauthorized?: boolean;
  forbidden?: boolean;
  forbiddenMessage?: string;
  children: ReactNode;
};

export function AuthRequired({
  message = "Sign in to view dashboard data.",
}: {
  message?: string;
}) {
  return (
    <div className="auth-required card">
      <h3>Sign in required</h3>
      <p className="muted">{message}</p>
      <a className="btn" href={DEV_LOGIN_URL}>
        Dev login
      </a>
    </div>
  );
}

export function RequestState({
  loading,
  error,
  unauthorized,
  forbidden,
  forbiddenMessage = "Admin access required.",
  children,
}: RequestStateProps) {
  if (loading) {
    return <p className="muted">Loading...</p>;
  }
  if (unauthorized) {
    return <AuthRequired />;
  }
  if (forbidden) {
    return (
      <div className="card">
        <p className="muted">{forbiddenMessage}</p>
      </div>
    );
  }
  if (error) {
    return (
      <div className="error-banner card">
        <strong>Request failed</strong>
        <p className="muted">{error.message}</p>
      </div>
    );
  }
  return <>{children}</>;
}
