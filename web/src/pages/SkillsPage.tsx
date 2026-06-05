import { useEffect, useState } from "react";
import { Skill, User, client } from "../api";

export default function SkillsPage({ user }: { user: User | null }) {
  const [skills, setSkills] = useState<Skill[]>([]);

  useEffect(() => {
    client.skills().then(setSkills).catch(console.error);
  }, []);

  if (user?.role !== "admin") {
    return <p className="muted">Admin access required to manage skills.</p>;
  }

  return (
    <div>
      <h2>Skills</h2>
      <div className="card">
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Slug</th>
              <th>Tools</th>
            </tr>
          </thead>
          <tbody>
            {skills.map((s) => (
              <tr key={s.id}>
                <td>{s.name}</td>
                <td>{s.slug}</td>
                <td>{Array.isArray(s.tools) ? s.tools.join(", ") : JSON.stringify(s.tools)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
