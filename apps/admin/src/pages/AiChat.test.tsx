import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { AiChatContent } from './AiChat';

const mocks = vi.hoisted(() => ({
  copilotChat: vi.fn(),
}));

vi.mock('@copilotkit/react-core/v2', () => ({
  CopilotChat: (props: Record<string, unknown>) => {
    mocks.copilotChat(props);
    return <div>Mock Chat</div>;
  },
}));

describe('AiChatContent', () => {
  it('connects the default agent and labels the mock-only boundary', () => {
    const html = renderToStaticMarkup(<AiChatContent />);

    expect(html).toContain('NexusAI Mock Agent');
    expect(html).toContain('不会执行后台操作');
    expect(mocks.copilotChat).toHaveBeenCalledWith(
      expect.objectContaining({
        agentId: 'default',
        labels: expect.objectContaining({
          chatDisclaimerText: '当前为 Mock 流式回复，不会执行真实后台操作。',
        }),
      }),
    );
  });
});
