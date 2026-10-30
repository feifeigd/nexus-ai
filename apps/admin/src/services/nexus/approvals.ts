export type ApprovalStatus = 'pending' | 'approved' | 'rejected';
export type ApprovalDecision = 'approve' | 'reject';

export interface CreateTicketInput {
  title: string;
  description: string;
}

export interface Approval {
  id: string;
  threadId: string;
  runId: string;
  toolCallId: string;
  toolName: string;
  arguments: CreateTicketInput;
  status: ApprovalStatus;
  ticketId?: string;
  createdAt: string;
  decidedAt?: string;
}

interface ApprovalListResponse {
  approvals: Approval[];
}

const NEXUS_API_BASE = '/api/nexus';

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${NEXUS_API_BASE}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...init?.headers,
    },
  });
  const text = await response.text();
  let body: unknown = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
  }
  if (!response.ok) {
    const detail =
      typeof body === 'string'
        ? body.trim()
        : body && typeof body === 'object' && 'message' in body
          ? String(body.message)
          : '';
    throw new Error(detail || `请求失败（HTTP ${response.status}）`);
  }
  return body as T;
}

export async function listApprovals(): Promise<Approval[]> {
  const result = await requestJSON<ApprovalListResponse>('/approvals');
  return result.approvals;
}

export function decideApproval(
  id: string,
  decision: ApprovalDecision,
): Promise<Approval> {
  return requestJSON<Approval>(
    `/approvals/${encodeURIComponent(id)}/${decision}`,
    { method: 'POST' },
  );
}
