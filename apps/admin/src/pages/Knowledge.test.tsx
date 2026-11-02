import { App as AntdApp } from 'antd';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import {
  IngestionFeedback,
  KnowledgeContent,
  KnowledgeDocumentTable,
  type KnowledgeDocument,
} from './Knowledge';

const document: KnowledgeDocument = {
  documentId: 'document-1',
  title: 'NexusAI 测试运维手册',
  createdAt: '2026-11-02T04:00:00Z',
  updatedAt: '2026-11-02T04:00:00Z',
};

describe('Knowledge', () => {
  it('renders the document ingestion form and processing contract', () => {
    const html = renderToStaticMarkup(
      <AntdApp>
        <KnowledgeContent />
      </AntdApp>,
    );

    expect(html).toContain('文档标题');
    expect(html).toContain('文档内容');
    expect(html).toContain('导入并生成向量');
    expect(html).toContain('Unicode 安全分块');
    expect(html).toContain('1536 维向量');
    expect(html).toContain('完整记录见下方列表');
    expect(html).toContain('已导入文档');
  });

  it('shows the persisted document ID after a successful import', () => {
    const html = renderToStaticMarkup(
      <IngestionFeedback documentId="document-1" onClose={vi.fn()} />,
    );

    expect(html).toContain('文档导入成功');
    expect(html).toContain('document-1');
    expect(html).toContain('事务写入');
  });

  it('shows an inline ingestion error', () => {
    const html = renderToStaticMarkup(
      <IngestionFeedback error="Embedding 服务不可用" onClose={vi.fn()} />,
    );

    expect(html).toContain('文档导入失败');
    expect(html).toContain('Embedding 服务不可用');
  });

  it('shows imported document metadata in the list', () => {
    const html = renderToStaticMarkup(<KnowledgeDocumentTable documents={[document]} />);

    expect(html).toContain('NexusAI 测试运维手册');
    expect(html).toContain('document-1');
    expect(html).toContain('导入时间');
  });

  it('shows an empty document list state', () => {
    const html = renderToStaticMarkup(<KnowledgeDocumentTable documents={[]} />);

    expect(html).toContain('暂无已导入文档');
  });
});
