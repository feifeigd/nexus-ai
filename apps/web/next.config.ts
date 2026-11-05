import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  cacheComponents: true,
  partialPrefetching: true,
  async rewrites() {
    return [
      {
        source: "/api/auth/:path*",
        destination: "http://127.0.0.1:8080/auth/:path*",
      },
      {
        source: "/api/usage",
        destination: "http://127.0.0.1:8080/usage",
      },
      {
        source: "/api/tenants/:path*",
        destination: "http://127.0.0.1:8080/tenants/:path*",
      },
    ];
  },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
