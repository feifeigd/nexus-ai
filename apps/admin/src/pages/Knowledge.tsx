import {
  CheckCircleOutlined,
  CloudUploadOutlined,
  DatabaseOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import {
  Alert,
  App as AntdApp,
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  Row,
  Space,
  Typography,
} from 'antd';
import type React from 'react';
import { useCallback, useState } from 'react';
import { ingestDocument } from '@/services/nexus/knowledge';

const NEXUS_API_OPTIONS = { baseURL: '/api/nexus', skipErrorHandler: true };

function errorMessage(error: unknown) {
  const status = (
    error as { response?: { status?: number } } | undefined
  )?.response?.status;
  if (status === 503) {
    return '服务端尚未配置 NEXUS_OPENAI_EMBEDDING_MODEL，暂时无法生成文档向量。';
  }
  if (status === 400) {
    return '请检查文档标题和内容：标题为 1～200 字，内容不能为空。';
  }
  return error instanceof Error
    ? error.message
    : '文档导入失败，请稍后重试。';
}

interface IngestionFeedbackProps {
  documentId?: string;
  error?: string;
  onClose: () => void;
}

export const IngestionFeedback: React.FC<IngestionFeedbackProps> = ({
  documentId,
  error,
  onClose,
}) => {
  if (error) {
    return (
      <Alert
        title="文档导入失败"
        description={error}
        type="error"
        showIcon
        closable
        onClose={onClose}
        style={{ marginBottom: 16 }}
      />
    );
  }
  if (documentId) {
    return (
      <Alert
        title="文档导入成功"
        description={
          <Space orientation="vertical" size={4}>
            <span>文档已完成分块、向量生成和事务写入。</span>
            <Typography.Text code copyable>
              {documentId}
            </Typography.Text>
          </Space>
        }
        type="success"
        showIcon
        style={{ marginBottom: 16 }}
      />
    );
  }
  return null;
};

export const KnowledgeContent: React.FC = () => {
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<API.IngestDocumentInput>();
  const [submitting, setSubmitting] = useState(false);
  const [documentId, setDocumentId] = useState<string>();
  const [error, setError] = useState<string>();

  const submit = useCallback(
    async (values: API.IngestDocumentInput) => {
      setSubmitting(true);
      setError(undefined);
      setDocumentId(undefined);
      try {
        const response = await ingestDocument(values, NEXUS_API_OPTIONS);
        if (!response.documentId) {
          throw new Error('服务端未返回文档 ID，请检查服务日志。');
        }
        setDocumentId(response.documentId);
        form.resetFields();
        await message.success('文档已成功导入知识库。');
      } catch (requestError) {
        setError(errorMessage(requestError));
      } finally {
        setSubmitting(false);
      }
    },
    [form, message],
  );

  return (
    <>
      <Alert
        title="文档由 go-zero 统一分块、生成向量并写入知识库"
        description="生产检索使用 PostgreSQL + pgvector。标题最多 200 字；导入期间请保持页面打开，不要重复提交。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <IngestionFeedback
        documentId={documentId}
        error={error}
        onClose={() => setError(undefined)}
      />
      <Row gutter={[24, 24]}>
        <Col xs={24} xl={14}>
          <Card
            title={
              <Space>
                <FileTextOutlined />
                导入文档
              </Space>
            }
          >
            <Form<API.IngestDocumentInput>
              form={form}
              layout="vertical"
              requiredMark="optional"
              onFinish={(values) => void submit(values)}
            >
              <Form.Item
                name="title"
                label="文档标题"
                rules={[
                  { required: true, whitespace: true, message: '请输入文档标题' },
                  { max: 200, message: '文档标题不能超过 200 字' },
                ]}
              >
                <Input
                  placeholder="例如：网络运维手册"
                  maxLength={200}
                  showCount
                  disabled={submitting}
                />
              </Form.Item>
              <Form.Item
                name="content"
                label="文档内容"
                rules={[
                  { required: true, whitespace: true, message: '请输入文档内容' },
                ]}
              >
                <Input.TextArea
                  placeholder="粘贴需要分块并生成向量的正文内容"
                  autoSize={{ minRows: 10, maxRows: 20 }}
                  showCount
                  disabled={submitting}
                />
              </Form.Item>
              <Form.Item style={{ marginBottom: 0 }}>
                <Space>
                  <Button
                    type="primary"
                    htmlType="submit"
                    icon={<CloudUploadOutlined />}
                    loading={submitting}
                  >
                    导入并生成向量
                  </Button>
                  <Button
                    disabled={submitting}
                    onClick={() => {
                      form.resetFields();
                      setError(undefined);
                      setDocumentId(undefined);
                    }}
                  >
                    清空
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </Col>
        <Col xs={24} xl={10}>
          <Card
            title={
              <Space>
                <DatabaseOutlined />
                处理流程
              </Space>
            }
          >
            <Space orientation="vertical" size={16} style={{ width: '100%' }}>
              <div>
                <Typography.Text strong>1. Unicode 安全分块</Typography.Text>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  默认每块 800 字、重叠 120 字，避免按字节截断中文。
                </Typography.Paragraph>
              </div>
              <div>
                <Typography.Text strong>2. 生成 1536 维向量</Typography.Text>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  使用服务端配置的 OpenAI 兼容 Embedding 模型。
                </Typography.Paragraph>
              </div>
              <div>
                <Typography.Text strong>3. 文档与分块事务写入</Typography.Text>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  任一分块失败，整篇文档都不会保留。
                </Typography.Paragraph>
              </div>
              <Divider style={{ margin: 0 }} />
              <div>
                <Typography.Text strong>本次会话</Typography.Text>
                {documentId ? (
                  <Typography.Paragraph style={{ marginBottom: 0, marginTop: 8 }}>
                    <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                    已导入 1 篇文档，最新文档 ID 见上方成功提示。
                  </Typography.Paragraph>
                ) : (
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 8 }}>
                    尚未导入文档。这里仅显示本次会话结果，不是完整文档列表。
                  </Typography.Paragraph>
                )}
              </div>
            </Space>
          </Card>
        </Col>
      </Row>
    </>
  );
};

const Knowledge: React.FC = () => (
  <PageContainer title="知识库">
    <AntdApp>
      <KnowledgeContent />
    </AntdApp>
  </PageContainer>
);

export default Knowledge;
