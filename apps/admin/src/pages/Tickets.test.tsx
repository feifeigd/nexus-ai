import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { TicketTable, type Ticket } from './Tickets';

const ticket: Ticket = {
  id: 'ticket-1',
  title: '网络故障',
  description: '无法访问办公系统',
  status: 'open',
  createdAt: '2026-10-31T02:00:00Z',
};

describe('TicketTable', () => {
  it('shows created ticket details and next action', () => {
    const html = renderToStaticMarkup(
      <TicketTable tickets={[ticket]} onUpdateStatus={vi.fn()} />,
    );

    expect(html).toContain('网络故障');
    expect(html).toContain('无法访问办公系统');
    expect(html).toContain('待处理');
    expect(html).toContain('ticket-1');
    expect(html).toContain('开始处理');
  });

  it('shows only the action allowed by each workflow state', () => {
    const html = renderToStaticMarkup(
      <TicketTable
        tickets={[
          ticket,
          { ...ticket, id: 'ticket-2', status: 'in_progress' },
          { ...ticket, id: 'ticket-3', status: 'resolved' },
        ]}
        onUpdateStatus={vi.fn()}
      />,
    );

    expect(html).toContain('处理中');
    expect(html).toContain('标记完成');
    expect(html).toContain('已完成');
    expect(html).toContain('已结束');
  });

  it('shows an empty state before any approval creates a ticket', () => {
    const html = renderToStaticMarkup(
      <TicketTable tickets={[]} onUpdateStatus={vi.fn()} />,
    );

    expect(html).toContain('暂无已创建工单');
  });
});
