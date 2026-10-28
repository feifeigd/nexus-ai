// Suspense 是 React 的加载边界：里面的内容还没准备好时，先显示 fallback；准备完成后，再替换成正式内容。
import { Suspense } from "react";

import ApiHealth from "./api-health";

export default function Home() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-950 px-6 text-slate-50">
      <section className="w-full max-w-2xl rounded-2xl border border-slate-800 bg-slate-900 p-10 shadow-2xl">
        <p className="text-sm font-medium tracking-widest text-cyan-400">
          NEXUSAI · USER WEB
        </p>
        <h1 className="mt-4 text-4xl font-semibold tracking-tight">NexusAI</h1>
        <p className="mt-4 text-lg text-slate-300">用户端 Web 骨架已就绪。</p>
        <Suspense
          fallback={
            <p role="status" className="mt-8 text-sm text-slate-500">
              正在检查 go-zero API…
            </p>
          }
        >
          <ApiHealth />
        </Suspense>
        <p className="mt-8 text-sm text-slate-500">
          Next.js · TypeScript · Tailwind CSS
        </p>
      </section>
    </main>
  );
}
