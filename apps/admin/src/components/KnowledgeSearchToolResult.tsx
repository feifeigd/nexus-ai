import { SearchOutlined } from '@ant-design/icons';
import { Alert, Card, Collapse, Empty, Space, Spin, Tag, Typography } from 'antd';
import type React from 'react';
import './KnowledgeSearchToolResult.less';

export interface KnowledgeSearchMatch {
  documentId: string;
  chunkId: string;
  ordinal: number;
  content: string;
  distance: number;
}

export interface ParsedKnowledgeSearchResult {
  matches: KnowledgeSearchMatch[];
  error?: string;
}

const INVALID_RESULT_MESSAGE = '知识检索结果格式异常，请检查服务端工具响应。';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}

function parseMatch(value: unknown): KnowledgeSearchMatch | null {
  if (!isRecord(value)) return null;
  if (
    !isNonEmptyString(value.documentId) ||
    !isNonEmptyString(value.chunkId) ||
    !isNonEmptyString(value.content) ||
    !Number.isInteger(value.ordinal) ||
    (value.ordinal as number) < 0 ||
    typeof value.distance !== 'number' ||
    !Number.isFinite(value.distance) ||
    value.distance < 0
  ) {
    return null;
  }
  return {
    documentId: value.documentId,
    chunkId: value.chunkId,
    ordinal: value.ordinal as number,
    content: value.content,
    distance: value.distance,
  };
}

export function parseKnowledgeSearchResult(result: string): ParsedKnowledgeSearchResult {
  try {
    const decoded: unknown = JSON.parse(result);
    if (!isRecord(decoded) || !Array.isArray(decoded.matches)) {
      return { matches: [], error: INVALID_RESULT_MESSAGE };
    }
    const matches = decoded.matches.map(parseMatch);
    if (matches.some((match) => match === null)) {
      return { matches: [], error: INVALID_RESULT_MESSAGE };
    }
    return { matches: matches as KnowledgeSearchMatch[] };
  } catch {
    return { matches: [], error: INVALID_RESULT_MESSAGE };
  }
}

export interface KnowledgeSearchToolResultProps {
  status: 'inProgress' | 'executing' | 'complete';
  query?: string;
  result?: string;
  defaultExpanded?: boolean;
}

export const KnowledgeSearchToolResult: React.FC<KnowledgeSearchToolResultProps> = ({
  status,
  query,
  result,
  defaultExpanded = false,
}) => {
  if (status !== 'complete') {
    return (
      <Card size="small" className="nexus-knowledge-search-card">
        <Space>
          <Spin size="small" />
          <span>{status === 'inProgress' ? '正在准备知识检索…' : '正在检索企业知识库…'}</span>
        </Space>
        {query ? (
          <Typography.Paragraph type="secondary" className="nexus-knowledge-search-query">
            查询：{query}
          </Typography.Paragraph>
        ) : null}
      </Card>
    );
  }

  const parsed = parseKnowledgeSearchResult(result ?? '');
  return (
    <Collapse
      size="small"
      className="nexus-knowledge-search-card"
      defaultActiveKey={defaultExpanded ? ['result'] : []}
      items={[
        {
          key: 'result',
          label: (
            <Space>
              <SearchOutlined />
              知识检索结果
            </Space>
          ),
          extra: parsed.error ? undefined : <Tag color="blue">{parsed.matches.length} 个分块</Tag>,
          children: (
            <>
              {query ? (
                <Typography.Paragraph type="secondary" className="nexus-knowledge-search-query">
                  查询：{query}
                </Typography.Paragraph>
              ) : null}
              {parsed.error ? (
                <Alert type="warning" showIcon title={parsed.error} />
              ) : parsed.matches.length === 0 ? (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="未找到匹配的知识分块"
                />
              ) : (
                <Space orientation="vertical" size={12} className="nexus-knowledge-search-matches">
                  {parsed.matches.map((match, index) => (
                    <Card
                      key={match.chunkId}
                      size="small"
                      type="inner"
                      className="nexus-knowledge-search-match"
                      title={`命中 ${index + 1}`}
                      extra={<Tag>距离 {match.distance.toFixed(4)}</Tag>}
                    >
                      <Space orientation="vertical" size={8} style={{ width: '100%' }}>
                        <div>
                          <Typography.Text type="secondary">文档 ID：</Typography.Text>
                          <Typography.Text code copyable>
                            {match.documentId}
                          </Typography.Text>
                        </div>
                        <Space size={16} wrap>
                          <Typography.Text type="secondary">
                            分块序号：{match.ordinal}
                          </Typography.Text>
                          <Typography.Text type="secondary">分块 ID：{match.chunkId}</Typography.Text>
                        </Space>
                        <Typography.Paragraph className="nexus-knowledge-search-content">
                          {match.content}
                        </Typography.Paragraph>
                      </Space>
                    </Card>
                  ))}
                </Space>
              )}
            </>
          ),
        },
      ]}
    />
  );
};
