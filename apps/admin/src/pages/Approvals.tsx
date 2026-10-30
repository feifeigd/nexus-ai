import {
  CheckOutlined,
  CloseOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import {
  Alert,
  App as AntdApp,
  Button,
  Card,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import {
  approveApproval,
  listApprovals,
  rejectApproval,
} from '@/services/nexus/approval';

export type Approval = NonNullable<
  API.ListApprovalsResponse['approvals']
>[number];
export type ApprovalDecision = 'approve' | 'reject';
type ApprovalStatus = Approval['status'];

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus' };
const statusLabels: Record<ApprovalStatus, { color: string; text: string }> = {
  pending: { color: 'processing', text: '待审批' },
  approved: { color: 'success', text: '已批准' },
  rejected: { color: 'default', text: '已拒绝' },
};

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : '请求失败，请稍后重试。';
}

interface ApprovalTableProps {
  approvals: Approval[];
  decidingId?: string;
  onDecide: (id: string, decision: ApprovalDecision) => void;
}

export const ApprovalTable: React.FC<ApprovalTableProps> = ({
  approvals,
  decidingId,
  onDecide,
}) => (
  <Table<Approval>
    rowKey="id"
    dataSource={approvals}
    pagination={false}
    scroll={{ x: 960 }}
    locale={{ emptyText: '暂无工具审批记录' }}
    columns={[
      {
        title: '工具与工单',
        key: 'ticket',
        render: (_, record) => (
          <div>
            <Typography.Text strong>{record.arguments.title}</Typography.Text>
            <Typography.Paragraph
              type="secondary"
              ellipsis={{ rows: 2, tooltip: record.arguments.description }}
              style={{ marginBottom: 4, maxWidth: 420 }}
            >
              {record.arguments.description || '无描述'}
            </Typography.Paragraph>
            <Typography.Text type="secondary" code>
              {record.toolName}
            </Typography.Text>
          </div>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        render: (status: ApprovalStatus) => (
          <Tag color={statusLabels[status]?.color}>
            {statusLabels[status]?.text ?? status}
          </Tag>
        ),
      },
      {
        title: '提议时间',
        dataIndex: 'createdAt',
        width: 190,
        render: (value: string) =>
          new Date(value).toLocaleString('zh-CN', { hour12: false }),
      },
      {
        title: '工单 ID',
        dataIndex: 'ticketId',
        width: 220,
        render: (value?: string) =>
          value ? <Typography.Text code>{value}</Typography.Text> : '—',
      },
      {
        title: '操作',
        key: 'actions',
        width: 190,
        fixed: 'right',
        render: (_, record) =>
          record.status === 'pending' ? (
            <Space>
              <Popconfirm
                title="批准创建这张工单？"
                description="批准后 go-zero 才会执行创建操作。"
                okText="确认批准"
                cancelText="取消"
                onConfirm={() => onDecide(record.id, 'approve')}
              >
                <Button
                  type="primary"
                  size="small"
                  icon={<CheckOutlined />}
                  loading={decidingId === record.id}
                  disabled={Boolean(decidingId)}
                >
                  批准
                </Button>
              </Popconfirm>
              <Popconfirm
                title="拒绝这次工具提议？"
                description="拒绝后不会创建工单，且不能再次批准。"
                okText="确认拒绝"
                okButtonProps={{ danger: true }}
                cancelText="取消"
                onConfirm={() => onDecide(record.id, 'reject')}
              >
                <Button
                  danger
                  size="small"
                  icon={<CloseOutlined />}
                  loading={decidingId === record.id}
                  disabled={Boolean(decidingId)}
                >
                  拒绝
                </Button>
              </Popconfirm>
            </Space>
          ) : (
            <Typography.Text type="secondary">已完成</Typography.Text>
          ),
      },
    ]}
  />
);

export const ApprovalsContent: React.FC = () => {
  const { message } = AntdApp.useApp();
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(true);
  const [decidingId, setDecidingId] = useState<string>();
  const [error, setError] = useState<string>();

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const response = await listApprovals(NEXUS_API_OPTIONS);
      setApprovals(response.approvals ?? []);
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const decide = useCallback(
    async (id: string, decision: ApprovalDecision) => {
      setDecidingId(id);
      setError(undefined);
      try {
        if (decision === 'approve') {
          await approveApproval({ id }, NEXUS_API_OPTIONS);
        } else {
          await rejectApproval({ id }, NEXUS_API_OPTIONS);
        }
        await load();
        await message.success(
          decision === 'approve' ? '已批准并创建工单。' : '已拒绝该工具提议。',
        );
      } catch (requestError) {
        setError(errorMessage(requestError));
      } finally {
        setDecidingId(undefined);
      }
    },
    [load, message],
  );

  return (
    <>
      <Alert
        title="模型只能提议工具调用，人工批准后才执行"
        description="当前提供 create_ticket 参考流程。默认使用内存存储，重启后记录清空；配置 NEXUS_POSTGRES_DSN 后持久化审批和已创建工单。真实认证、租户隔离和审计尚未接入。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      {error ? (
        <Alert
          title="审批请求失败"
          description={error}
          type="error"
          showIcon
          closable
          onClose={() => setError(undefined)}
          style={{ marginBottom: 16 }}
        />
      ) : null}
      <Card
        title={`审批记录（${approvals.length}）`}
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
        <ApprovalTable
          approvals={approvals}
          decidingId={decidingId}
          onDecide={(id, decision) => void decide(id, decision)}
        />
      </Card>
    </>
  );
};

const Approvals: React.FC = () => (
  <PageContainer title="工具审批">
    <AntdApp>
      <ApprovalsContent />
    </AntdApp>
  </PageContainer>
);

export default Approvals;
