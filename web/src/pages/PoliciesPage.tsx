import { useEffect, useState } from "react";
import { Policy, User, client } from "../api";

export default function PoliciesPage({ user }: { user: User | null }) {
  const [policies, setPolicies] = useState<Policy[]>([]);

  useEffect(() => {
    client.policies().then(setPolicies).catch(console.error);
  }, []);

  if (user?.role !== "admin") {
    return <p className="muted">Admin access required to manage policies.</p>;
  }

  return (
    <div>
      <h2>Policies</h2>
      {policies.map((p) => (
        <div className="card" key={p.id}>
          <h3>
            {p.name}{" "}
            <span className="muted">{p.enabled ? "(enabled)" : "(disabled)"}</span>
          </h3>
          <p>{p.description}</p>
          <pre>{JSON.stringify(p.rules, null, 2)}</pre>
        </div>
      ))}
    </div>
  );
}
