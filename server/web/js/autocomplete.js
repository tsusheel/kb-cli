/**
 * KB-CLI Web Fuzzy Finder — Caret-Anchored Autocomplete Component
 * Manages live suggestion dropdowns for tags (#), types (@), statuses (is:), areas (+),
 * and plugin triggers, with full keyboard navigation and cursor token insertion.
 */

import { dom } from './dom.js';
import { searchRegistry } from './search-registry.js';
import { searchTokens } from './search-tokens.js';
import { escapeHtml } from './utils.js';

class AutocompleteController {
  constructor() {
    this.container = null;
    this.activeInput = null;
    this.suggestions = [];
    this.selectedIndex = 0;
    this.activeWordInfo = null;
    this.isOpen = false;
  }

  init() {
    this.container = document.getElementById('search-autocomplete');
    if (!this.container) return;

    this.bindInput(dom.searchInput);
    this.bindInput(dom.mobileSearchInput);

    // Close when clicking outside
    document.addEventListener('click', (e) => {
      if (this.isOpen && !this.container.contains(e.target) && e.target !== this.activeInput) {
        this.close();
      }
    });
  }

  bindInput(input) {
    if (!input) return;

    input.addEventListener('input', () => {
      this.handleInputChange(input);
    });

    input.addEventListener('focus', () => {
      this.handleInputFocus(input);
    });

    input.addEventListener('keydown', (e) => {
      this.handleKeyDown(e, input);
    });

    input.addEventListener('click', () => {
      this.handleInputChange(input);
    });
  }

