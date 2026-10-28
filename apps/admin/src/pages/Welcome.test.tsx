import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { WelcomeContent } from './Welcome';

describe('WelcomeContent', () => {
  it('renders the NexusAI administration overview', () => {
    const html = renderToStaticMarkup(<WelcomeContent />);

    expect(html).toContain('NexusAI 管理后台');
    expect(html).toContain('基于 Ant Design Pro 的独立管理控制台');
  });
});
