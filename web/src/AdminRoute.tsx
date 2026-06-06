import { type ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "./auth";

export function AdminRoute({ children }: { children: ReactNode }) {
  const { user, status } = useAuth();

  if (status === "loading") {
    return (
      <div className="login-shell">
        <p className="muted">Checking session...</p>
      </div>
    );
  }

  if (user?.role !== "admin") {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}