  getActiveWordInfo(input) {
    const pos = input.selectionStart;
    const text = input.value;
    const leftText = text.slice(0, pos);

    // Match the active word immediately preceding the cursor
    const match = leftText.match(/(?:^|\s)([@#:+!a-zA-Z0-9_-]+)$/);
    if (!match) return null;

    const word = match[1];
    const startIndex = pos - word.length;
    const endIndex = pos;

    return { word, startIndex, endIndex };
  }

  async handleInputChange(input) {
    this.activeInput = input;
    const wordInfo = this.getActiveWordInfo(input);

    if (!wordInfo || !wordInfo.word) {
      if (input.value.trim() === '') {
        // Show quick picks on empty search bar
        this.showQuickPicks(input);
      } else {
        this.close();
      }
      return;
    }

    this.activeWordInfo = wordInfo;
    const suggestions = await searchRegistry.getSuggestionsForWord(wordInfo.word);

    if (suggestions && suggestions.length > 0) {
      this.open(suggestions, input);
    } else {
      this.close();
    }
  }

  handleInputFocus(input) {
    this.activeInput = input;
    if (input.value.trim() === '') {
      this.showQuickPicks(input);
    } else {
      this.handleInputChange(input);
    }
  }

  showQuickPicks(input) {
    this.activeInput = input;
    this.activeWordInfo = { word: '', startIndex: 0, endIndex: 0 };
    const quickPicks = searchRegistry.getQuickPicks();
    this.open(quickPicks, input, 'Quick Filters');
  }

  open(suggestions, input, headerTitle = null) {
    this.suggestions = suggestions;
    this.selectedIndex = 0;
    this.isOpen = true;

    if (input === dom.mobileSearchInput && dom.mobileBottomSearch) {
      if (this.container.parentElement !== dom.mobileBottomSearch) {
        dom.mobileBottomSearch.appendChild(this.container);
      }
      this.container.classList.add('mobile-autocomplete');
    } else {
      const searchContainer = document.querySelector('.search-container');
      if (searchContainer && this.container.parentElement !== searchContainer) {
        searchContainer.appendChild(this.container);
      }
      this.container.classList.remove('mobile-autocomplete');
    }

    this.render(headerTitle);
    this.container.style.display = 'block';
  }

  close() {
    this.isOpen = false;
    this.suggestions = [];
    this.activeWordInfo = null;
    if (this.container) {
      this.container.style.display = 'none';
      this.container.innerHTML = '';
    }
  }

  render(headerTitle) {
    if (!this.container || this.suggestions.length === 0) return;

    let title = headerTitle;
    if (!title) {
      const first = this.suggestions[0];
      title = first.category || 'Suggestions';
    }

    let html = `
      <div class="autocomplete-header">
        <span>${escapeHtml(title)}</span>
        <span class="autocomplete-hint"><kbd>Tab</kbd> to insert</span>
      </div>
      <div class="autocomplete-list">
    `;

    this.suggestions.forEach((item, index) => {
      const isSelected = index === this.selectedIndex;
      const badgeHtml = item.badge
        ? `<span class="autocomplete-badge ${escapeHtml(item.badge)}">${escapeHtml(item.label.replace(/^[@#]/, ''))}</span>`
        : '';
      const countHtml = item.count ? `<span class="autocomplete-count">${escapeHtml(item.count)}</span>` : '';
      const descHtml = item.description ? `<span class="autocomplete-desc">${escapeHtml(item.description)}</span>` : '';

      html += `
        <div class="autocomplete-item ${isSelected ? 'selected' : ''}" data-index="${index}">
          <div class="autocomplete-item-main">
            <span class="autocomplete-label">${escapeHtml(item.label)}</span>
            ${badgeHtml}
            ${descHtml}
          </div>
          ${countHtml}
        </div>
      `;
    });

    html += `
      </div>
      <div class="autocomplete-footer">
        <span><kbd>↑</kbd><kbd>↓</kbd> navigate</span>
        <span><kbd>↵</kbd> select</span>
        <span><kbd>Esc</kbd> close</span>
      </div>
    `;

    this.container.innerHTML = html;

    // Attach click listeners to suggestions
    this.container.querySelectorAll('.autocomplete-item').forEach((el) => {
      el.addEventListener('mousedown', (e) => {
        e.preventDefault(); // prevent input blur
        const index = parseInt(el.dataset.index, 10);
        this.selectSuggestion(index);
      });
    });
  }

  handleKeyDown(e, input) {
    if (!this.isOpen || this.suggestions.length === 0) return;

    if (e.key === 'ArrowDown' || (e.ctrlKey && e.key === 'n')) {
      e.preventDefault();
      e.stopPropagation();
      this.selectedIndex = (this.selectedIndex + 1) % this.suggestions.length;
      this.updateSelection();
      return;
    }

    if (e.key === 'ArrowUp' || (e.ctrlKey && e.key === 'p')) {
      e.preventDefault();
      e.stopPropagation();
      this.selectedIndex = (this.selectedIndex - 1 + this.suggestions.length) % this.suggestions.length;
      this.updateSelection();
      return;
    }

    if (e.key === 'Tab' || e.key === 'Enter') {
      e.preventDefault();
      e.stopPropagation();
      this.selectSuggestion(this.selectedIndex);
      return;
    }

    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      this.close();
      return;
    }
  }

  updateSelection() {
    const items = this.container.querySelectorAll('.autocomplete-item');
    items.forEach((el, idx) => {
      el.classList.toggle('selected', idx === this.selectedIndex);
      if (idx === this.selectedIndex) {
        el.scrollIntoView({ block: 'nearest' });
      }
    });
  }

  selectSuggestion(index) {
    const item = this.suggestions[index];
    if (!item || !this.activeInput) return;

    if (searchTokens.isTokenCandidate(item.text)) {
      searchTokens.addToken(item.text);

      const text = this.activeInput.value;
      if (this.activeWordInfo && this.activeWordInfo.startIndex >= 0) {
        const before = text.slice(0, this.activeWordInfo.startIndex).trim();
        const after = text.slice(this.activeWordInfo.endIndex).trim();
        this.activeInput.value = before + (before && after ? ' ' : '') + after;
      } else {
        this.activeInput.value = '';
      }

      this.activeInput.focus();
      this.close();
      this.activeInput.dispatchEvent(new Event('input', { bubbles: true }));
      return;
    }

    const text = this.activeInput.value;
    const replacement = item.text + ' ';

    let newText = '';
    let newCursorPos = 0;

    if (this.activeWordInfo && this.activeWordInfo.startIndex >= 0) {
      const before = text.slice(0, this.activeWordInfo.startIndex);
      const after = text.slice(this.activeWordInfo.endIndex);
      newText = before + replacement + after;
      newCursorPos = (before + replacement).length;
    } else {
      newText = replacement;
      newCursorPos = replacement.length;
    }

    this.activeInput.value = newText;
    this.activeInput.focus();
    if (typeof this.activeInput.setSelectionRange === 'function') {
      this.activeInput.setSelectionRange(newCursorPos, newCursorPos);
    }

    this.close();

    // Trigger input event to update results
    this.activeInput.dispatchEvent(new Event('input', { bubbles: true }));
  }
}

export const autocomplete = new AutocompleteController();

export function initAutocomplete() {
  autocomplete.init();
}
