"use client";

import { useEffect, useState } from "react";
import { CopilotChat } from "@copilotkit/react-core/v2";
import { AuthForm } from "../components/auth-form";
import { UsagePanel } from "../components/usage-panel";
import { clearAuth, getAuthUser, getToken, type AuthUser } from "../lib/auth";

export default function ChatPanel() {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [ready, setReady] = useState(false);
  const [chatKey, setChatKey] = useState(0);

  // useEffect 是 React 的 Hook，用于在「组件渲染之后」执行副作用（Side Effects）——
  // 即那些跟「根据 props/state 计算 UI」无关的操作，常见有：
  //   · 请求后端 API
  //   · 读写 localStorage / sessionStorage
  //   · 订阅事件 / 启动定时器（需要在 cleanup 里清理）
  //   · 操作浏览器 API（history、document 等）
  // 第二个参数是依赖数组；传 [] 表示「只在首次渲染后执行一次」。
  // 这里一次性从 localStorage 读登录状态并置 ready，避免 SSR 阶段拿不到 window。
  useEffect(() => {
    // 同步 setState 在这里只跑一次、不会级联渲染，属合理场景。
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setUser(getAuthUser());
    setReady(true);
  }, []);

  if (!ready) {
    return <div className="p-6 text-slate-400">加载中…</div>;
  }

  if (!user || !getToken()) {
    return (
      <AuthForm
        onLoggedIn={() => {
          setUser(getAuthUser());
          setChatKey((k) => k + 1);
        }}
      />
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-slate-200 px-4 py-2 text-sm text-slate-600">
        <span>已登录：{user.username}</span>
        <button
          onClick={() => {
            clearAuth();
            setUser(null);
          }}
          className="text-slate-400 hover:text-slate-600"
        >
          退出
        </button>
      </div>
      <div className="shrink-0 border-b border-slate-200 p-3">
        <UsagePanel />
      </div>
      <div className="min-h-0 flex-1">
        <CopilotChat
          key={chatKey}
          agentId="default"
          className="h-full"
          labels={{
            chatInputPlaceholder: "输入消息，按 Enter 发送…",
            chatDisclaimerText: "回复由服务端 go-zero 配置的 Agent 模式生成。",
            welcomeMessageText: "向 NexusAI Agent 发送一条消息。",
            modalHeaderTitle: "NexusAI",
          }}
        />
      </div>
    </div>
  );
}
