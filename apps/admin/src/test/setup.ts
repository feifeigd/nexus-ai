// vitest 用 `environment: 'node'` 运行测试，此时没有浏览器的 localStorage。
// Node 22+ 在未正确配置 --localstorage-file 时会暴露一个 getItem 失效的实验性
// localStorage 全局，导致依赖 localStorage 的模块（utils/auth、requestErrorConfig）
// 在单测里抛 `localStorage.getItem is not a function`。这里注入一个内存版实现。
function memoryStorage(): Storage {
  let store = new Map<string, string>();
  return {
    get length() {
      return store.size;
    },
    clear() {
      store = new Map();
    },
    getItem(key) {
      return store.get(key) ?? null;
    },
    key(index) {
      return Array.from(store.keys())[index] ?? null;
    },
    removeItem(key) {
      store.delete(key);
    },
    setItem(key, value) {
      store.set(key, String(value));
    },
  };
}

for (const name of ['localStorage', 'sessionStorage'] as const) {
  Object.defineProperty(globalThis, name, {
    value: memoryStorage(),
    writable: true,
    configurable: true,
  });
}
