import {
  LogoutOutlined,
  SettingOutlined,
  SkinOutlined,
} from '@ant-design/icons';
import { history, useModel } from '@umijs/max';
import type { MenuProps } from 'antd';
import { Spin } from 'antd';
import React, { startTransition } from 'react';
import { outLogin } from '@/services/ant-design-pro/api';
import HeaderDropdown from '../HeaderDropdown';

type GlobalHeaderRightProps = {
  children?: React.ReactNode;
};

const ROLE_LABELS: Record<string, string> = {
  admin: '管理员',
  auditor: '审核员',
  readonly: '只读',
  user: '普通用户',
};

const baseMenuItems: MenuProps['items'] = [
  {
    key: 'settings',
    icon: <SettingOutlined />,
    label: '个人设置',
  },
  {
    key: 'theme',
    icon: <SkinOutlined />,
    label: '主题设置',
  },
  {
    type: 'divider' as const,
  },
  {
    key: 'logout',
    icon: <LogoutOutlined />,
    label: '退出登录',
  },
];

const loginOut = async () => {
  try {
    await outLogin();
  } catch {
    // Local logout has already cleared user state; redirect should still proceed.
  }
  const { search, pathname } = window.location;
  const urlParams = new URL(window.location.href).searchParams;
  const searchParams = new URLSearchParams({
    redirect: pathname + search,
  });
  const redirect = urlParams.get('redirect');
  if (window.location.pathname !== '/user/login' && !redirect) {
    history.replace({
      pathname: '/user/login',
      search: searchParams.toString(),
    });
  }
};

export const AvatarDropdown: React.FC<GlobalHeaderRightProps> = ({
  children,
}) => {
  const { initialState, setInitialState } = useModel('@@initialState');

  const onMenuClick: MenuProps['onClick'] = (event) => {
    const { key } = event;
    if (key === 'logout') {
      startTransition(() => {
        setInitialState((s) => ({ ...s, currentUser: undefined }));
      });
      loginOut();
      return;
    }
    if (key === 'theme') {
      setInitialState((s) => ({ ...s, settingDrawerOpen: true }));
      return;
    }
    history.push(`/account/${key}`);
  };

  if (!initialState) {
    return <Spin size="small" />;
  }

  const { currentUser } = initialState;

  if (!currentUser) {
    return <Spin size="small" />;
  }

  const role = (currentUser as { role?: string }).role;
  const roleLabel = ROLE_LABELS[role ?? ''] ?? role ?? '未知角色';
  const items: MenuProps['items'] = [
    {
      key: 'user-info',
      label: `${currentUser.name ?? ''} · ${roleLabel}`,
      disabled: true,
    },
    { type: 'divider' as const },
    ...(baseMenuItems ?? []),
  ];

  return (
    <HeaderDropdown
      placement="bottomRight"
      menu={{
        selectedKeys: [],
        onClick: onMenuClick,
        items,
      }}
      arrow
    >
      {children}
    </HeaderDropdown>
  );
};
