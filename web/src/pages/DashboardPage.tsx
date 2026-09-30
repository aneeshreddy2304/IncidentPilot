import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import AiPanel from "../components/AiPanel";
import ProposalsPanel from "../components/ProposalsPanel";
import UsageCard from "../components/UsageCard";
import { ErrorBox, Spinner, StatusBadge } from "../components/ui";
import { timeAgo } from "../util";

export default function DashboardPage() {
  const { data, error, isLoading, dataUpdatedAt } = useQuery({ queryKey: ["overview"], queryFn: api.overview, refetchInterval: 10_000 });
  const { data: settings } = useQuery({ queryKey: ["server-settings"], queryFn: api.serverSettings });
  const assisted = settings?.mode === "assisted";
  const healthy = data && data.nodes.ready === data.nodes.total && data.pods.failed === 0 && data.deployments.available === data.deployments.total;

  return (
    <div className="mx-auto max-w-[1500px]">
      <section className="mb-6 flex flex-col justify-between gap-4 border-b border-slate-800/80 pb-6 md:flex-row md:items-end">
        <div>
          <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-emerald-400">
            <span className="h-px w-6 bg-emerald-500" /> Live cluster intelligence
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-white sm:text-3xl">Incident command center</h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-400">Monitor workload health, investigate evidence, and review AI-assisted remediation from one operational view.</p>
        </div>
        <div className="flex items-center gap-3 text-xs text-slate-500">
          <span className={`h-2 w-2 rounded-full ${healthy ? "bg-emerald-400" : "bg-amber-400"}`} />
          <span>{data ? (healthy ? "Cluster operating normally" : "Attention required") : "Connecting to cluster"}</span>
          {dataUpdatedAt > 0 && <span>· Updated {timeAgo(new Date(dataUpdatedAt).toISOString())} ago</span>}
        </div>
      </section>

      {assisted && <ProposalsPanel />}
      <div className="mb-6"><AiPanel /></div>
      {isLoading && <Spinner label="Loading live cluster telemetry..." />}
      {error != null && <ErrorBox title="Cluster telemetry unavailable" message={(error as Error).message} />}

      {data && (
        <>
          <div className="mb-6 grid grid-cols-2 gap-3 xl:grid-cols-4">
            <StatCard title="Node readiness" value={`${data.nodes.ready}/${data.nodes.total}`} sub="nodes ready" to="/resources/nodes" alert={data.nodes.ready < data.nodes.total} />
            <StatCard title="Running pods" value={`${data.pods.running}/${data.pods.total}`} sub={`${data.pods.failed} failed · ${data.pods.pending} pending`} to="/resources/pods" alert={data.pods.failed > 0 || data.pods.pending > 0} />
            <StatCard title="Namespaces" value={String(data.namespaces)} sub="active scopes" />
            <StatCard title="Deployment health" value={`${data.deployments.available}/${data.deployments.total}`} sub="fully available" to="/resources/deployments" alert={data.deployments.available < data.deployments.total} />
          </div>

          <div className="grid gap-4 xl:grid-cols-[1.1fr_1.4fr]">
            <section className="card p-5">
              <div className="mb-5">
                <p className="eyebrow">Workload state</p>
                <h2 className="mt-1 font-semibold text-white">Pod distribution</h2>
              </div>
              <PodBreakdown pods={data.pods} />
            </section>

            <section className="card p-5">
              <div className="mb-4 flex items-center justify-between">
                <div><p className="eyebrow">Latest signals</p><h2 className="mt-1 font-semibold text-white">Kubernetes warnings</h2></div>
                <Link to="/events" className="link">Open event stream</Link>
              </div>
              {data.warnings.length === 0 ? (
                <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-4 py-5 text-sm text-emerald-200">No warning events detected in the current window.</div>
              ) : (
                <ul className="divide-y divide-slate-800">
                  {data.warnings.slice(0, 6).map((w, i) => (
                    <li key={i} className="py-3 first:pt-0 last:pb-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <StatusBadge status={w.type} /><span className="text-sm font-medium text-slate-100">{w.reason}</span>
                        <span className="text-xs text-slate-500">{w.namespace}/{w.object} · {timeAgo(w.lastSeen)} ago</span>
                      </div>
                      <p className="mt-1 truncate text-sm text-slate-400" title={w.message}>{w.message}</p>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </div>
          <div className="mt-4"><UsageCard /></div>
        </>
      )}
    </div>
  );
}

function StatCard({ title, value, sub, to, alert }: { title: string; value: string; sub: string; to?: string; alert?: boolean }) {
  const body = (
    <div className={`card group relative min-h-32 overflow-hidden p-4 transition-colors ${to ? "hover:border-emerald-500/40" : ""}`}>
      <div className={`absolute inset-y-0 left-0 w-0.5 ${alert ? "bg-amber-400" : "bg-emerald-500/70"}`} />
      <div className="text-xs font-medium uppercase tracking-[0.12em] text-slate-500">{title}</div>
      <div className="mt-3 flex items-center gap-2 text-3xl font-semibold tracking-tight text-white">
        {value}<span className={`h-2 w-2 rounded-full ${alert ? "bg-amber-400" : "bg-emerald-400"}`} aria-label={alert ? "Needs attention" : "Healthy"} />
      </div>
      <div className="mt-1 text-xs text-slate-500">{sub}</div>
    </div>
  );
  return to ? <Link to={to}>{body}</Link> : body;
}

function PodBreakdown({ pods }: { pods: { running: number; pending: number; succeeded: number; failed: number; unknown: number; total: number } }) {
  const segments = [
    { label: "Running", count: pods.running, color: "bg-emerald-500" },
    { label: "Pending", count: pods.pending, color: "bg-amber-400" },
    { label: "Succeeded", count: pods.succeeded, color: "bg-sky-400" },
    { label: "Failed", count: pods.failed, color: "bg-red-500" },
    { label: "Unknown", count: pods.unknown, color: "bg-slate-500" },
  ].filter((s) => s.count > 0);
  if (pods.total === 0) return <p className="text-sm text-slate-500">No pods are reporting to this cluster.</p>;
  return (
    <div>
      <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-slate-800" aria-label="Pod status distribution">
        {segments.map((s) => <div key={s.label} className={s.color} style={{ width: `${(s.count / pods.total) * 100}%` }} title={`${s.label}: ${s.count}`} />)}
      </div>
      <div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-3">
        {segments.map((s) => (
          <div key={s.label} className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2">
            <div className="flex items-center gap-2 text-xs text-slate-500"><span className={`h-2 w-2 rounded-full ${s.color}`} />{s.label}</div>
            <div className="mt-1 text-lg font-semibold text-slate-100">{s.count}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
