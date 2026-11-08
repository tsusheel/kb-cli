/**
 * KB-CLI Web Fuzzy Finder — Search Tokens & Interactive Pills Engine
 * Manages active filter pills (#tags, @types, is:status, +area) inside the search bar,
 * handling token conversion, backspace deletion, and pill removal.
 */

import { dom } from './dom.js';

class SearchTokensManager {
  constructor() {
    this.tokens = []; // Array of { type: string, raw: string, label: string }
    this.listeners = [];
  }

  init() {
    const wrapper = document.getElementById('search-input-wrapper');
    if (wrapper && dom.searchInput) {
      wrapper.addEventListener('click', (e) => {
        if (!e.target.closest('.token-remove-btn')) {
          dom.searchInput.focus();
        }
      });
    }

    this.bindInputEvents(dom.searchInput);
    this.bindInputEvents(dom.mobileSearchInput);
  }

  bindInputEvents(input) {
    if (!input) return;

    input.addEventListener('keydown', (e) => {
      // 1. Enter key blurs input (unless autocomplete handled it)
      if (e.key === 'Enter') {
        const autocompleteEl = document.getElementById('search-autocomplete');
        const isAutocompleteOpen = autocompleteEl && autocompleteEl.style.display !== 'none';
        if (!isAutocompleteOpen) {
          e.preventDefault();
          input.blur();
          return;
        }
      }

      // 2. Backspace in empty input deletes the last pill
      if (e.key === 'Backspace' && input.selectionStart === 0 && input.selectionEnd === 0) {
        if (this.tokens.length > 0) {
          e.preventDefault();
          this.removeTokenAt(this.tokens.length - 1);
        }
        return;
      }

      // 3. Space key converts the typed word into a pill if it matches a qualifier
      if (e.key === ' ') {
        const val = input.value;
        const pos = input.selectionStart;
        const left = val.slice(0, pos);
        const match = left.match(/(?:^|\s)([@#:+!a-zA-Z0-9_-]+)$/);
        if (match) {
          const word = match[1];
          if (this.isTokenCandidate(word)) {
            e.preventDefault();
            this.addToken(word);
            // Remove word from input
            const before = left.slice(0, pos - word.length).trim();
            const after = val.slice(pos).trim();
            input.value = before + (before && after ? ' ' : '') + after;
            this.notifyChange();
          }
        }
      }
    });

    // 4. Handle paste containing structured tokens
    input.addEventListener('paste', (e) => {
      const pasteText = (e.clipboardData || window.clipboardData)?.getData('text');
      if (pasteText && (pasteText.includes('#') || pasteText.includes('@') || pasteText.includes('is:') || pasteText.includes('+'))) {
        const words = pasteText.split(/\s+/);
        const nonTokens = [];
        let addedAny = false;

        for (const word of words) {
          if (this.isTokenCandidate(word)) {
            this.addToken(word, false);
            addedAny = true;
          } else {
            nonTokens.push(word);
          }
        }

        if (addedAny) {
          e.preventDefault();
          this.renderPills();
          input.value = (input.value + ' ' + nonTokens.join(' ')).trim();
          this.notifyChange();
        }
      }
    });
  }

  isTokenCandidate(word) {
    if (!word || word.length < 2) return false;
    // #tag, -#tag
    if (word.startsWith('#') || word.startsWith('-#')) return true;
    // @type, -@type
    if (word.startsWith('@') || word.startsWith('-@')) return true;
    // +area
    if (word.startsWith('+')) return true;
    // is:status, status:status
    if (word.toLowerCase().startsWith('is:') || word.toLowerCase().startsWith('status:')) return true;
    // type:type, area:area
    if (word.toLowerCase().startsWith('type:') || word.toLowerCase().startsWith('area:')) return true;

    return false;
  }

  classifyToken(raw) {
    const lower = raw.toLowerCase();
    if (lower.startsWith('-#') || lower.startsWith('-@')) {
      return { type: 'negated', raw, label: raw };
    }
    if (lower.startsWith('#')) {
      return { type: 'tag', raw, label: raw };
    }
    if (lower.startsWith('@') || lower.startsWith('type:')) {
      return { type: 'type', raw, label: raw.startsWith('type:') ? `@${raw.slice(5)}` : raw };
    }
    if (lower.startsWith('is:') || lower.startsWith('status:')) {
      return { type: 'status', raw, label: raw };
    }
    if (lower.startsWith('+') || lower.startsWith('area:')) {
      return { type: 'area', raw, label: raw.startsWith('area:') ? `+${raw.slice(5)}` : raw };
    }
    return { type: 'custom', raw, label: raw };
  }

  addToken(rawToken, renderNow = true) {
    const clean = rawToken.trim();
    if (!clean || !this.isTokenCandidate(clean)) return;

    // Check duplicate
    if (this.tokens.some((t) => t.raw.toLowerCase() === clean.toLowerCase())) {
      return;
    }

    const tokenObj = this.classifyToken(clean);
    this.tokens.push(tokenObj);

    if (renderNow) {
      this.renderPills();
      this.notifyChange();
    }
  }

  toggleTag(tag) {
    const raw = `#${tag}`;
    const idx = this.tokens.findIndex((t) => t.raw.toLowerCase() === raw.toLowerCase());
    if (idx !== -1) {
      this.removeTokenAt(idx);
    } else {
      this.addToken(raw);
    }
  }

  removeToken(rawToken) {
    const idx = this.tokens.findIndex((t) => t.raw.toLowerCase() === rawToken.toLowerCase());
    if (idx !== -1) {
      this.removeTokenAt(idx);
    }
  }

  removeTokenAt(index) {
    if (index >= 0 && index < this.tokens.length) {
      this.tokens.splice(index, 1);
      this.renderPills();
      this.notifyChange();
    }
  }

  clearTokens(renderNow = true) {
    this.tokens = [];
    if (renderNow) {
      this.renderPills();
      this.notifyChange();
    }
  }

  getTokens() {
    return this.tokens;
  }

  /**
   * Combines active pill tokens with any active input text into a unified search string.
   */
  getFullQueryString() {
    const tokenParts = this.tokens.map((t) => t.raw);
    const textPart = dom.searchInput ? dom.searchInput.value.trim() : '';
    return [...tokenParts, textPart].filter(Boolean).join(' ');
  }

  renderPills() {
    const containers = [
      document.getElementById('search-tokens-bar'),
      document.getElementById('mobile-search-tokens-bar'),
    ].filter(Boolean);

    containers.forEach((container) => {
      if (this.tokens.length === 0) {
        container.innerHTML = '';
        container.style.display = 'none';
        return;
      }

      container.style.display = 'inline-flex';
      container.innerHTML = this.tokens
        .map(
          (t, idx) => `
          <span class="search-token-pill token-${t.type}" data-index="${idx}">
            <span class="token-text">${t.label}</span>
            <button type="button" class="token-remove-btn" data-index="${idx}" title="Remove filter">✕</button>
          </span>
        `
        )
        .join('');

      container.querySelectorAll('.token-remove-btn').forEach((btn) => {
        btn.addEventListener('click', (e) => {
          e.stopPropagation();
          const idx = parseInt(btn.dataset.index, 10);
          this.removeTokenAt(idx);
          if (dom.searchInput) dom.searchInput.focus();
        });
      });
    });

    // Update clear button visibility
    if (dom.clearSearchBtn) {
      const hasContent = this.tokens.length > 0 || (dom.searchInput && dom.searchInput.value.trim().length > 0);
      dom.clearSearchBtn.style.display = hasContent ? 'block' : 'none';
    }
  }

  notifyChange() {
    if (dom.searchInput) {
      dom.searchInput.dispatchEvent(new Event('input', { bubbles: true }));
    }
  }
}

export const searchTokens = new SearchTokensManager();

export function initSearchTokens() {
  searchTokens.init();
}

