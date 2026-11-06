export default [
  {
    path: '/user',
    layout: false,
    routes: [
      {
        name: 'login',
        path: '/user/login',
        component: './user/login',
      },
    ],
  },
  {
    path: '/welcome',
    name: 'welcome',
    icon: 'smile',
    component: './Welcome',
  },
  {
    path: '/ai-chat',
    name: 'chatbot',
    icon: 'robot',
    component: './AiChat',
  },
  {
    path: '/knowledge',
    name: 'knowledge',
    icon: 'database',
    component: './Knowledge',
  },
  {
    path: '/usage',
    name: 'usage',
    icon: 'barChart',
    component: './Usage',
  },
  {
    path: '/approvals',
    name: 'approvals',
    icon: 'audit',
    component: './Approvals',
  },
  {
    path: '/tickets',
    name: 'tickets',
    icon: 'container',
    component: './Tickets',
  },
  {
    path: '/audit-logs',
    name: 'auditLogs',
    icon: 'fileText',
    component: './Audit',
  },
  {
    path: '/machines',
    name: 'machines',
    icon: 'cluster',
    component: './Machines',
  },
  {
    path: '/admin',
    name: 'admin',
    icon: 'crown',
    access: 'canAdmin',
    routes: [
      {
        path: '/admin',
        redirect: '/admin/sub-page',
      },
      {
        path: '/admin/sub-page',
        name: 'sub-page',
        component: './Admin',
      },
    ],
  },
  {
    path: '/',
    redirect: '/welcome',
  },
  {
    component: './exception/404',
    layout: false,
    path: './*',
  },
];
