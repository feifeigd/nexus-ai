import { request } from '@umijs/max';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  approveApproval,
  listApprovals,
  rejectApproval,
} from './nexus/approval';

vi.mock('@umijs/max', () => ({ request: vi.fn() }));

const requestMock = vi.mocked(request);
const options = { baseURL: '/api/nexus' };

beforeEach(() => {
  requestMock.mockReset();
  requestMock.mockResolvedValue({});
});

describe('generated Nexus API client', () => {
  it('loads approvals through the same-origin go-zero proxy', async () => {
    await listApprovals(options);

    expect(requestMock).toHaveBeenCalledWith(
      '/approvals',
      expect.objectContaining({ method: 'GET', baseURL: '/api/nexus' }),
    );
  });

  it.each([
    ['approve', approveApproval],
    ['reject', rejectApproval],
  ] as const)('posts the %s decision without a request body', async (decision, operation) => {
    await operation({ id: 'approval-1' }, options);

    expect(requestMock).toHaveBeenCalledWith(
      `/approvals/approval-1/${decision}`,
      expect.objectContaining({
        method: 'POST',
        baseURL: '/api/nexus',
        params: {},
      }),
    );
  });
});
