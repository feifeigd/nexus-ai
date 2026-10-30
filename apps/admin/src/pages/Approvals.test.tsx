import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import type { Approval } from '@/services/nexus/approvals';
import { ApprovalTable } from './Approvals';

const pendingApproval: Approval = {
  id: 'approval-1',
  threadId: 'thread-1',
  runId: 'run-1',
  toolCallId: 'call-1',
  toolName: 'create_ticket',
  arguments: { title: '网络故障', description: '无法访问办公系统' },
  status: 'pending',
  createdAt: '2026-10-30T02:00:00Z',
};

describe('ApprovalTable', () => {
  it('shows validated proposal arguments and pending actions', () => {
    const html = renderToStaticMarkup(
      <ApprovalTable approvals={[pendingApproval]} onDecide={vi.fn()} />,
    );

    expect(html).toContain('网络故障');
    expect(html).toContain('无法访问办公系统');
    expect(html).toContain('create_ticket');
    expect(html).toContain('待审批');
    expect(html).toContain('批准');
    expect(html).toContain('拒绝');
  });

  it('does not offer another decision for a completed approval', () => {
    const html = renderToStaticMarkup(
      <ApprovalTable
        approvals={[
          {
            ...pendingApproval,
            status: 'approved',
            ticketId: 'ticket-1',
            decidedAt: '2026-10-30T02:01:00Z',
          },
        ]}
        onDecide={vi.fn()}
      />,
    );

    expect(html).toContain('已批准');
    expect(html).toContain('ticket-1');
    expect(html).toContain('已完成');
    expect(html).not.toContain('确认批准');
    expect(html).not.toContain('确认拒绝');
  });
});
