import { PageContainer } from '@ant-design/pro-components';
import { Card, Typography } from 'antd';
import React from 'react';

export const WelcomeContent: React.FC = () => (
  <Card>
    <Typography.Title level={2}>NexusAI 管理后台</Typography.Title>
    <Typography.Paragraph>
      基于 Ant Design Pro 的独立管理控制台。
    </Typography.Paragraph>
    <Typography.Paragraph type="secondary">
      当前提供基础布局、登录和权限示例，后续将接入企业知识库与智能工单管理功能。
    </Typography.Paragraph>
  </Card>
);

const Welcome: React.FC = () => (
  <PageContainer title="控制台">
    <WelcomeContent />
  </PageContainer>
);

export default Welcome;
