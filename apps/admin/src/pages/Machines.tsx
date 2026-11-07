import { PlayCircleOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import {
  Alert,
  App as AntdApp,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';
import type React from 'react';
import { useCallback, useEffect, useState } from 'react';
import {
  createMachine,
  deleteMachine,
  execMachineCommand,
  listMachines,
  updateMachine,
} from '@/services/nexus/machine';

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus', skipErrorHandler: true };

export type Machine = NonNullable<API.ListMachinesResponse['machines']>[number];

interface MachineFormValues {
  name?: string;
  privateAddress: string;
  publicAddress?: string;
  region?: string;
  rootPublicKey?: string;
  sshPrivateKey?: string;
  joinedK8s?: boolean;
}

function errorMessage(error: unknown) {
  // go-zero 的 logic 错误以纯文本返回（HTTP 400），透出后端真实信息（如
  // 「该机器未配置 SSH 私钥」「命令不在白名单内」），而不是 axios 的通用报错。
  const responseData = (
    error as { response?: { data?: unknown } } | undefined
  )?.response?.data;
  if (typeof responseData === 'string' && responseData.trim()) {
    return responseData;
  }
  return error instanceof Error ? error.message : '请求失败，请稍后重试。';
}

export const MachineTable: React.FC<{
  machines: Machine[];
  loading?: boolean;
  deletingId?: string;
  onEdit: (machine: Machine) => void;
  onDelete: (id: string) => void;
  onExec: (machine: Machine) => void;
}> = ({ machines, loading, deletingId, onEdit, onDelete, onExec }) => (
  <Table<Machine>
    rowKey="id"
    dataSource={machines}
    loading={loading}
    pagination={false}
    scroll={{ x: 1080 }}
    locale={{ emptyText: '暂无机器，点击右上角「新建机器」登记一台。' }}
    columns={[
      {
        title: '名称',
        dataIndex: 'name',
        width: 120,
        render: (value?: string) =>
          value ? <Typography.Text strong>{value}</Typography.Text> : '—',
      },
      {
        title: '私有地址',
        dataIndex: 'privateAddress',
        width: 150,
        render: (value: string) => (
          <Typography.Text code copyable>
            {value}
          </Typography.Text>
        ),
      },
      {
        title: '公网地址',
        dataIndex: 'publicAddress',
        width: 150,
        render: (value?: string) =>
          value ? <Typography.Text code>{value}</Typography.Text> : '—',
      },
      {
        title: '区域',
        dataIndex: 'region',
        width: 110,
        render: (value?: string) => value || '—',
      },
      {
        title: 'root 公钥',
        dataIndex: 'rootPublicKey',
        width: 220,
        render: (value?: string) =>
          value ? (
            <Typography.Text
              code
              ellipsis={{ tooltip: value }}
              style={{ maxWidth: 200 }}
            >
              {value}
            </Typography.Text>
          ) : (
            '—'
          ),
      },
      {
        title: 'K8s',
        dataIndex: 'joinedK8s',
        width: 100,
        render: (joined: boolean) =>
          joined ? <Tag color="success">已加入</Tag> : <Tag>未加入</Tag>,
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        width: 180,
        render: (value: string) =>
          new Date(value).toLocaleString('zh-CN', { hour12: false }),
      },
      {
        title: '操作',
        key: 'actions',
        width: 200,
        fixed: 'right',
        render: (_, record) => (
          <Space size={0}>
            <Button
              type="link"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => onExec(record)}
            >
              执行命令
            </Button>
            <Button type="link" size="small" onClick={() => onEdit(record)}>
              编辑
            </Button>
            <Popconfirm
              title="删除这台机器？"
              description="删除后不可恢复。"
              okText="确认删除"
              cancelText="取消"
              okButtonProps={{ danger: true, loading: deletingId === record.id }}
              onConfirm={() => onDelete(record.id)}
            >
              <Button
                type="link"
                size="small"
                danger
                loading={deletingId === record.id}
              >
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ]}
  />
);

export const MachineFormModal: React.FC<{
  open: boolean;
  editing?: Machine;
  submitting?: boolean;
  onSubmit: (values: MachineFormValues) => void;
  onCancel: () => void;
}> = ({ open, editing, submitting, onSubmit, onCancel }) => {
  const [form] = Form.useForm<MachineFormValues>();

  // 用 key 强制表单在「新建 / 编辑不同机器」之间重新挂载，
  // 让 initialValues 生效，避免复用上一次的残留字段值。
  const initialValues: Partial<MachineFormValues> = editing
    ? {
        name: editing.name,
        privateAddress: editing.privateAddress,
        publicAddress: editing.publicAddress,
        region: editing.region,
        rootPublicKey: editing.rootPublicKey,
        joinedK8s: editing.joinedK8s,
      }
    : { joinedK8s: false };

  return (
    <Modal
      title={editing ? '编辑机器' : '新建机器'}
      open={open}
      onCancel={onCancel}
      onOk={() => form.submit()}
      okText={editing ? '保存' : '创建'}
      confirmLoading={submitting}
    >
      <Form<MachineFormValues>
        key={editing?.id ?? 'create'}
        form={form}
        layout="vertical"
        initialValues={initialValues}
        onFinish={onSubmit}
      >
        <Form.Item name="name" label="名称">
          <Input placeholder="例如：web-1（可选）" maxLength={100} />
        </Form.Item>
        <Form.Item
          name="privateAddress"
          label="私有地址"
          rules={[{ required: true, whitespace: true, message: '请输入私有地址' }]}
        >
          <Input placeholder="例如：10.0.0.1" maxLength={64} />
        </Form.Item>
        <Form.Item name="publicAddress" label="公网地址">
          <Input placeholder="例如：1.2.3.4（可选）" maxLength={64} />
        </Form.Item>
        <Form.Item name="region" label="所在区域">
          <Input placeholder="例如：cn-east（可选）" maxLength={64} />
        </Form.Item>
        <Form.Item
          name="rootPublicKey"
          label="root 公钥"
          extra="用于安装基础环境，例如 ssh-rsa 开头的完整公钥。"
        >
          <Input.TextArea
            placeholder="ssh-rsa AAAA…（可选）"
            autoSize={{ minRows: 3, maxRows: 6 }}
          />
        </Form.Item>
        <Form.Item
          name="sshPrivateKey"
          label="SSH 登录私钥"
          extra="用于远程执行命令（root 登录）。私钥只写不读：编辑时留空保持不变，输入新值则替换。"
        >
          <Input.TextArea
            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----（可选）"
            autoSize={{ minRows: 3, maxRows: 8 }}
          />
        </Form.Item>
        <Form.Item
          name="joinedK8s"
          label="是否已加入 K8s 集群"
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
};

const ALLOWED_COMMANDS_HINT =
  'uptime / df / free / hostname / uname / systemctl status / docker ps / kubectl get / tail';

const outputStyle: React.CSSProperties = {
  background: '#f6f8fa',
  padding: 12,
  borderRadius: 6,
  overflowX: 'auto',
  margin: '8px 0 0',
  whiteSpace: 'pre-wrap',
  overflowWrap: 'anywhere',
};

export const MachineExecDrawer: React.FC<{
  machine?: Machine;
  open: boolean;
  onClose: () => void;
}> = ({ machine, open, onClose }) => {
  const [command, setCommand] = useState('');
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<API.ExecMachineCommandResponse>();
  const [execError, setExecError] = useState<string>();

  // 每次打开清空上一次的命令与输出。
  useEffect(() => {
    if (open) {
      setCommand('');
      setResult(undefined);
      setExecError(undefined);
    }
  }, [open, machine?.id]);

  const run = useCallback(async () => {
    if (!machine) return;
    const trimmed = command.trim();
    if (!trimmed) {
      setExecError('请输入要执行的命令。');
      return;
    }
    setRunning(true);
    setExecError(undefined);
    setResult(undefined);
    try {
      const response = await execMachineCommand(
        { id: machine.id },
        { command: trimmed },
        NEXUS_API_OPTIONS,
      );
      setResult(response);
    } catch (requestError) {
      setExecError(errorMessage(requestError));
    } finally {
      setRunning(false);
    }
  }, [machine, command]);

  return (
    <Drawer
      title={machine ? `执行命令 · ${machine.name || machine.privateAddress}` : '执行命令'}
      open={open}
      onClose={onClose}
      size="large"
    >
      <Space orientation="vertical" size={16} style={{ width: '100%' }}>
        <Alert
          title="命令白名单（只读诊断，逐步开放）"
          description={`仅允许：${ALLOWED_COMMANDS_HINT}。白名单由 go-zero 配置，此处提示仅作参考。`}
          type="info"
          showIcon
        />
        <Input.Search
          value={command}
          onChange={(event) => setCommand(event.target.value)}
          onSearch={() => void run()}
          enterButton="运行"
          loading={running}
          placeholder="例如：df -h"
        />
        {execError ? (
          <Alert
            title="执行失败"
            description={execError}
            type="error"
            showIcon
            closable
            onClose={() => setExecError(undefined)}
          />
        ) : null}
        {result ? (
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            <Space>
              <Typography.Text type="secondary">退出码</Typography.Text>
              <Tag color={result.exitCode === 0 ? 'success' : 'error'}>
                {result.exitCode}
              </Tag>
            </Space>
            {result.stdout ? (
              <div>
                <Typography.Text type="secondary">stdout</Typography.Text>
                <pre style={outputStyle}>{result.stdout}</pre>
              </div>
            ) : null}
            {result.stderr ? (
              <div>
                <Typography.Text type="secondary">stderr</Typography.Text>
                <pre style={{ ...outputStyle, background: '#fff1f0' }}>
                  {result.stderr}
                </pre>
              </div>
            ) : null}
          </Space>
        ) : null}
      </Space>
    </Drawer>
  );
};

export const MachinesContent: React.FC = () => {
  const { message } = AntdApp.useApp();
  const [machines, setMachines] = useState<Machine[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string>();
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Machine>();
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<string>();
  const [execMachine, setExecMachine] = useState<Machine>();

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError(undefined);
    try {
      const response = await listMachines(NEXUS_API_OPTIONS);
      setMachines(response.machines ?? []);
    } catch (requestError) {
      setLoadError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const openCreate = useCallback(() => {
    setEditing(undefined);
    setModalOpen(true);
  }, []);

  const openEdit = useCallback((machine: Machine) => {
    setEditing(machine);
    setModalOpen(true);
  }, []);

  const submit = useCallback(
    async (values: MachineFormValues) => {
      setSubmitting(true);
      try {
        if (editing) {
          await updateMachine({ id: editing.id }, values, NEXUS_API_OPTIONS);
          await message.success('机器已更新。');
        } else {
          await createMachine(values, NEXUS_API_OPTIONS);
          await message.success('机器已创建。');
        }
        setModalOpen(false);
        void load();
      } catch (requestError) {
        await message.error(errorMessage(requestError));
      } finally {
        setSubmitting(false);
      }
    },
    [editing, load, message],
  );

  const remove = useCallback(
    async (id: string) => {
      setDeletingId(id);
      try {
        await deleteMachine({ id }, NEXUS_API_OPTIONS);
        await message.success('机器已删除。');
        void load();
      } catch (requestError) {
        await message.error(errorMessage(requestError));
      } finally {
        setDeletingId(undefined);
      }
    },
    [load, message],
  );

  return (
    <>
      <Alert
        title="登记运营中的裸机服务器"
        description="私有地址必填；root 公钥用于安装基础环境。查看权限为 staff，增删改权限为 admin，均经 go-zero 鉴权并写审计日志。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      {loadError ? (
        <Alert
          title="机器列表加载失败"
          description={loadError}
          type="error"
          showIcon
          closable
          onClose={() => setLoadError(undefined)}
          style={{ marginBottom: 16 }}
        />
      ) : null}
      <Card
        title={`机器列表（${machines.length}）`}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={() => void load()}
            >
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建机器
            </Button>
          </Space>
        }
      >
        <MachineTable
          machines={machines}
          loading={loading}
          deletingId={deletingId}
          onEdit={openEdit}
          onDelete={(id) => void remove(id)}
          onExec={setExecMachine}
        />
      </Card>
      <MachineFormModal
        open={modalOpen}
        editing={editing}
        submitting={submitting}
        onSubmit={(values) => void submit(values)}
        onCancel={() => setModalOpen(false)}
      />
      <MachineExecDrawer
        machine={execMachine}
        open={Boolean(execMachine)}
        onClose={() => setExecMachine(undefined)}
      />
    </>
  );
};

const Machines: React.FC = () => (
  <PageContainer title="机器列表">
    <AntdApp>
      <MachinesContent />
    </AntdApp>
  </PageContainer>
);

export default Machines;
