import { NavLink } from "react-router-dom";

const NAV_ITEMS = [
  { to: "/", label: "Overview", end: true },
  { to: "/auto-healer", label: "Pod Auto-Healer" },
  { to: "/rollout-manager", label: "Rollout Manager" },
  { to: "/resource-optimizer", label: "Resource Optimizer" },
  { to: "/autoscaler", label: "Auto-Scaling Controller" },
  { to: "/events", label: "Live Events" },
];

export default function Sidebar() {
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
    </nav>
  );
}
