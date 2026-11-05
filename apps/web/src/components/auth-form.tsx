"use client";

import { useState } from "react";
import { setAuth } from "../lib/auth";

type Mode = "login" | "register";

export function AuthForm({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [mode, setMode] = useState<Mode>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setError("");
    if (!username.trim() || !password) {
      setError("请输入用户名和密码");
      return;
    }
    setBusy(true);
    try {
      const res = await fetch(`/api/auth/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: username.trim(), password }),
      });
      const text = await res.text();
      if (!res.ok) {
        setError(text || "操作失败，请稍后重试");
        return;
      }
      const data = JSON.parse(text);
      setAuth(data.token, { username: data.username, role: data.role });
      onLoggedIn();
    } catch (e) {
      setError(e instanceof Error ? e.message : "操作失败，请稍后重试");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="rounded-xl border border-slate-700 bg-slate-900 p-6">
      <h2 className="text-lg font-semibold text-slate-50">
        {mode === "login" ? "登录" : "注册"}
      </h2>
      <p className="mt-2 text-sm text-slate-400">
        登录后才能使用 AI 聊天（对话 token 会记录到你的用量）。
      </p>
      <div className="mt-4 flex flex-col gap-3">
        <input
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="用户名"
          className="rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 outline-none focus:border-cyan-500"
        />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="密码"
          onKeyDown={(e) => {
            if (e.key === "Enter") void submit();
          }}
          className="rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 outline-none focus:border-cyan-500"
        />
        {error ? <p className="text-sm text-red-400">{error}</p> : null}
        <button
          onClick={() => void submit()}
          disabled={busy}
          className="rounded-lg bg-cyan-600 px-3 py-2 font-medium text-white hover:bg-cyan-500 disabled:opacity-50"
        >
          {busy ? "处理中…" : mode === "login" ? "登录" : "注册"}
        </button>
        <button
          onClick={() => setMode(mode === "login" ? "register" : "login")}
          className="text-sm text-slate-400 hover:text-cyan-400"
        >
          {mode === "login" ? "没有账号？去注册" : "已有账号？去登录"}
        </button>
        <div className="my-1 flex items-center gap-3 text-xs text-slate-500">
          <span className="h-px flex-1 bg-slate-700" />
          或
          <span className="h-px flex-1 bg-slate-700" />
        </div>
        <a
          href="/api/auth/github/login"
          className="rounded-lg border border-slate-600 bg-slate-800 px-3 py-2 text-center text-slate-100 hover:bg-slate-700"
        >
          使用 GitHub 登录
        </a>
      </div>
    </div>
  );
}
