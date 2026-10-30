import { generateService } from '@umijs/openapi';
import { fileURLToPath } from 'node:url';

const adminRoot = fileURLToPath(new URL('../', import.meta.url));
const schemaPath =
  process.env.OPENAPI_SCHEMA_PATH || `${adminRoot}schema/openapi.json`;

await generateService({
  requestLibPath: "import { request } from '@umijs/max'",
  schemaPath,
  projectName: 'nexus',
  serversPath: `${adminRoot}src/services`,
  hook: {
    // Group operations by their source .api contract, not by individual function.
    customFileNames: (operation) => [
      operation['x-nexus-contract'] || 'client',
    ],
  },
});
