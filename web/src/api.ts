export type AuthConfig = {
  dev_login_enabled: boolean;
};

export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
};

export type AuditLog = {
  id: string;
  tool_name: string;
  action: string;
  outcome: string;
  reason?: string;
  latency_ms: number;
  created_at: string;
};

export type Stats = {
  total_calls: number;
  allowed_calls: number;
  denied_calls: number;
  top_tools: { tool_name: string; count: number }[];
};

export const DEV_LOGIN_EMAIL = "admin@mcpguard.local";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export function isUnauthorized(error: unknown): error is ApiError {
  return error instanceof ApiError && error.status === 401;
}

export function isForbidden(error: unknown): error is ApiError {
  return error instanceof ApiError && error.status === 403;
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(body.error || res.statusText, res.status);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

async function publicFetch<T>(path: string): Promise<T> {
  const res = await fetch(path, { credentials: "include" });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(body.error || res.statusText, res.status);
  }
  return res.json();
}

export function devLoginUrl(email: string) {
  return `/auth/dev-login?email=${encodeURIComponent(email)}`;
}

export type PolicyRules = {
  deny_tools?: string[];
  deny_write_for_roles?: string[];
  write_tools?: string[];
  channel_allowlist?: Record<string, string[]>;
};

export type Policy = {
  id: string;
  name: string;
  description: string;
  rules: PolicyRules;
  enabled: boolean;
};

export type Skill = {
  id: string;
  name: string;
  slug: string;
  description: string;
  tools: string[] | string;
};

export type Agent = {
  id: string;
  name: string;
  skill_id?: string;
  skill?: Skill;
};

export type ActiveSession = {
  id: string;
  agent_id: string;
  last_seen: string;
  agent?: Agent;
};

export const client = {
  authConfig: () => publicFetch<AuthConfig>("/auth/config"),
  me: () => api<User>("/me"),
  stats: () => api<Stats>("/stats"),
  audit: () => api<AuditLog[]>("/audit?limit=50"),
  tools: () => api<{ tools: string[] }>("/tools"),
  skills: () => api<Skill[]>("/skills"),
  policies: () => api<Policy[]>("/policies"),
  agents: () => api<Agent[]>("/agents"),
  allAgents: () => api<Agent[]>("/agents?all=true"),
  activeAgents: () => api<ActiveSession[]>("/agents/active"),
  setPolicyDenyTool: (policyId: string, toolName: string, blocked: boolean) =>
    api<Policy>(`/policies/${policyId}/deny-tools`, {
      method: "PATCH",
      body: JSON.stringify({ tool_name: toolName, blocked }),
    }),
  updateAgentSkill: (agentId: string, skillId: string | null) =>
    api<Agent>(`/agents/${agentId}`, {
      method: "PUT",
      body: JSON.stringify({ skill_id: skillId }),
    }),
  createAgent: (body: { name: string; skill_id?: string | null }) =>
    api<{ agent: Agent; api_key: string }>("/agents", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  rotateAgentKey: (agentId: string) =>
    api<{ api_key: string }>(`/agents/${agentId}/rotate-key`, {
      method: "POST",
    }),
  deleteAgent: (agentId: string) =>
    api<void>(`/agents/${agentId}`, { method: "DELETE" }),
};

export function skillTools(skill: Skill | undefined): string[] {
  if (!skill) return [];
  if (Array.isArray(skill.tools)) return skill.tools;
  try {
    return JSON.parse(skill.tools as string);
  } catch {
    return [];
  }
}

export function displayToolName(name: string | undefined | null): string {
  if (!name) return "unknown";
  return name.replace(/^slack\./, "").replace(/_/g, " ");
}

export function policyDenyTools(policy: Policy | undefined): string[] {
  return policy?.rules?.deny_tools ?? [];
}
