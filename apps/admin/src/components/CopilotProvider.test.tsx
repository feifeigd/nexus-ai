import type React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { CopilotProvider } from './CopilotProvider';

vi.mock('@copilotkit/react-core/v2', () => ({
  CopilotKitProvider: ({
    children,
    runtimeUrl,
    useSingleEndpoint,
  }: {
    children: React.ReactNode;
    runtimeUrl: string;
    useSingleEndpoint: boolean;
  }) => (
    <div
      data-runtime-url={runtimeUrl}
      data-single-endpoint={String(useSingleEndpoint)}
    >
      {children}
    </div>
  ),
}));

describe('CopilotProvider', () => {
  it('connects to the shared runtime with separate agent endpoints', () => {
    const html = renderToStaticMarkup(
      <CopilotProvider>
        <span>后台聊天</span>
      </CopilotProvider>,
    );

    expect(html).toContain('data-runtime-url="/api/copilotkit"');
    expect(html).toContain('data-single-endpoint="false"');
    expect(html).toContain('后台聊天');
  });
});
