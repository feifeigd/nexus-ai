import { PageContainer } from '@ant-design/pro-components';
import { Card, Table, Typography } from 'antd';
import type React from 'react';
import { useEffect, useState } from 'react';
import { listAuditLogs } from '@/services/nexus/audit';

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus', skipErrorHandler: true };

export type AuditLogItem = NonNullable<
  API.ListAuditLogsResponse['items']
>[number];

const ACTION_LABELS: Record<string, string> = {
  approve_approval: '批准审批',
  reject_approval: '拒绝审批',
  update_ticket_status: '更新工单状态',
  create_tenant: '创建组织',
  switch_tenant: '切换租户',
  add_tenant_member: '添加成员',
  ingest_document: '导入文档',
  delete_document: '删除文档',
  create_machine: '创建机器',
  update_machine: '更新机器',
  delete_machine: '删除机器',
  run_ssh_command: '执行 SSH 命令',
};

const Audit: React.FC = () => {
  const [items, setItems] = useState<AuditLogItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const response = await listAuditLogs({ ...NEXUS_API_OPTIONS });
        if (!cancelled) setItems(response.items ?? []);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <PageContainer title="审计日志">
      <Card>
        <Table<AuditLogItem>
          rowKey="id"
          dataSource={items}
          loading={loading}
          pagination={{ pageSize: 20, hideOnSinglePage: true }}
          locale={{ emptyText: '暂无审计记录' }}
          columns={[
            {
              title: '时间',
              dataIndex: 'createdAt',
              width: 180,
              render: (value: string) =>
                new Date(value).toLocaleString('zh-CN', { hour12: false }),
            },
            { title: '操作人', dataIndex: 'actorUsername', width: 140 },
            {
              title: '动作',
              dataIndex: 'action',
              width: 140,
              render: (value: string) => ACTION_LABELS[value] ?? value,
            },
            { title: '资源类型', dataIndex: 'resourceType', width: 120 },
            {
              title: '资源 ID',
              dataIndex: 'resourceId',
              width: 260,
              render: (value: string) => (
                <Typography.Text code ellipsis>
                  {value}
                </Typography.Text>
              ),
            },
            { title: '详情', dataIndex: 'detail', ellipsis: true },
          ]}
        />
      </Card>
    </PageContainer>
  );
};

export default Audit;
