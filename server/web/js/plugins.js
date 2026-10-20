/**
 * KB-CLI Web Plugin Manager
 * Loads active plugins, registers client-side routes, and integrates with the SPA lifecycle.
 */

import { pluginHost } from './plugin-host.js';

export const pluginsManager = {
  activePlugins: [],

  async init() {
    try {
      const res = await fetch('/api/plugins');
      if (!res.ok) return;
      const records = await res.json();
      if (!Array.isArray(records)) return;

      this.activePlugins = records.filter(p => p.enabled);

      // Listen for client-side route changes
      window.addEventListener('hashchange', () => {
        this.handleRouting();
      });

      this.handleRouting();
    } catch (err) {
      console.warn('Plugins discovery unavailable:', err);
    }
  },

  handleRouting() {
    const hash = window.location.hash || '';
    if (hash.startsWith('#/')) {
      pluginHost.dispatchRoute(hash);
    }
  }
};
