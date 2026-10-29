import { PageContainer } from '@ant-design/pro-components';
import { CopilotChat } from '@copilotkit/react-core/v2';
import { Alert, Card } from 'antd';
import type React from 'react';

export const AiChatContent: React.FC = () => (
  <Card>
    <Alert
      title="NexusAI Mock Agent"
      description="当前仅提供 Mock 流式回复，不会执行后台操作；业务工具、权限校验和人工审批将在后续接入。"
      type="info"
      showIcon
      style={{ marginBottom: 16 }}
    />
    <div style={{ height: 'min(680px, calc(100vh - 260px))', minHeight: 480 }}>
      <CopilotChat
        agentId="default"
        style={{ height: '100%' }}
        labels={{
          chatInputPlaceholder: '输入消息，按 Enter 发送…',
          chatDisclaimerText: '当前为 Mock 流式回复，不会执行真实后台操作。',
          welcomeMessageText: '向 NexusAI 管理后台 Mock Agent 发送一条消息。',
          modalHeaderTitle: 'NexusAI 管理助手',
        }}
      />
    </div>
  </Card>
);

const AiChat: React.FC = () => (
  <PageContainer title="AI 助手">
    <AiChatContent />
  </PageContainer>
);

export default AiChat;
