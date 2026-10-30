import { PageContainer } from '@ant-design/pro-components';
import { CopilotChat } from '@copilotkit/react-core/v2';
import { Alert, Card } from 'antd';
import type React from 'react';

export const AiChatContent: React.FC = () => (
  <Card>
    <Alert
      title="NexusAI 管理助手"
      description="支持 Mock 或 OpenAI 兼容模型，回复来源由 go-zero 配置；创建工单等工具提议必须先经过人工审批，不会未经批准执行后台操作。"
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
          chatDisclaimerText: '回复来源由 go-zero 配置；未经人工批准，不会执行真实后台操作。',
          welcomeMessageText: '向 NexusAI 管理后台 AI 助手发送一条消息。',
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
