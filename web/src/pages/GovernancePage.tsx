import { useMemo, useState } from "react";
import { useAuth } from "../auth";
import { useApiQuery } from "../useApiQuery";
import { AuthRequired, RequestState } from "../RequestState";
import {
  Agent,
  Policy,
  Skill,
  client,
  displayToolName,
  policyDenyTools,
  skillTools,
} from "../api";

function toolLabel(name: string) {
  return displayToolName(name);
}

function Toggle({
  allowed,
  disabled,
  onChange,
}: {
  allowed: boolean;
  disabled?: boolean;
  onChange: (allowed: boolean) => void;
}) {
  return (
    <label className={`toggle ${allowed ? "on" : "off"}`}>
      <input
        type="checkbox"
        checked={allowed}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="toggle-track">
        <span className="toggle-thumb" />
      </span>
      <span className="toggle-text">{allowed ? "Allowed" : "Blocked"}</span>
    </label>
  );
}

export default function GovernancePage() {
  const { user, status } = useAuth();
  const enabled = status === "authenticated" && user?.role === "admin";
  const [filter, setFilter] = useState("");
  const [saving, setSaving] = useState<string | null>(null);
  const [policy, setPolicy] = useState<Policy | null>(null);

  const toolsQuery = useApiQuery("gov-tools", () => client.tools(), enabled);
  const policiesQuery = useApiQuery("gov-policies", () => client.policies(), enabled);
  const agentsQuery = useApiQuery("gov-agents", () => client.agents(), enabled);
  const skillsQuery = useApiQuery("gov-skills", () => client.skills(), enabled);
  const activePolicy =
    policy ??
    policiesQuery.data?.find((p) => p.name === "default") ??
    policiesQuery.data?.[0] ??
    null;
  const tools = toolsQuery.data?.tools ?? [];
  const agents = agentsQuery.data ?? [];
  const skills = skillsQuery.data ?? [];

  const filteredTools = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return tools;
    return tools.filter((t) => t.toLowerCase().includes(q));
  }, [tools, filter]);

  const loading =
    status === "loading" ||
    toolsQuery.loading ||
    policiesQuery.loading ||
    agentsQuery.loading ||
    skillsQuery.loading;
  const unauthorized =
    status === "unauthenticated" ||
    toolsQuery.unauthorized ||
    policiesQuery.unauthorized;
  const error =
    toolsQuery.error || policiesQuery.error || agentsQuery.error || skillsQuery.error;

  if (status === "loading") return <p className="muted">Loading...</p>;
  if (status === "unauthenticated" || unauthorized) return <AuthRequired />;
  if (user?.role !== "admin") {
    return (
      <div className="page">
        <div className="card">
          <p className="muted">Admin access required to manage governance settings.</p>
        </div>
      </div>
    );
  }

  async function toggleTool(toolName: string, blocked: boolean) {
    if (!activePolicy) return;
    setSaving(toolName);
    try {
      const updated = await client.setPolicyDenyTool(activePolicy.id, toolName, blocked);
      setPolicy(updated);
    } finally {
      setSaving(null);
    }
  }

  const denyTools = policyDenyTools(activePolicy ?? undefined);

  async function assignSkill(agent: Agent, skillId: string) {
    setSaving(agent.id);
    try {
      await client.updateAgentSkill(agent.id, skillId || null);
      agentsQuery.refetch();
    } finally {
      setSaving(null);
    }
  }

  return (
    <RequestState loading={loading} error={error}>
      <div className="page">
        <header className="page-header">
          <div>
            <h2>Governance</h2>
            <p className="muted">
              Block tools globally or change an agent&apos;s skill. Changes apply on the next MCP tool call.
            </p>
          </div>
        </header>

        {!activePolicy && (
          <div className="card warn-banner">
            No policy found. Restart the gateway to restore the default policy.
          </div>
        )}

        <section className="card">
          <div className="section-head">
            <h3>Agents</h3>
            <span className="muted">{agents.length} registered</span>
          </div>
          <div className="agent-grid">
            {agents.map((agent) => (
              <div className="agent-card" key={agent.id}>
                <div className="agent-name">{agent.name}</div>
                <label className="field-label">Skill</label>
                <select
                  value={agent.skill_id ?? ""}
                  disabled={saving === agent.id}
                  onChange={(e) => void assignSkill(agent, e.target.value)}
                >
                  <option value="">No skill</option>
                  {skills.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
                <p className="muted small">
                  {skillTools(agent.skill).length} tools in skill
                </p>
              </div>
            ))}
            {agents.length === 0 && <p className="muted">No agents yet.</p>}
          </div>
        </section>

        <section className="card">
          <div className="section-head">
            <div>
              <h3>Tool Access</h3>
              <p className="muted small">
                Policy: {activePolicy?.name ?? "—"} · toggle off to block a tool for all agents
              </p>
            </div>
            <input
              className="search-input"
              placeholder="Filter tools..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>

          <div className="tool-list">
            {filteredTools.map((tool) => {
              const blocked = denyTools.includes(tool);
              return (
                <div className={`tool-row ${blocked ? "blocked" : "allowed"}`} key={tool}>
                  <div>
                    <div className="tool-name">{toolLabel(tool)}</div>
                    <div className="tool-id muted small">{tool}</div>
                  </div>
                  <Toggle
                    allowed={!blocked}
                    disabled={!activePolicy || saving === tool}
                    onChange={(allowed) => void toggleTool(tool, !allowed)}
                  />
                </div>
              );
            })}
            {filteredTools.length === 0 && (
              <p className="muted">No tools match your filter.</p>
            )}
          </div>
        </section>

        <section className="card">
          <div className="section-head">
            <h3>Skills</h3>
          </div>
          <div className="skill-grid">
            {skills.map((skill: Skill) => (
              <div className="skill-card" key={skill.id}>
                <strong>{skill.name}</strong>
                <p className="muted small">{skill.description}</p>
                <div className="chip-row">
                  {skillTools(skill).map((t) => (
                    <span className="chip" key={t}>
                      {toolLabel(t)}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>
      </div>
    </RequestState>
  );
}
