import { ReloadOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import {
  Alert,
  App as AntdApp,
  Button,
  Card,
  Col,
  Row,
  Segmented,
  Statistic,
  Table,
  Typography,
} from 'antd';
import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import { listTokenUsage } from '@/services/nexus/usage';

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus', skipErrorHandler: true };

export type TokenUsageItem = NonNullable<
  API.ListTokenUsageResponse['items']
>[number];

export const UsageContent: React.FC = () => {
  const [items, setItems] = useState<TokenUsageItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [scope, setScope] = useState<'self' | 'all'>('self');

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const response = await listTokenUsage(
        scope === 'all' ? { scope: 'all' } : {},
        { ...NEXUS_API_OPTIONS },
      );
      setItems(response.items ?? []);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : '用量明细加载失败，请稍后重试。',
      );
    } finally {
      setLoading(false);
    }
  }, [scope]);

  useEffect(() => {
    void load();
  }, [load]);

  const totalTokens = items.reduce((sum, item) => sum + item.totalTokens, 0);
  const totalPrompt = items.reduce((sum, item) => sum + item.promptTokens, 0);
  const totalCompletion = items.reduce(
    (sum, item) => sum + item.completionTokens,
    0,
  );

  return (
    <>
      <Alert
        title="对话 token 消耗明细"
        description="每次 AG-UI 对话完成后按 thread/run 记录；当前只统计对话模型 token，语义核验与嵌入暂未计入。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Segmented
        value={scope}
        onChange={(value) => setScope(value as 'self' | 'all')}
        options={[
          { label: '只看自己', value: 'self' },
          { label: '看全部', value: 'all' },
        ]}
        style={{ marginBottom: 16 }}
      />
      {error ? (
        <Alert
          title="用量明细加载失败"
          description={error}
          type="error"
          showIcon
          closable
          onClose={() => setError(undefined)}
          style={{ marginBottom: 16 }}
        />
      ) : null}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} md={6}>
          <Card>
            <Statistic title="对话次数" value={items.length} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic title="输入 token" value={totalPrompt} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic title="输出 token" value={totalCompletion} />
          </Card>
        </Col>
        <Col xs={12} md={6}>
          <Card>
            <Statistic title="合计 token" value={totalTokens} />
          </Card>
        </Col>
      </Row>
      <Card
        title={`明细（${items.length}）`}
        extra={
          <Button
            icon={<ReloadOutlined />}
            loading={loading}
            onClick={() => void load()}
          >
            刷新
          </Button>
        }
      >
        <Table<TokenUsageItem>
          rowKey={(record) => `${record.threadId}-${record.runId}`}
          dataSource={items}
          loading={loading}
          pagination={{ pageSize: 20, hideOnSinglePage: true }}
          scroll={{ x: 960 }}
          locale={{ emptyText: '暂无消耗明细' }}
          columns={[
            {
              title: '时间',
              dataIndex: 'createdAt',
              width: 180,
              render: (value: string) =>
                new Date(value).toLocaleString('zh-CN', { hour12: false }),
            },
            {
              title: 'Thread ID',
              dataIndex: 'threadId',
              width: 280,
              render: (value: string) => (
                <Typography.Text code ellipsis>
                  {value}
                </Typography.Text>
              ),
            },
            { title: '输入 token', dataIndex: 'promptTokens', width: 110 },
            { title: '输出 token', dataIndex: 'completionTokens', width: 110 },
            {
              title: '合计 token',
              dataIndex: 'totalTokens',
              width: 110,
              render: (value: number) => (
                <Typography.Text strong>{value}</Typography.Text>
              ),
            },
            { title: '核验 token', dataIndex: 'judgeTokens', width: 110 },
          ]}
        />
      </Card>
    </>
  );
};

const Usage: React.FC = () => (
  <PageContainer title="用量统计">
    <AntdApp>
      <UsageContent />
    </AntdApp>
  </PageContainer>
);

export default Usage;
