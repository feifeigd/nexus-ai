import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import {
  KnowledgeSearchToolResult,
  parseKnowledgeSearchResult,
} from './KnowledgeSearchToolResult';

const match = {
  documentId: 'document-1',
  chunkId: 'chunk-1',
  ordinal: 0,
  content: '第一章：蓝雀告警排查。',
  distance: 0.123456,
};

describe('parseKnowledgeSearchResult', () => {
  it('parses valid knowledge matches without changing their source metadata', () => {
    const parsed = parseKnowledgeSearchResult(JSON.stringify({ matches: [match] }));

    expect(parsed.error).toBeUndefined();
    expect(parsed.matches).toEqual([match]);
  });

  it('accepts an empty match list', () => {
    expect(parseKnowledgeSearchResult('{"matches":[]}')).toEqual({ matches: [] });
  });

  it.each([
    '{invalid json',
    '{"matches":null}',
    '{"matches":[{}]}',
    JSON.stringify({ matches: [{ ...match, distance: -0.1 }] }),
    JSON.stringify({ matches: [{ ...match, ordinal: 0.5 }] }),
    JSON.stringify({ matches: [{ ...match, documentId: '' }] }),
  ])('rejects malformed tool output instead of rendering partial data', (result) => {
    const parsed = parseKnowledgeSearchResult(result);

    expect(parsed.matches).toEqual([]);
    expect(parsed.error).toContain('格式异常');
  });
});

describe('KnowledgeSearchToolResult', () => {
  it('collapses completed retrieval results by default', () => {
    const html = renderToStaticMarkup(
      <KnowledgeSearchToolResult
        status="complete"
        result={JSON.stringify({ matches: [match] })}
      />,
    );

    expect(html).toContain('ant-collapse');
    expect(html).toContain('知识检索结果');
    expect(html).not.toContain('ant-collapse-item-active');
  });

  it('shows query, source IDs, chunk ordinal, content and cosine distance when expanded', () => {
    const html = renderToStaticMarkup(
      <KnowledgeSearchToolResult
        status="complete"
        query="蓝雀告警如何排查？"
        result={JSON.stringify({ matches: [match] })}
        defaultExpanded
      />,
    );

    expect(html).toContain('ant-collapse-item-active');
    expect(html).toContain('知识检索结果');
    expect(html).toContain('蓝雀告警如何排查？');
    expect(html).toContain('document-1');
    expect(html).toContain('chunk-1');
    expect(html).toContain('分块序号：');
    expect(html).toContain('0.1235');
    expect(html).toContain('第一章：蓝雀告警排查。');
  });

  it('distinguishes an empty retrieval result from a malformed result', () => {
    const emptyHtml = renderToStaticMarkup(
      <KnowledgeSearchToolResult status="complete" result='{"matches":[]}' defaultExpanded />,
    );
    const errorHtml = renderToStaticMarkup(
      <KnowledgeSearchToolResult
        status="complete"
        result="sensitive raw output"
        defaultExpanded
      />,
    );

    expect(emptyHtml).toContain('未找到匹配的知识分块');
    expect(errorHtml).toContain('知识检索结果格式异常');
    expect(errorHtml).not.toContain('sensitive raw output');
  });

  it('shows progress while the tool is executing', () => {
    const html = renderToStaticMarkup(
      <KnowledgeSearchToolResult status="executing" query="运维手册" />,
    );

    expect(html).toContain('正在检索企业知识库');
    expect(html).toContain('运维手册');
    expect(html).not.toContain('ant-collapse');
  });
});
