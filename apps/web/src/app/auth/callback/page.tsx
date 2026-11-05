"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { setAuth } from "../../../lib/auth";

export default function GitHubCallbackPage() {
  const router = useRouter();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("token");
    const username = params.get("username");
    const role = params.get("role");
    if (token && username && role) {
      setAuth(token, { username, role });
    }
    // 无论成功还是失败都回到首页：成功则已登录，失败则停留在登录表单。
    router.replace("/");
  }, [router]);

  return (
    <div className="flex min-h-screen items-center justify-center text-slate-400">
      正在登录…
    </div>
  );
}
