/**
 * KB-CLI Web Fuzzy Finder — Auth Controller
 * Handles access password prompt, visibility toggle, authentication errors, and logout.
 */

import { state } from './state.js';
import { dom } from './dom.js';
import { showToast } from './utils.js';
import { login, logout, setOnUnauthorized } from './api.js';

let authSuccessCallback = null;
let logoutCallback = null;

export function showAuthModal() {
  if (dom.authModal) {
    dom.authModal.style.display = 'flex';
    if (dom.authError) dom.authError.style.display = 'none';
    if (dom.authPassword) {
      dom.authPassword.value = '';
      setTimeout(() => dom.authPassword.focus(), 50);
    }
  }
}

export function hideAuthModal() {
  if (dom.authModal) {
    dom.authModal.style.display = 'none';
  }
}

/**
 * Initializes authentication listeners.
 */
export function setupAuthListeners({ onAuthSuccess, onLogout } = {}) {
  authSuccessCallback = onAuthSuccess;
  logoutCallback = onLogout;

  // Bind 401 response trigger to open auth modal
  setOnUnauthorized(showAuthModal);

  // Toggle password visibility
  if (dom.togglePasswordBtn && dom.authPassword) {
    dom.togglePasswordBtn.addEventListener('click', () => {
      const isPassword = dom.authPassword.type === 'password';
      dom.authPassword.type = isPassword ? 'text' : 'password';
      dom.togglePasswordBtn.textContent = isPassword ? '🔒' : '👁';
    });
  }

  // Auth form submission
  if (dom.authForm) {
    dom.authForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const password = dom.authPassword ? dom.authPassword.value : '';
      if (!password) return;

      if (dom.authSubmitBtn) {
        dom.authSubmitBtn.disabled = true;
        dom.authSubmitBtn.textContent = 'Unlocking...';
      }
      if (dom.authError) dom.authError.style.display = 'none';

      try {
        const { ok, data } = await login(password);

        if (ok && data.authenticated) {
          state.isAuthenticated = true;
          state.isAuthRequired = data.auth_required !== false;
          hideAuthModal();
          if (dom.logoutBtn) {
            dom.logoutBtn.style.display = state.isAuthRequired ? 'inline-flex' : 'none';
          }
          showToast('Knowledge Base unlocked');
          if (typeof authSuccessCallback === 'function') {
            await authSuccessCallback();
          }
          if (dom.searchInput) dom.searchInput.focus();
        } else {
          if (dom.authError) {
            dom.authError.textContent = data.error || 'Incorrect password. Please try again.';
            dom.authError.style.display = 'block';
          }
          if (dom.authPassword) {
            dom.authPassword.select();
            dom.authPassword.focus();
          }
        }
      } catch (err) {
        if (dom.authError) {
          dom.authError.textContent = `Connection error: ${err.message}`;
          dom.authError.style.display = 'block';
        }
      } finally {
        if (dom.authSubmitBtn) {
          dom.authSubmitBtn.disabled = false;
          dom.authSubmitBtn.textContent = 'Unlock';
        }
      }
    });
  }

  // Logout button
  if (dom.logoutBtn) {
    dom.logoutBtn.addEventListener('click', async () => {
      await logout();
      dom.logoutBtn.style.display = 'none';
      if (typeof logoutCallback === 'function') {
        logoutCallback();
      }
      showAuthModal();
      showToast('Signed out');
    });
  }
}
