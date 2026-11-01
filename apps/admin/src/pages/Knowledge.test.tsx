import { App as AntdApp } from 'antd';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { IngestionFeedback, KnowledgeContent } from './Knowledge';

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
    expect(html).toContain('不是完整文档列表');
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
});
