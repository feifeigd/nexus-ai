"use client";

import "@copilotkit/react-core/v2/styles.css";
import { CopilotKitProvider } from "@copilotkit/react-core/v2";
import { getToken } from "../lib/auth";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      useSingleEndpoint={false}
      headers={() => {
        const token = getToken();
        return token ? { Authorization: `Bearer ${token}` } : {};
      }}
    >
      {children}
    </CopilotKitProvider>
  );
}
