import { useEffect, useState } from "react";
import { ShadowFlag, client } from "../api";

export default function ShadowPage() {
  const [flags, setFlags] = useState<ShadowFlag[]>([]);

  useEffect(() => {
    client.shadow().then(setFlags).catch(console.error);
  }, []);

  return (
    <div>
      <h2>Shadow AI Detection</h2>
      <p className="muted">
        Flags tool usage observed outside the MCP Guard gateway audit trail.
      </p>
      <div className="card">
        <table>
          <thead>
            <tr>
              <th>Agent</th>
              <th>Tool</th>
              <th>Source</th>
              <th>Message</th>
              <th>Detected</th>
            </tr>
          </thead>
          <tbody>
            {flags.map((f, i) => (
              <tr key={i}>
                <td>{f.agent_name}</td>
                <td>{f.tool_name}</td>
                <td>{f.source}</td>
                <td>{f.message}</td>
                <td>{new Date(f.detected).toLocaleString()}</td>
              </tr>
            ))}
            {flags.length === 0 && (
              <tr>
                <td colSpan={5} className="muted">
                  No shadow AI events detected.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
