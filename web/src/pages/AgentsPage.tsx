import { FormEvent, useState } from "react";
import { useAuth } from "../auth";
import { useApiQuery } from "../useApiQuery";
import { RequestState } from "../RequestState";
import { Agent, Skill, client, skillTools } from "../api";

function KeyReveal({
  apiKey,
  title,
  onClose,
}: {
  apiKey: string;
  title: string;
  onClose: () => void;
}) {
  const [copied, setCopied] = useState(false);

  async function copyKey() {
    await navigator.clipboard.writeText(apiKey);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal card key-reveal" onClick={(e) => e.stopPropagation()}>
        <h3>{title}</h3>
        <p className="muted small">
          Copy this key now. It will not be shown again. Store it securely.
        </p>
        <pre className="api-key-display">{apiKey}</pre>
        <div className="btn-row">
          <button type="button" className="btn" onClick={() => void copyKey()}>
            {copied ? "Copied" : "Copy key"}
          </button>
          <button type="button" className="btn btn-secondary" onClick={onClose}>
            Done
          </button>
        </div>
        <p className="muted small usage-snippet">
          MCP clients:{" "}
          <code>Authorization: Bearer {apiKey.slice(0, 12)}...</code>
        </p>
      </div>
    </div>
  );
}

export default function AgentsPage() {
  const { status } = useAuth();
  const enabled = status === "authenticated";
  const [name, setName] = useState("");
  const [skillId, setSkillId] = useState("");
  const [creating, setCreating] = useState(false);
  const [rotating, setRotating] = useState<string | null>(null);
  const [createError, setCreateError] = useState<string | null>(null);
  const [revealedKey, setRevealedKey] = useState<{ key: string; title: string } | null>(null);

  const agentsQuery = useApiQuery("agents-list", () => client.agents(), enabled);
  const skillsQuery = useApiQuery("agents-skills", () => client.skills(), enabled);

  const agents = agentsQuery.data ?? [];
  const skills = skillsQuery.data ?? [];
  const loading = status === "loading" || agentsQuery.loading || skillsQuery.loading;
  const unauthorized = status === "unauthenticated" || agentsQuery.unauthorized;
  const error = agentsQuery.error || skillsQuery.error;

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;

    setCreating(true);
    setCreateError(null);
    try {
      const result = await client.createAgent({
        name: trimmed,
        skill_id: skillId || null,
      });
      setName("");
      setSkillId("");
      setRevealedKey({ key: result.api_key, title: `API key for ${result.agent.name}` });
      agentsQuery.refetch();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to create agent");
    } finally {
      setCreating(false);
    }
  }

  async function handleRotate(agent: Agent) {
    if (
      !window.confirm(
        `Rotate the API key for "${agent.name}"? The current key will stop working immediately.`
      )
    ) {
      return;
    }

    setRotating(agent.id);
    try {
      const result = await client.rotateAgentKey(agent.id);
      setRevealedKey({ key: result.api_key, title: `New API key for ${agent.name}` });
    } catch (err) {
      window.alert(err instanceof Error ? err.message : "Failed to rotate key");
    } finally {
      setRotating(null);
    }
  }

  return (
    <RequestState loading={loading} unauthorized={unauthorized} error={error}>
      <div className="page">
        <header className="page-header">
          <div>
            <h2>Agents</h2>
            <p className="muted">
              Create agents and API keys for MCP clients. Keys are shown once at creation or rotation.
            </p>
          </div>
        </header>

        <section className="card">
          <h3>Create agent</h3>
          <form className="create-agent-form" onSubmit={(e) => void handleCreate(e)}>
            <div className="form-grid">
              <div>
                <label className="field-label" htmlFor="agent-name">
                  Name
                </label>
                <input
                  id="agent-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="my-agent"
                  required
                />
              </div>
              <div>
                <label className="field-label" htmlFor="agent-skill">
                  Skill (optional)
                </label>
                <select
                  id="agent-skill"
                  value={skillId}
                  onChange={(e) => setSkillId(e.target.value)}
                >
                  <option value="">No skill</option>
                  {skills.map((s: Skill) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            {createError && <p className="muted error-text">{createError}</p>}
            <button type="submit" className="btn" disabled={creating}>
              {creating ? "Creating..." : "Create agent & key"}
            </button>
          </form>
        </section>

        <section className="card">
          <div className="section-head">
            <h3>Your agents</h3>
            <span className="muted">{agents.length} registered</span>
          </div>
          <div className="agent-grid">
            {agents.map((agent) => (
              <div className="agent-card" key={agent.id}>
                <div className="agent-name">{agent.name}</div>
                <p className="muted small">
                  Skill: {agent.skill?.name ?? "None"} · {skillTools(agent.skill).length} tools
                </p>
                <button
                  type="button"
                  className="btn btn-secondary btn-sm"
                  disabled={rotating === agent.id}
                  onClick={() => void handleRotate(agent)}
                >
                  {rotating === agent.id ? "Rotating..." : "Rotate API key"}
                </button>
              </div>
            ))}
            {agents.length === 0 && (
              <p className="muted">No agents yet. Create one above to get an API key.</p>
            )}
          </div>
        </section>

        <section className="card">
          <h3>Connect an MCP client</h3>
          <p className="muted small">
            Point your MCP client at <code>/mcp</code> on this gateway and send the agent API key as a
            Bearer token:
          </p>
          <pre className="usage-block">Authorization: Bearer mcpg_&lt;your_key&gt;</pre>
        </section>
      </div>

      {revealedKey && (
        <KeyReveal
          apiKey={revealedKey.key}
          title={revealedKey.title}
          onClose={() => setRevealedKey(null)}
        />
      )}
    </RequestState>
  );
}
