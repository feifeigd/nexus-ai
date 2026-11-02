import { PageContainer } from '@ant-design/pro-components';
import {
  CopilotChat,
  CopilotChatAssistantMessage,
  useDefaultRenderTool,
  useRenderTool,
} from '@copilotkit/react-core/v2';
import { Alert, Card } from 'antd';
import type React from 'react';
import { z } from 'zod';
import { KnowledgeSearchToolResult } from '@/components/KnowledgeSearchToolResult';
import './AiChat.less';

/**
 * 纯工具调用消息没有正文，隐藏无意义的复制按钮；
 * 有正文时不覆盖 copyButton，保留 CopilotKit 原生复制行为。
 */
const AssistantMessageWithoutEmptyCopy = Object.assign(
  (props: React.ComponentProps<typeof CopilotChatAssistantMessage>) => {
    if (!props.message.content?.trim()) {
      return <CopilotChatAssistantMessage {...props} copyButton={() => null} />;
    }
    return <CopilotChatAssistantMessage {...props} />;
  },
  CopilotChatAssistantMessage,
);

export const AiChatContent: React.FC = () => {
  useDefaultRenderTool();
  useRenderTool({
    name: 'search_knowledge',
    agentId: 'default',
    parameters: z.object({
      query: z.string().min(1).max(1000),
      limit: z.number().int().min(1).max(10).optional(),
    }),
    render: ({ status, parameters, result }) => (
      <KnowledgeSearchToolResult status={status} query={parameters.query} result={result} />
    ),
  });

  return (
    <Card>
      <Alert
        title="NexusAI 管理助手"
        description="支持 Mock 或 OpenAI 兼容模型，回复来源由 go-zero 配置；创建工单等工具提议必须先经过人工审批，不会未经批准执行后台操作。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <div
        className="nexus-ai-chat"
        style={{ height: 'min(680px, calc(100vh - 260px))', minHeight: 480 }}
      >
        <CopilotChat
          agentId="default"
          style={{ height: '100%' }}
          messageView={{ assistantMessage: AssistantMessageWithoutEmptyCopy }}
          labels={{
            chatInputPlaceholder: '输入消息，按 Enter 发送…',
            chatDisclaimerText: '回复来源由 go-zero 配置；未经人工批准，不会执行真实后台操作。',
            welcomeMessageText: '向 NexusAI 管理后台 AI 助手发送一条消息。',
            modalHeaderTitle: 'NexusAI 管理助手',
          }}
        />
      </div>
    </Card>
  );
};

const AiChat: React.FC = () => (
  <PageContainer title="AI 助手">
    <AiChatContent />
  </PageContainer>
);

export default AiChat;
