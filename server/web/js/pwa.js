/**
 * KB-CLI Web Fuzzy Finder — PWA & Offline Support
 * Handles service worker lifecycle, offline indicators, background sync cues, and shortcut actions.
 */

import { dom } from './dom.js';
import { showToast } from './utils.js';
import { openNoteModal } from './modal.js';

let onlineSyncCallback = null;

export function initPWA({ onOnlineSync } = {}) {
  onlineSyncCallback = onOnlineSync;

  // 1. Service Worker Registration
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      navigator.serviceWorker
        .register('/sw.js', { scope: '/' })
        .then((registration) => {
          registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing;
            if (newWorker) {
              newWorker.addEventListener('statechange', () => {
                if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                  showToast('Update available! Reload to update.', 'Reload', () => {
                    newWorker.postMessage({ type: 'SKIP_WAITING' });
                  });
                }
              });
            }
          });
        })
        .catch((err) => {
          console.warn('Service worker registration failed:', err);
        });

      let refreshing = false;
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        if (!refreshing) {
          refreshing = true;
          window.location.reload();
        }
      });
    });
  }

  // 2. Install Status
  window.addEventListener('appinstalled', () => {
    showToast('KB App installed successfully! 🎉');
  });

  // 3. Online/Offline Network Monitoring
  const updateOnlineStatus = () => {
    const isOnline = navigator.onLine;
    if (dom.offlineBadge) {
      dom.offlineBadge.style.display = isOnline ? 'none' : 'inline-flex';
    }
  };

  window.addEventListener('online', () => {
    updateOnlineStatus();
    showToast('Connected — syncing latest data');
    if (typeof onlineSyncCallback === 'function') {
      onlineSyncCallback();
    }
  });

  window.addEventListener('offline', () => {
    updateOnlineStatus();
    showToast('Working offline — cached notes available');
  });

  updateOnlineStatus();
}

/**
 * Handles PWA manifest shortcuts / quick actions (e.g. ?action=new, ?action=search).
 */
export function handlePWAUrlAction() {
  try {
    const params = new URLSearchParams(window.location.search);
    const action = params.get('action');
    if (action === 'new') {
      setTimeout(() => openNoteModal(), 200);
    } else if (action === 'search') {
      if (dom.searchInput) dom.searchInput.focus();
    }
  } catch (_) {}
}
