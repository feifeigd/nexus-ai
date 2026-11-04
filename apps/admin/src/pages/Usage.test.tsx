import { App as AntdApp } from 'antd';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { UsageContent } from './Usage';

describe('Usage', () => {
  it('renders the usage summary and detail table', () => {
    const html = renderToStaticMarkup(
      <AntdApp>
        <UsageContent />
      </AntdApp>,
    );

    expect(html).toContain('对话 token 消耗明细');
    expect(html).toContain('对话次数');
    expect(html).toContain('输入 token');
    expect(html).toContain('输出 token');
    expect(html).toContain('合计 token');
    expect(html).toContain('明细（0）');
  });
});
