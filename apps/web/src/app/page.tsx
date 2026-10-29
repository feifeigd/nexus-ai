// Suspense 是 React 的加载边界：里面的内容还没准备好时，先显示 fallback；准备完成后，再替换成正式内容。
import { Suspense } from "react";

import ApiHealth from "./api-health";
import ChatPanel from "./chat-panel";

export default function Home() {
  return (
    <main className="min-h-screen bg-slate-950 px-6 py-10 text-slate-50">
      <div className="mx-auto max-w-6xl">
        <header>
          <p className="text-sm font-medium tracking-widest text-cyan-400">
            NEXUSAI · USER WEB
          </p>
          <h1 className="mt-3 text-4xl font-semibold tracking-tight">NexusAI</h1>
          <p className="mt-3 text-lg text-slate-300">
            企业级 AI 应用开发框架
          </p>
        </header>

        <div className="mt-8 grid gap-6 lg:grid-cols-[0.8fr_1.2fr]">
          <section className="rounded-2xl border border-slate-800 bg-slate-900 p-8 shadow-2xl">
            <p className="text-sm font-medium text-cyan-400">AG-UI STREAM</p>
            <h2 className="mt-3 text-2xl font-semibold">流式聊天链路已接通</h2>
            <p className="mt-4 leading-7 text-slate-300">
              消息经过 CopilotKit Runtime 发送到 go-zero，再通过 AG-UI SSE
              事件逐段返回。
            </p>
            <Suspense
              fallback={
                <p role="status" className="mt-8 text-sm text-slate-500">
                  正在检查 go-zero API…
                </p>
              }
            >
              <ApiHealth />
            </Suspense>
            <p className="mt-8 text-sm leading-6 text-slate-500">
              当前使用 Mock Agent，仅验证协议与界面，不调用真实模型。
            </p>
          </section>

          <section
            aria-labelledby="agent-chat-title"
            className="overflow-hidden rounded-2xl border border-slate-700 bg-white shadow-2xl"
          >
            <div className="border-b border-slate-200 px-6 py-4 text-slate-950">
              <h2 id="agent-chat-title" className="font-semibold">
                NexusAI Agent
              </h2>
              <p className="mt-1 text-xs text-slate-500">MOCK · STREAMING</p>
            </div>
            <div className="h-[560px]">
              <ChatPanel />
            </div>
          </section>
        </div>
      </div>
    </main>
  );
}
