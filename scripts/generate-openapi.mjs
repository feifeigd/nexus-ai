import {
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { spawnSync } from 'node:child_process';
import { basename, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const apiRoot = `${root}services/api`;
const apiDir = join(apiRoot, 'api');
const schemaDir = `${root}apps/admin/schema`;
const schemaPath = join(schemaDir, 'openapi.json');
mkdirSync(schemaDir, { recursive: true });
const tempDir = mkdtempSync(join(schemaDir, '.goctl-'));

function runGoctl(apiFile) {
  const outputDir = join(tempDir, apiFile);
  mkdirSync(outputDir, { recursive: true });
  const result = spawnSync(
    'go',
    [
      'run',
      'github.com/zeromicro/go-zero/tools/goctl@v1.10.2',
      'api',
      'swagger',
      '--api',
      `api/${apiFile}`,
      '--dir',
      outputDir,
      '--filename',
      'openapi',
    ],
    { cwd: apiRoot, stdio: 'inherit' },
  );
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`goctl failed for ${apiFile} with exit code ${result.status}`);
  }
  return JSON.parse(readFileSync(join(outputDir, 'openapi.json'), 'utf8'));
}

function operations(schema) {
  return Object.values(schema.paths ?? {}).flatMap((pathItem) =>
    Object.values(pathItem).filter((operation) => operation?.operationId),
  );
}

try {
  const schema = runGoctl('nexus.api');

  // Preserve the source contract for each operation so the TS generator can
  // create one service file per .api file rather than one file per function.
  const sourceByOperation = new Map();
  const contractFiles = readdirSync(apiDir)
    .filter((file) => file.endsWith('.api') && file !== 'nexus.api')
    .sort();
  for (const contractFile of contractFiles) {
    const sourceSchema = runGoctl(contractFile);
    const source = basename(contractFile, '.api');
    for (const operation of operations(sourceSchema)) {
      const previous = sourceByOperation.get(operation.operationId);
      if (previous && previous !== source) {
        throw new Error(
          `Operation ${operation.operationId} is duplicated in ${previous}.api and ${source}.api`,
        );
      }
      sourceByOperation.set(operation.operationId, source);
    }
  }
  for (const operation of operations(schema)) {
    const source = sourceByOperation.get(operation.operationId);
    if (!source) {
      throw new Error(`No source contract found for ${operation.operationId}`);
    }
    operation['x-nexus-contract'] = source;
  }

  // goctl writes the current time to x-date; remove it for deterministic output.
  delete schema['x-date'];

  const definitions = {};
  const definitionName = (operationId, suffix) =>
    `${operationId[0].toUpperCase()}${operationId.slice(1)}${suffix}`;
  const reference = (name) => ({ $ref: `#/definitions/${name}` });

  for (const [path, pathItem] of Object.entries(schema.paths ?? {})) {
    const pathNames = [...path.matchAll(/\{([^}]+)\}/g)].map(
      (match) => match[1],
    );
    for (const operation of Object.values(pathItem)) {
      // Lift inline payloads into named models so Umi emits reusable TS types.
      for (const parameter of operation.parameters ?? []) {
        if (
          parameter.in === 'body' &&
          parameter.schema &&
          !parameter.schema.$ref
        ) {
          const name = definitionName(operation.operationId, 'Input');
          definitions[name] = parameter.schema;
          parameter.schema = reference(name);
        }
      }
      const successResponse = operation.responses?.['200'];
      if (successResponse?.schema && !successResponse.schema.$ref) {
        const name = definitionName(operation.operationId, 'Response');
        definitions[name] = successResponse.schema;
        successResponse.schema = reference(name);
      }

      // goctl leaves path placeholders undeclared, while Umi needs declarations
      // to generate TypeScript parameter types.
      const parameters = operation.parameters ?? [];
      const declared = new Set(
        parameters
          .filter((parameter) => parameter.in === 'path')
          .map((parameter) => parameter.name),
      );
      operation.parameters = [
        ...parameters,
        ...pathNames
          .filter((name) => !declared.has(name))
          .map((name) => ({
            name,
            in: 'path',
            required: true,
            type: 'string',
          })),
      ];
    }
  }
  if (Object.keys(definitions).length > 0) {
    schema.definitions = definitions;
  }

  writeFileSync(schemaPath, `${JSON.stringify(schema, null, 2)}\n`);
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
