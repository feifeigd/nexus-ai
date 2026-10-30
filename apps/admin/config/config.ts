import { existsSync } from 'node:fs';
import { join } from 'node:path';
import { loadEnvFile } from 'node:process';
import { defineConfig } from '@umijs/max';
import defaultSettings from './defaultSettings';
import proxy from './proxy';
import routes from './routes';

const ENV_FILE_PATH = join(__dirname, '../.env');
if (existsSync(ENV_FILE_PATH)) {
  loadEnvFile(ENV_FILE_PATH);
}

const { UMI_ENV = 'dev' } = process.env;
const PUBLIC_PATH = '/';
const OPENAPI_SCHEMA_PATH =
  process.env.OPENAPI_SCHEMA_PATH || join(__dirname, '../schema/openapi.json');

export default defineConfig({
  alias: {
    '@root': join(__dirname, '..'),
  },
  hash: true,
  publicPath: PUBLIC_PATH,
  routes,
  ignoreMomentLocale: true,
  proxy: proxy[UMI_ENV as keyof typeof proxy],
  fastRefresh: true,
  routePrefetch: {},
  manifest: {},
  model: {},
  initialState: {},
  title: 'NexusAI Admin',
  layout: {
    locale: true,
    ...defaultSettings,
  },
  moment2dayjs: {
    preset: 'antd',
    plugins: ['duration', 'relativeTime'],
  },
  locale: {
    default: 'zh-CN',
    antd: true,
    baseNavigator: true,
  },
  antd: {
    appConfig: {},
    configProvider: {
      variant: 'filled',
      theme: {
        token: {
          fontFamily: 'AlibabaSans, sans-serif',
        },
      },
    },
  },
  request: {},
  access: {},
  plugins: ['@umijs/max-plugin-openapi'],
  openAPI: existsSync(OPENAPI_SCHEMA_PATH)
    ? [
        {
          requestLibPath: "import { request } from '@umijs/max'",
          schemaPath: OPENAPI_SCHEMA_PATH,
          projectName: 'nexus',
          mock: false,
        },
      ]
    : [],
  headScripts: [
    { src: join(PUBLIC_PATH, 'scripts/loading.js'), async: true },
  ],
  mock: {
    include: ['src/pages/**/_mock.ts'],
  },
  exportStatic: {},
  esbuildMinifyIIFE: true,
  chainWebpack(config) {
    config.target(['web', 'es2017']);
  },
  define: {
    'process.env.CI': process.env.CI,
    'process.env.COMMIT_HASH': process.env.COMMIT_HASH || '',
    __APP_VERSION__: require('./../package.json').version,
    __UMI_VERSION__: require('@umijs/max/package.json').version,
    __UTOO_VERSION__: require('@utoo/pack/package.json').version,
  },
});
