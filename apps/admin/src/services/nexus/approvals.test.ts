import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  type Approval,
  decideApproval,
  listApprovals,
} from './approvals';

const approval: Approval = {
  id: 'approval-1',
  threadId: 'thread-1',
  runId: 'run-1',
  toolCallId: 'call-1',
  toolName: 'create_ticket',
  arguments: { title: '网络故障', description: '无法访问办公系统' },
  status: 'pending',
  createdAt: '2026-10-30T02:00:00Z',
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('approval API', () => {
  it('loads approvals through the same-origin go-zero proxy', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ approvals: [approval] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await expect(listApprovals()).resolves.toEqual([approval]);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/nexus/approvals',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    );
  });

  it.each(['approve', 'reject'] as const)(
    'posts the %s decision without a request body',
    async (decision) => {
      const fetchMock = vi.fn().mockResolvedValue(
        new Response(JSON.stringify(approval), { status: 200 }),
      );
      vi.stubGlobal('fetch', fetchMock);

      await decideApproval('approval/1', decision);
      expect(fetchMock).toHaveBeenCalledWith(
        `/api/nexus/approvals/${encodeURIComponent('approval/1')}/${decision}`,
        expect.objectContaining({ method: 'POST' }),
      );
    },
  );

  it('surfaces backend decision errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response('approval has already been decided\n', { status: 409 }),
      ),
    );

    await expect(decideApproval('approval-1', 'approve')).rejects.toThrow(
      'approval has already been decided',
    );
  });
});
