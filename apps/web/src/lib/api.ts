export type HealthResponse = {
  status: "ok";
  service: string;
};

const apiBaseUrl = process.env.NEXUS_API_INTERNAL_URL ?? "http://127.0.0.1:8080";

export async function getApiHealth(): Promise<HealthResponse | null> {
  try {
    const response = await fetch(`${apiBaseUrl}/healthz`, {
      signal: AbortSignal.timeout(3000),
    });

    if (!response.ok) {
      return null;
    }

    const data: unknown = await response.json();
    if (typeof data !== "object" || data === null) {
      return null;
    }

    const { status, service } = data as Record<string, unknown>;
    if (status !== "ok" || typeof service !== "string") {
      return null;
    }

    return { status, service };
  } catch {
    return null;
  }
}
