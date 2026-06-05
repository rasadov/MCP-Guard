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

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const client = {
  me: () => api<User>("/me"),
  stats: () => api<Stats>("/stats"),
  audit: () => api<AuditLog[]>("/audit?limit=50"),
  tools: () => api<{ tools: string[] }>("/tools"),
  shadow: () => api<ShadowFlag[]>("/shadow"),
  skills: () => api<Skill[]>("/skills"),
  policies: () => api<Policy[]>("/policies"),
  agents: () => api<Agent[]>("/agents"),
  activeAgents: () => api<ActiveSession[]>("/agents/active"),
};

export type ShadowFlag = {
  agent_name: string;
  tool_name: string;
  source: string;
  message: string;
  detected: string;
};

export type Skill = {
  id: string;
  name: string;
  slug: string;
  description: string;
  tools: string[];
};

export type Policy = {
  id: string;
  name: string;
  description: string;
  rules: Record<string, unknown>;
  enabled: boolean;
};

export type Agent = {
  id: string;
  name: string;
  skill_id?: string;
};

export type ActiveSession = {
  id: string;
  agent_id: string;
  last_seen: string;
  agent?: Agent;
};
