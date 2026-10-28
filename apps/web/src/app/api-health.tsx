import { getApiHealth } from "../lib/api";

export default async function ApiHealth() {
  const health = await getApiHealth();

  return (
    <div className="mt-8 flex items-center gap-3 rounded-xl border border-slate-800 bg-slate-950 p-4">
      <span
        aria-hidden="true"
        className={`h-2.5 w-2.5 rounded-full ${health ? "bg-emerald-400" : "bg-amber-400"}`}
      />
      <div>
        <p role="status" className="text-sm font-medium text-slate-200">
          {health ? "go-zero API 已连接" : "go-zero API 未连接"}
        </p>
        <p className="mt-1 text-xs text-slate-500">
          {health
            ? `${health.service} · ${health.status}`
            : "请启动 services/api 后刷新页面。"}
        </p>
      </div>
    </div>
  );
}
