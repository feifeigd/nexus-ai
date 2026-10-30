import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { TicketTable, type Ticket } from './Tickets';

const ticket: Ticket = {
  id: 'ticket-1',
  title: '网络故障',
  description: '无法访问办公系统',
  status: 'open',
  createdAt: '2026-10-31T02:00:00Z',
};

describe('TicketTable', () => {
  it('shows created ticket details and status', () => {
    const html = renderToStaticMarkup(<TicketTable tickets={[ticket]} />);

    expect(html).toContain('网络故障');
    expect(html).toContain('无法访问办公系统');
    expect(html).toContain('待处理');
    expect(html).toContain('ticket-1');
  });

  it('shows an empty state before any approval creates a ticket', () => {
    const html = renderToStaticMarkup(<TicketTable tickets={[]} />);

    expect(html).toContain('暂无已创建工单');
  });
});
