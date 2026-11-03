"use client";

import { CopilotChat } from "@copilotkit/react-core/v2";

export default function ChatPanel() {
  return (
    <CopilotChat
      // 连接 default Agent
      agentId="default"
      className="h-full"
      labels={{
        chatInputPlaceholder: "输入消息，按 Enter 发送…",
        chatDisclaimerText: "回复由服务端 go-zero 配置的 Agent 模式生成。",
        welcomeMessageText: "向 NexusAI Agent 发送一条消息。",
        modalHeaderTitle: "NexusAI",
      }}
    />
  );
}
