import { useEffect, useState } from "react";
import { client } from "../api";

export default function ToolsPage() {
  const [tools, setTools] = useState<string[]>([]);

  useEffect(() => {
    client.tools().then((r) => setTools(r.tools)).catch(console.error);
  }, []);

  return (
    <div>
      <h2>Approved Tools</h2>
      <div className="card">
        <ul>
          {tools.map((t) => (
            <li key={t}>{t}</li>
          ))}
          {tools.length === 0 && <li className="muted">No tools discovered yet.</li>}
        </ul>
      </div>
    </div>
  );
}
