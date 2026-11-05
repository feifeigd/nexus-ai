import { beforeEach, describe, expect, it, vi } from 'vitest';
import { clearAuth, setAuth } from './utils/auth';

// Mock all heavy dependencies before importing app
const mockReplace = vi.fn();
const mockHistory = {
  location: {
    pathname: '/welcome',
    search: '',
    hash: '',
  },
  replace: mockReplace,
};

vi.mock('@umijs/max', () => ({
  history: mockHistory,
  Link: ({ children }: any) => children,
}));

vi.mock('@/components', () => ({
  AvatarDropdown: () => null,
  DocLink: () => null,
  ErrorBoundary: ({ children }: any) => children,
  Footer: () => null,
  LangDropdown: () => null,
  OfflineBanner: () => null,
  VersionDropdown: () => null,
}));

vi.mock('@/components/CopilotProvider', () => ({
  CopilotProvider: ({ children }: any) => children,
}));

vi.mock('@ant-design/pro-components', () => ({
  SettingDrawer: () => null,
}));

vi.mock('@ant-design/icons', () => ({
  LinkOutlined: () => null,
}));

vi.mock('./requestErrorConfig', () => ({
  errorConfig: {},
}));

vi.mock('../config/defaultSettings', () => ({
  default: { navTheme: 'light' },
}));

describe('app getInitialState', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearAuth();
    mockHistory.location = {
      pathname: '/welcome',
      search: '',
      hash: '',
    };
  });

  it('should read the auth user from localStorage when not on login page', async () => {
    setAuth('token-1', { username: 'alice', role: 'user' });
    const { getInitialState } = await import('./app');

    const state = await getInitialState();

    expect(state.currentUser).toEqual({
      name: 'alice',
      username: 'alice',
      role: 'user',
    });
    expect(state.settingDrawerOpen).toBe(false);
    expect(state.fetchUserInfo).toBeDefined();
  });

  it('should redirect to login when not logged in', async () => {
    const { getInitialState } = await import('./app');

    const state = await getInitialState();

    expect(mockReplace).toHaveBeenCalledWith(
      expect.stringContaining('/user/login?redirect='),
    );
    expect(state.currentUser).toBeUndefined();
  });

  it('should not read the auth user on login page', async () => {
    mockHistory.location = {
      pathname: '/user/login',
      search: '',
      hash: '',
    };
    const { getInitialState } = await import('./app');

    const state = await getInitialState();

    expect(state.currentUser).toBeUndefined();
    expect(state.fetchUserInfo).toBeDefined();
    expect(mockReplace).not.toHaveBeenCalled();
  });

  it('should encode the redirect path correctly when not logged in', async () => {
    mockHistory.location = {
      pathname: '/admin/users',
      search: '?page=2',
      hash: '#section',
    };
    const { getInitialState } = await import('./app');

    await getInitialState();

    expect(mockReplace).toHaveBeenCalledWith(
      `/user/login?redirect=${encodeURIComponent('/admin/users?page=2#section')}`,
    );
  });

  it('should include default settings in initial state', async () => {
    setAuth('token-1', { username: 'alice', role: 'user' });
    const { getInitialState } = await import('./app');

    const state = await getInitialState();

    expect(state.settings).toEqual({ navTheme: 'light' });
  });

  it('fetchUserInfo should return the auth user on success', async () => {
    setAuth('token-1', { username: 'alice', role: 'user' });
    const { getInitialState } = await import('./app');

    const state = await getInitialState();
    const user = await state.fetchUserInfo?.();

    expect(user).toEqual({ name: 'alice', username: 'alice', role: 'user' });
  });
});
