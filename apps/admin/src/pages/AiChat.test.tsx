import { readFileSync } from 'node:fs';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { AiChatContent } from './AiChat';

const mocks = vi.hoisted(() => ({
  copilotChat: vi.fn(),
  defaultRenderTool: vi.fn(),
}));

vi.mock('@copilotkit/react-core/v2', () => ({
  CopilotChat: (props: Record<string, unknown>) => {
    mocks.copilotChat(props);
    return <div>Mock Chat</div>;
  },
  useDefaultRenderTool: mocks.defaultRenderTool,
}));

describe('AiChatContent', () => {
  it('connects the default agent and labels the configurable model boundary', () => {
    const html = renderToStaticMarkup(<AiChatContent />);

    expect(html).toContain('NexusAI 管理助手');
    expect(html).toContain('class="nexus-ai-chat"');
    expect(html).toContain('回复来源由 go-zero 配置');
    expect(html).toContain('不会未经批准执行后台操作');
    expect(html).not.toContain('NexusAI Mock Agent');
    expect(mocks.defaultRenderTool).toHaveBeenCalledOnce();
    expect(mocks.copilotChat).toHaveBeenCalledWith(
      expect.objectContaining({
        agentId: 'default',
        labels: expect.objectContaining({
          chatDisclaimerText: '回复来源由 go-zero 配置；未经人工批准，不会执行真实后台操作。',
          welcomeMessageText: '向 NexusAI 管理后台 AI 助手发送一条消息。',
        }),
      }),
    );
  });

  it('contains long tool payloads within the chat width', () => {
    const styles = readFileSync(new URL('./AiChat.less', import.meta.url), 'utf8');

    expect(styles).toContain("[data-testid='copilot-tool-render']");
    expect(styles).toContain('grid-template-columns: minmax(0, 1fr);');
    expect(styles).toContain('white-space: pre-wrap;');
    expect(styles).toContain('overflow-wrap: anywhere;');
  });
});
