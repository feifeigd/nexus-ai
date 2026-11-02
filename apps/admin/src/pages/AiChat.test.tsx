import { readFileSync } from 'node:fs';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { AiChatContent } from './AiChat';

const mocks = vi.hoisted(() => ({
  assistantMessage: vi.fn(),
  copilotChat: vi.fn(),
  defaultRenderTool: vi.fn(),
  renderTool: vi.fn(),
}));

vi.mock('@copilotkit/react-core/v2', () => ({
  CopilotChat: (props: Record<string, unknown>) => {
    mocks.copilotChat(props);
    return <div>Mock Chat</div>;
  },
  CopilotChatAssistantMessage: (props: Record<string, unknown>) => {
    mocks.assistantMessage(props);
    return <div>Mock Assistant Message</div>;
  },
  useDefaultRenderTool: mocks.defaultRenderTool,
  useRenderTool: mocks.renderTool,
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
    expect(mocks.renderTool).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'search_knowledge',
        agentId: 'default',
        parameters: expect.anything(),
        render: expect.any(Function),
      }),
    );
    expect(mocks.copilotChat).toHaveBeenCalledWith(
      expect.objectContaining({
        agentId: 'default',
        messageView: expect.objectContaining({ assistantMessage: expect.any(Function) }),
        labels: expect.objectContaining({
          chatDisclaimerText: '回复来源由 go-zero 配置；未经人工批准，不会执行真实后台操作。',
          welcomeMessageText: '向 NexusAI 管理后台 AI 助手发送一条消息。',
        }),
      }),
    );
  });

  it('keeps the native copy button for assistant messages with content', () => {
    renderToStaticMarkup(<AiChatContent />);
    const chatProps = mocks.copilotChat.mock.calls.at(-1)?.[0];
    if (!chatProps) throw new Error('CopilotChat was not rendered');
    const AssistantMessage = chatProps.messageView.assistantMessage;

    renderToStaticMarkup(
      <AssistantMessage message={{ id: 'assistant-1', role: 'assistant', content: '复制内容' }} />,
    );

    expect(mocks.assistantMessage.mock.calls.at(-1)?.[0].copyButton).toBeUndefined();
  });

  it('hides the copy button for tool-only assistant messages', () => {
    renderToStaticMarkup(<AiChatContent />);
    const chatProps = mocks.copilotChat.mock.calls.at(-1)?.[0];
    if (!chatProps) throw new Error('CopilotChat was not rendered');
    const AssistantMessage = chatProps.messageView.assistantMessage;

    renderToStaticMarkup(<AssistantMessage message={{ id: 'tool-1', role: 'assistant', content: '' }} />);

    const CopyButton = mocks.assistantMessage.mock.calls.at(-1)?.[0].copyButton;
    expect(CopyButton).toEqual(expect.any(Function));
    expect(renderToStaticMarkup(<CopyButton />)).toBe('');
  });

  it('contains long tool payloads within the chat width', () => {
    const styles = readFileSync(new URL('./AiChat.less', import.meta.url), 'utf8');

    expect(styles).toContain("[data-testid='copilot-tool-render']");
    expect(styles).toContain('grid-template-columns: minmax(0, 1fr);');
    expect(styles).toContain('white-space: pre-wrap;');
    expect(styles).toContain('overflow-wrap: anywhere;');
  });
});
