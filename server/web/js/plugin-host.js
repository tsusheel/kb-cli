/**
 * KB-CLI Web PluginHost SDK (v1.0.0)
 * Sandboxed, capability-controlled bridge between the core SPA and external plugins.
 */

class PluginHost {
  constructor() {
    this.routes = new Map();
    this.searchProviders = new Map();
    this.thingsPadLayouts = new Map();
    this.activePlugins = new Set();
  }

  /**
   * Creates a scoped host context for a specific plugin.
   * Plugins receive only this scoped object and cannot tamper with other plugins.
   */
  createScopedHost(pluginId) {
    const host = this;
    this.activePlugins.add(pluginId);

    return {
      pluginId,
      routes: {
        register(path, handler) {
          host.registerRoute(pluginId, path, handler);
        },
        navigate(path) {
          window.location.hash = path.startsWith('#') ? path : '#' + path;
        }
      },
      search: {
        registerProvider(providerFn) {
          host.searchProviders.set(pluginId, providerFn);
        }
      },
      thingsPad: {
        registerLayout(layoutDef) {
          host.thingsPadLayouts.set(pluginId, layoutDef);
        }
      },
      ui: {
        showToast(message, actionText, actionCallback) {
          import('./utils.js').then(m => m.showToast(message, actionText, actionCallback));
        },
        escapeHtml(str) {
          if (!str) return '';
          return String(str).replace(/[&<>"']/g, m => ({
            '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;'
          }[m]));
        }
      },
      api: {
        async request(endpoint, options = {}) {
          const cleanEndpoint = endpoint.startsWith('/') ? endpoint : '/' + endpoint;
          const url = `/api/plugins/${encodeURIComponent(pluginId)}${cleanEndpoint}`;
          const res = await fetch(url, {
            ...options,
            headers: {
              'Content-Type': 'application/json',
              ...(options.headers || {})
            }
          });
          if (!res.ok) {
            const errBody = await res.json().catch(() => ({ error: res.statusText }));
            throw new Error(errBody.error || `HTTP ${res.status}`);
          }
          return res.json();
        }
      }
    };
  }

  registerRoute(pluginId, path, handler) {
    const key = path.startsWith('#') ? path.slice(1) : path;
    this.routes.set(key, { pluginId, handler });
  }

  unregisterPlugin(pluginId) {
    for (const [path, r] of this.routes.entries()) {
      if (r.pluginId === pluginId) {
        this.routes.delete(path);
      }
    }
    this.searchProviders.delete(pluginId);
    this.thingsPadLayouts.delete(pluginId);
    this.activePlugins.delete(pluginId);
  }

  dispatchRoute(hash) {
    const cleanHash = (hash.startsWith('#') ? hash.slice(1) : hash).split('?')[0];
    if (this.routes.has(cleanHash)) {
      const { handler } = this.routes.get(cleanHash);
      handler();
      return true;
    }
    return false;
  }
}

export const pluginHost = new PluginHost();
