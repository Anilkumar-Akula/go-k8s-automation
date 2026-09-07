import { useState } from "react";
import { NavLink } from "react-router-dom";
import { getToken, setToken } from "../api/client";

const NAV_ITEMS = [
  { to: "/", label: "Overview", end: true },
  { to: "/auto-healer", label: "Pod Auto-Healer" },
  { to: "/rollout-manager", label: "Rollout Manager" },
  { to: "/resource-optimizer", label: "Resource Optimizer" },
  { to: "/autoscaler", label: "Auto-Scaling Controller" },
  { to: "/events", label: "Live Events" },
  { to: "/audit", label: "Audit History" },
];

export default function Sidebar() {
  const [token, setTokenInput] = useState(getToken());

  function save() {
    setToken(token.trim());
    window.location.reload(); // simplest way to re-auth every open connection/poll
  }

  return (
    <nav className="sidebar">
      <div className="sidebar-brand">K8s Automation</div>
      <ul className="sidebar-nav">
        {NAV_ITEMS.map((item) => (
          <li key={item.to}>
            <NavLink
              to={item.to}
              end={item.end}
              className={({ isActive }) => (isActive ? "active" : "")}
            >
              {item.label}
            </NavLink>
          </li>
        ))}
      </ul>
      <div className="sidebar-auth">
        <label htmlFor="auth-token">Auth token</label>
        <input
          id="auth-token"
          type="text"
          autoComplete="off"
          placeholder="none (auth disabled)"
          value={token}
          onChange={(e) => setTokenInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && save()}
        />
        <button onClick={save}>Save</button>
      </div>
    </nav>
  );
}
