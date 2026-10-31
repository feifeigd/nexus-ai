import { ReloadOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import {
  Alert,
  App as AntdApp,
  Button,
  Card,
  Table,
  Tag,
  Typography,
} from 'antd';
import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import { listTickets, updateTicketStatus } from '@/services/nexus/ticket';

export type Ticket = NonNullable<API.ListTicketsResponse['tickets']>[number];
export type TicketStatus = 'open' | 'in_progress' | 'resolved';

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus' };
const statusLabels: Record<string, { color: string; text: string }> = {
  open: { color: 'processing', text: '待处理' },
  in_progress: { color: 'warning', text: '处理中' },
  resolved: { color: 'success', text: '已完成' },
};

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : '请求失败，请稍后重试。';
}

interface TicketTableProps {
  tickets: Ticket[];
  updatingId?: string;
  onUpdateStatus: (id: string, status: TicketStatus) => void;
}

export const TicketTable: React.FC<TicketTableProps> = ({
  tickets,
  updatingId,
  onUpdateStatus,
}) => (
  <Table<Ticket>
    rowKey="id"
    dataSource={tickets}
    pagination={false}
    scroll={{ x: 1080 }}
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
        render: (status: string) => (
          <Tag color={statusLabels[status]?.color}>
            {statusLabels[status]?.text ?? status}
          </Tag>
        ),
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
      {
        title: '操作',
        key: 'actions',
        width: 130,
        fixed: 'right',
        render: (_, record) => {
          const nextStatus =
            record.status === 'open'
              ? 'in_progress'
              : record.status === 'in_progress'
                ? 'resolved'
                : undefined;
          if (!nextStatus) {
            return <Typography.Text type="secondary">已结束</Typography.Text>;
          }
          return (
            <Button
              type="primary"
              size="small"
              loading={updatingId === record.id}
              disabled={Boolean(updatingId)}
              onClick={() => onUpdateStatus(record.id, nextStatus)}
            >
              {nextStatus === 'in_progress' ? '开始处理' : '标记完成'}
            </Button>
          );
        },
      },
    ]}
  />
);

export const TicketsContent: React.FC = () => {
  const { message } = AntdApp.useApp();
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [updatingId, setUpdatingId] = useState<string>();
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

  const changeStatus = useCallback(
    async (id: string, status: TicketStatus) => {
      setUpdatingId(id);
      setError(undefined);
      try {
        await updateTicketStatus({ id }, { status }, NEXUS_API_OPTIONS);
        await load();
        await message.success(
          status === 'in_progress' ? '工单已开始处理。' : '工单已完成。',
        );
      } catch (requestError) {
        setError(errorMessage(requestError));
      } finally {
        setUpdatingId(undefined);
      }
    },
    [load, message],
  );

  return (
    <>
      <Alert
        title="已批准工具提议创建的工单"
        description="支持待处理、处理中、已完成的顺序流转；真实认证、租户隔离和审计尚未接入。"
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
        <TicketTable
          tickets={tickets}
          updatingId={updatingId}
          onUpdateStatus={(id, status) => void changeStatus(id, status)}
        />
      </Card>
    </>
  );
};

const Tickets: React.FC = () => (
  <PageContainer title="工单管理">
    <AntdApp>
      <TicketsContent />
    </AntdApp>
  </PageContainer>
);

export default Tickets;
