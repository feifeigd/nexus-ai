"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { getToken } from "../../lib/auth";

type Tenant = { id: string; name: string };

function authHeaders(): Record<string, string> {
  const token = getToken();
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

async function readError(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as { message?: string } | string;
    if (typeof data === "string") return data;
    return data.message ?? "操作失败";
  } catch {
    return "操作失败";
  }
}

export default function TenantsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [name, setName] = useState("");
  const [memberTenant, setMemberTenant] = useState("");
  const [username, setUsername] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/tenants", { headers: authHeaders() });
      if (!res.ok) {
        setError(await readError(res));
        return;
      }
      const data = (await res.json()) as { tenants?: Tenant[] };
      setTenants(data.tenants ?? []);
    } catch {
      setError("加载租户失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // 一次性初始化：挂载时拉取租户列表。
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load();
  }, [load]);

  const createTenant = async () => {
    setError("");
    setMessage("");
    if (!name.trim()) {
      setError("请输入组织名称");
      return;
    }
    setBusy(true);
    try {
      const res = await fetch("/api/tenants", {
        method: "POST",
        headers: authHeaders(),
        body: JSON.stringify({ name: name.trim() }),
      });
      const data = (await res.json()) as { name?: string };
      if (!res.ok) {
        setError(await readError(res));
        return;
      }
      setMessage(`已创建组织「${data.name}」并切换进去`);
      setName("");
      await load();
    } catch {
      setError("创建组织失败");
    } finally {
      setBusy(false);
    }
  };

  const addMember = async () => {
    setError("");
    setMessage("");
    if (!memberTenant) {
      setError("请先选择一个组织");
      return;
    }
    if (!username.trim()) {
      setError("请输入要添加的用户名");
      return;
    }
    setBusy(true);
    try {
      const res = await fetch(`/api/tenants/${memberTenant}/members`, {
        method: "POST",
        headers: authHeaders(),
        body: JSON.stringify({ username: username.trim() }),
      });
      const data = (await res.json()) as { username?: string };
      if (!res.ok) {
        setError(await readError(res));
        return;
      }
      setMessage(`已把「${data.username}」加入组织`);
      setUsername("");
    } catch {
      setError("添加成员失败");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 p-6 text-slate-200">
      <div className="mx-auto max-w-2xl space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold text-slate-50">组织租户</h1>
          <Link href="/" className="text-sm text-slate-400 hover:text-cyan-400">
            ← 返回聊天
          </Link>
        </div>

        <div className="rounded-xl border border-slate-700 bg-slate-900 p-4">
          <h2 className="text-sm font-semibold text-slate-50">创建组织</h2>
          <p className="mt-1 text-sm text-slate-400">
            创建后你的账号会切进这个组织，可再拉成员加入共享知识库和工单。
          </p>
          <div className="mt-3 flex gap-2">
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="组织名称"
              className="flex-1 rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 outline-none focus:border-cyan-500"
            />
            <button
              onClick={() => void createTenant()}
              disabled={busy}
              className="rounded-lg bg-cyan-600 px-3 py-2 font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
            >
              创建
            </button>
          </div>
        </div>

        {message ? (
          <p className="rounded-lg border border-cyan-800 bg-cyan-950/40 px-3 py-2 text-sm text-cyan-300">
            {message}
          </p>
        ) : null}
        {error ? (
          <p className="rounded-lg border border-red-800 bg-red-950/40 px-3 py-2 text-sm text-red-400">
            {error}
          </p>
        ) : null}

        <div className="rounded-xl border border-slate-700 bg-slate-900 p-4">
          <h2 className="text-sm font-semibold text-slate-50">我的租户</h2>
          {loading ? (
            <p className="mt-2 text-sm text-slate-400">加载中…</p>
          ) : tenants.length === 0 ? (
            <p className="mt-2 text-sm text-slate-400">还没有创建过组织</p>
          ) : (
            <ul className="mt-3 space-y-2 text-sm">
              {tenants.map((tenant) => (
                <li
                  key={tenant.id}
                  className="flex items-center justify-between gap-3 rounded-lg bg-slate-800 px-3 py-2"
                >
                  <div className="min-w-0">
                    <div className="truncate text-slate-100">{tenant.name}</div>
                    <div className="truncate font-mono text-xs text-slate-500">
                      {tenant.id}
                    </div>
                  </div>
                  <button
                    onClick={() => setMemberTenant(tenant.id)}
                    className={`shrink-0 rounded px-2 py-1 text-xs ${
                      memberTenant === tenant.id
                        ? "bg-cyan-600 text-white"
                        : "bg-slate-700 text-slate-200 hover:bg-slate-600"
                    }`}
                  >
                    {memberTenant === tenant.id ? "已选中" : "选为当前"}
                  </button>
                </li>
              ))}
            </ul>
          )}

          {memberTenant ? (
            <div className="mt-4 border-t border-slate-700 pt-3">
              <h3 className="text-sm font-semibold text-slate-50">添加成员</h3>
              <div className="mt-2 flex gap-2">
                <input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="要添加的用户名"
                  className="flex-1 rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 outline-none focus:border-cyan-500"
                />
                <button
                  onClick={() => void addMember()}
                  disabled={busy}
                  className="rounded-lg bg-cyan-600 px-3 py-2 font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
                >
                  添加
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
