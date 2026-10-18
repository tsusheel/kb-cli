/**
 * KB-CLI Web Fuzzy Finder — Application Entrypoint
 * Orchestrates modules, initializes UI subsystems, and starts lifecycle.
 */

import { dom } from './dom.js';
import { state } from './state.js';
import { checkAuthStatus } from './api.js';
import { showAuthModal, setupAuthListeners } from './auth.js';
import { initMarkdownLibraries } from './markdown.js';
import { initPWA, handlePWAUrlAction } from './pwa.js';
import { setupModalListeners } from './modal.js';
import {
  setupFinderListeners,
  loadData,
  loadTags,
  loadStats,
  renderEmptyPreview,
} from './finder.js';

async function init() {
  // 1. External UI libraries
  initMarkdownLibraries();

  // 2. Service Worker & Offline PWA sync
  initPWA({
    onOnlineSync: async () => {
      await loadData();
      await loadTags();
      await loadStats();
    },
  });

  // 3. UI Event Listeners
  setupFinderListeners();

  setupModalListeners({
    onNoteSaved: async () => {
      await loadData();
      await loadTags();
    },
  });

  setupAuthListeners({
    onAuthSuccess: async () => {
      await loadData();
      await loadTags();
      await loadStats();
    },
    onLogout: () => {
      state.allItems = [];
      state.filteredItems = [];
      if (dom.resultsList) {
        dom.resultsList.innerHTML =
          '<div class="empty-list">Locked. Please authenticate to view notes.</div>';
      }
      renderEmptyPreview();
    },
  });

  // 4. Initial Authentication & Data Hydration
  const auth = await checkAuthStatus();
  if (auth.isAuthRequired && !auth.isAuthenticated) {
    showAuthModal();
  } else {
    await loadData();
    await loadTags();
    await loadStats();
    handlePWAUrlAction();
  }
}

// Start application
if (document.readyState === 'loading') {
  window.addEventListener('DOMContentLoaded', init);
} else {
  init();
}
