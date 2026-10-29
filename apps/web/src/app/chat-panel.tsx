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
        chatDisclaimerText: "当前为 Mock 流式回复，不是真实模型输出。",
        welcomeMessageText: "向 NexusAI Mock Agent 发送一条消息。",
        modalHeaderTitle: "NexusAI",
      }}
    />
  );
}
