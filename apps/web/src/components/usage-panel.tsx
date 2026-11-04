"use client";

import { useEffect, useState } from "react";
import { getToken } from "../lib/auth";

interface UsageItem {
  threadId: string;
  runId: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  judgeTokens: number;
  createdAt: string;
}

export function UsagePanel() {
  const [items, setItems] = useState<UsageItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError("");
      try {
        const token = getToken();
        const res = await fetch("/api/usage", {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!res.ok) {
          setError("加载用量失败");
          return;
        }
        const data = (await res.json()) as { items?: UsageItem[] };
        if (!cancelled) setItems(data.items ?? []);
      } catch {
        if (!cancelled) setError("加载用量失败");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const totalTokens = items.reduce((sum, item) => sum + item.totalTokens, 0);

  return (
    <div className="rounded-xl border border-slate-700 bg-slate-900 p-4 text-slate-200">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-50">我的用量</h3>
        <span className="text-sm text-cyan-400">{totalTokens} tokens</span>
      </div>
      {loading ? (
        <p className="mt-2 text-sm text-slate-400">加载中…</p>
      ) : error ? (
        <p className="mt-2 text-sm text-red-400">{error}</p>
      ) : items.length === 0 ? (
        <p className="mt-2 text-sm text-slate-400">还没有对话记录</p>
      ) : (
        <ul className="mt-3 space-y-2 text-sm">
          {items.slice(0, 10).map((item) => (
            <li
              key={`${item.threadId}-${item.runId}`}
              className="flex justify-between gap-3"
            >
              <span className="truncate text-slate-400">
                {new Date(item.createdAt).toLocaleString("zh-CN", {
                  hour12: false,
                })}
              </span>
              <span className="shrink-0">{item.totalTokens} tokens</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
