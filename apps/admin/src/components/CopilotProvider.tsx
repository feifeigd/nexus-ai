import { CopilotKitProvider } from '@copilotkit/react-core/v2';
import type React from 'react';
import { getToken } from '@/utils/auth';

export const COPILOT_RUNTIME_URL = '/api/copilotkit';

export const CopilotProvider: React.FC<React.PropsWithChildren> = ({
  children,
}) => (
  <CopilotKitProvider
    runtimeUrl={COPILOT_RUNTIME_URL}
    useSingleEndpoint={false}
    headers={() => {
      const token = getToken();
      return token ? { Authorization: `Bearer ${token}` } : {};
    }}
  >
    {children}
  </CopilotKitProvider>
);
