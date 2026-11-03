"use client";

import "@copilotkit/react-core/v2/styles.css";
import { CopilotKitProvider } from "@copilotkit/react-core/v2";

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <CopilotKitProvider
      runtimeUrl="/api/copilotkit"
      useSingleEndpoint={false}
    >
      {children}
    </CopilotKitProvider>
  );
}
