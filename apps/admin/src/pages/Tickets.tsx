import { ReloadOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { Alert, Button, Card, Table, Tag, Typography } from 'antd';
import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import { listTickets } from '@/services/nexus/ticket';

export type Ticket = NonNullable<API.ListTicketsResponse['tickets']>[number];

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus' };

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : '请求失败，请稍后重试。';
}

interface TicketTableProps {
  tickets: Ticket[];
}

export const TicketTable: React.FC<TicketTableProps> = ({ tickets }) => (
  <Table<Ticket>
    rowKey="id"
    dataSource={tickets}
    pagination={false}
    scroll={{ x: 960 }}
    locale={{ emptyText: '暂无已创建工单' }}
    columns={[
      {
        title: '工单',
        key: 'ticket',
        render: (_, record) => (
          <div>
            <Typography.Text strong>{record.title}</Typography.Text>
            <Typography.Paragraph
              type="secondary"
              ellipsis={{ rows: 2, tooltip: record.description }}
              style={{ marginBottom: 0, maxWidth: 480 }}
            >
              {record.description || '无描述'}
            </Typography.Paragraph>
          </div>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        render: (status: string) =>
          status === 'open' ? <Tag color="processing">待处理</Tag> : <Tag>{status}</Tag>,
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        width: 190,
        render: (value: string) =>
          new Date(value).toLocaleString('zh-CN', { hour12: false }),
      },
      {
        title: '工单 ID',
        dataIndex: 'id',
        width: 250,
        render: (value: string) => <Typography.Text code>{value}</Typography.Text>,
      },
    ]}
  />
);

export const TicketsContent: React.FC = () => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const response = await listTickets(NEXUS_API_OPTIONS);
      setTickets(response.tickets ?? []);
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <>
      <Alert
        title="已批准工具提议创建的工单"
        description="当前仅提供只读列表；真实认证、租户隔离、审计以及工单处理流程尚未接入。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      {error ? (
        <Alert
          title="工单请求失败"
          description={error}
          type="error"
          showIcon
          closable
          onClose={() => setError(undefined)}
          style={{ marginBottom: 16 }}
        />
      ) : null}
      <Card
        title={`工单记录（${tickets.length}）`}
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
        <TicketTable tickets={tickets} />
      </Card>
    </>
  );
};

const Tickets: React.FC = () => (
  <PageContainer title="工单管理">
    <TicketsContent />
  </PageContainer>
);

export default Tickets;
