import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { api } from "../api/client";
import { useNamespace, useTheme } from "../context";

const NAV_SECTIONS: { title: string; items: { label: string; to: string }[] }[] = [
  {
    title: "Operations",
    items: [
      { label: "Command center", to: "/" },
      { label: "AI investigator", to: "/assistant" },
      { label: "Review history", to: "/insights" },
      { label: "Cluster events", to: "/events" },
    ],
  },
  {
    title: "Workloads",
    items: [
      { label: "Pods", to: "/resources/pods" },
      { label: "Deployments", to: "/resources/deployments" },
      { label: "StatefulSets", to: "/resources/statefulsets" },
      { label: "DaemonSets", to: "/resources/daemonsets" },
      { label: "Jobs", to: "/resources/jobs" },
      { label: "CronJobs", to: "/resources/cronjobs" },
    ],
  },
  {
    title: "Service fabric",
    items: [
      { label: "Services", to: "/resources/services" },
      { label: "Ingresses", to: "/resources/ingresses" },
    ],
  },
  {
    title: "Configuration",
    items: [
      { label: "ConfigMaps", to: "/resources/configmaps" },
      { label: "Secrets", to: "/resources/secrets" },
      { label: "Persistent volumes", to: "/resources/persistentvolumeclaims" },
    ],
  },
  {
    title: "Platform",
    items: [
      { label: "Nodes", to: "/resources/nodes" },
      { label: "Settings", to: "/settings" },
      { label: "Documentation", to: "/docs" },
    ],
  },
];

const MOBILE_ITEMS = NAV_SECTIONS[0].items;
const NON_NAMESPACED_PATHS = new Set(["/", "/assistant", "/insights", "/settings", "/docs"]);

function isNamespaceScoped(pathname: string): boolean {
  if (NON_NAMESPACED_PATHS.has(pathname) || pathname.startsWith("/docs/")) return false;
  return pathname !== "/resources/nodes";
}

function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <img src="/logo.svg" alt="" className={compact ? "h-8 w-8" : "h-10 w-10"} />
      <div>
        <div className="text-sm font-semibold tracking-tight text-white">IncidentPilot</div>
        {!compact && <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500">Cluster intelligence</div>}
      </div>
    </div>
  );
}

export default function Layout({ children }: { children: ReactNode }) {
  const { namespace, setNamespace } = useNamespace();
  const { dark, toggle } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const showNamespace = isNamespaceScoped(location.pathname);
  const { data: namespaces } = useQuery({ queryKey: ["namespaces"], queryFn: api.namespaces, refetchInterval: 30_000 });

  return (
    <div className="flex min-h-screen bg-slate-950">
      <a href="#main-content" className="sr-only z-50 rounded-md bg-emerald-600 px-4 py-2 text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4">
        Skip to main content
      </a>
      <aside className="hidden h-screen w-64 shrink-0 flex-col border-r border-slate-800/80 bg-[#0b1120] lg:sticky lg:top-0 lg:flex">
        <button onClick={() => navigate("/")} className="px-5 py-5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500">
          <Brand />
        </button>
        <div className="mx-4 mb-4 rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2">
          <div className="flex items-center gap-2 text-xs font-medium text-emerald-300">
            <span className="h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_10px_rgba(52,211,153,.8)]" />
            Cluster connection
          </div>
          <div className="mt-1 truncate text-xs text-slate-500">kind-incidentpilot</div>
        </div>
        <nav className="flex-1 space-y-5 overflow-y-auto px-3 pb-6" aria-label="Primary navigation">
          {NAV_SECTIONS.map((section) => (
            <div key={section.title}>
              <div className="px-3 pb-1.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-slate-600">{section.title}</div>
              <div className="space-y-0.5">
                {section.items.map((item) => (
                  <NavLink key={item.to} to={item.to} end={item.to === "/"} className={({ isActive }) => `group flex min-h-9 items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${isActive ? "bg-emerald-500/10 font-medium text-emerald-300" : "text-slate-400 hover:bg-slate-800/70 hover:text-slate-100"}`}>
                    <span className="h-1.5 w-1.5 rounded-full bg-current opacity-70" />
                    {item.label}
                  </NavLink>
                ))}
              </div>
            </div>
          ))}
        </nav>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-20 flex min-h-16 items-center justify-between gap-4 border-b border-slate-800/80 bg-slate-950/90 px-4 backdrop-blur-xl sm:px-6">
          <button onClick={() => navigate("/")} className="lg:hidden" aria-label="Go to command center"><Brand compact /></button>
          <div className="hidden lg:block"><div className="text-xs font-medium uppercase tracking-[0.16em] text-slate-500">Live operations workspace</div></div>
          <div className="flex items-center gap-2 sm:gap-3">
            {showNamespace && (
              <label className="flex items-center gap-2 text-xs text-slate-500">
                <span className="hidden sm:inline">Namespace</span>
                <select className="input max-w-36" value={namespace} onChange={(e) => setNamespace(e.target.value)} aria-label="Namespace">
                  <option value="">All namespaces</option>
                  {(namespaces ?? []).map((ns) => <option key={ns} value={ns}>{ns}</option>)}
                </select>
              </label>
            )}
            <ModeBadge />
            <button className="btn-ghost" onClick={toggle} aria-label="Toggle color theme">{dark ? "Light" : "Dark"}</button>
          </div>
        </header>
        <main id="main-content" className="flex-1 p-4 pb-24 sm:p-6 sm:pb-24 xl:p-8 lg:pb-8">{children}</main>
        <MobileNav />
      </div>
    </div>
  );
}

function MobileNav() {
  return (
    <nav className="fixed inset-x-3 bottom-3 z-30 grid grid-cols-4 rounded-2xl border border-slate-700 bg-slate-900/95 p-1.5 shadow-2xl backdrop-blur-xl lg:hidden" aria-label="Mobile navigation">
      {MOBILE_ITEMS.map((item) => (
        <NavLink key={item.to} to={item.to} end={item.to === "/"} className={({ isActive }) => `flex min-h-11 items-center justify-center rounded-xl px-1 text-center text-[11px] font-medium ${isActive ? "bg-emerald-500/15 text-emerald-300" : "text-slate-400"}`}>
          {item.label.replace("Command center", "Command").replace("AI investigator", "Investigate").replace("Review history", "History").replace("Cluster events", "Events")}
        </NavLink>
      ))}
    </nav>
  );
}

function ModeBadge() {
  const { data } = useQuery({ queryKey: ["server-settings"], queryFn: api.serverSettings });
  if (!data) return null;
  const assisted = data.mode === "assisted";
  return (
    <span title={assisted ? "The agent can propose changes for approval." : "IncidentPilot observes and advises only."} className={`rounded-full border px-2.5 py-1 text-xs font-medium ${assisted ? "border-amber-500/30 bg-amber-500/10 text-amber-300" : "border-slate-700 bg-slate-800/60 text-slate-300"}`}>
      {assisted ? "Assisted" : "Read only"}
    </span>
  );
}
