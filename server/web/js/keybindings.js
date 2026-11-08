/**
 * KB-CLI Web Fuzzy Finder — Scalable Keybinding & Command Engine
 * Provides a modular, context-aware shortcut registry extensible across core and plugins,
 * supporting multi-key chords, 2-stage Escape clearing/blurring, and a dynamic Help modal.
 */

import { dom } from './dom.js';
import { state } from './state.js';
import { showToast } from './utils.js';
import { closeNoteModal, openNoteModal } from './modal.js';
import {
  closeDocModal,
  closeFolderModal,
  openNewDocModal,
  openEditDocModal,
  navigateDocs,
  handleDocsSearch,
} from './docs.js';
import {
  selectItem,
  toggleViewMode,
  toggleCurrentItemStatus,
  deleteCurrentItem,
  filterAndRender,
  showMobileList,
} from './finder.js';
import { triggerSync } from './api.js';

class KeybindingRegistry {
  constructor() {
    this.bindings = new Map();
    this.chordBuffer = [];
    this.lastChordTime = 0;
    this.chordTimeoutMs = 450;
    this.boundKeyHandler = this.handleKeyDown.bind(this);
    this.initialized = false;
  }

  /**
   * Registers a single keybinding or array of keys for an action.
   * @param {Object} def
   * @param {string} def.id Unique identifier (e.g. 'core.search.focus')
   * @param {string|string[]} def.keys e.g. '/', ['ctrl+k', 'meta+k'], 'g g', 'ctrl+enter'
   * @param {string} def.description Human-readable label for UI / cheatsheet
   * @param {string} [def.category='General'] Section name for the help cheatsheet
   * @param {string|function} [def.context='global'] 'global', 'list', 'input', 'modal', 'notes', 'docs', or custom predicate
   * @param {string} [def.pluginId] Optional identifier of the plugin owning this binding
   * @param {number} [def.priority=0] Higher priorities evaluate first
   * @param {function} def.handler Callback (event, context) => void
   */
  register(def) {
    if (!def.id || !def.keys || !def.handler) {
      console.warn('Invalid keybinding registration:', def);
      return;
    }
    const keys = Array.isArray(def.keys) ? def.keys : [def.keys];
    const normalizedKeys = keys.map((k) => k.toLowerCase().trim());

    this.bindings.set(def.id, {
      ...def,
      keys: normalizedKeys,
      category: def.category || 'General',
      context: def.context || 'global',
      priority: def.priority || 0,
    });
  }

  unregister(id) {
    this.bindings.delete(id);
  }

  unregisterPlugin(pluginId) {
    for (const [id, def] of this.bindings.entries()) {
      if (def.pluginId === pluginId) {
        this.bindings.delete(id);
      }
    }
  }

  /**
   * Groups all registered bindings by category for display in the Help modal.
   */
  getGroupedBindings() {
    const groups = {};
    for (const def of this.bindings.values()) {
      const cat = def.category || 'General';
      if (!groups[cat]) groups[cat] = [];
      groups[cat].push(def);
    }
    return groups;
  }

  /**
   * Returns whether any application modal overlay is currently visible.
   */
  isModalOpen() {
    return Boolean(
      (dom.shortcutsModal && dom.shortcutsModal.style.display !== 'none') ||
      (dom.noteModal && dom.noteModal.style.display !== 'none') ||
      (dom.docModal && dom.docModal.style.display !== 'none') ||
      (dom.folderModal && dom.folderModal.style.display !== 'none') ||
      (dom.authModal && dom.authModal.style.display !== 'none')
    );
  }

  getActiveModal() {
    if (dom.shortcutsModal && dom.shortcutsModal.style.display !== 'none') return dom.shortcutsModal;
    if (dom.noteModal && dom.noteModal.style.display !== 'none') return dom.noteModal;
    if (dom.docModal && dom.docModal.style.display !== 'none') return dom.docModal;
    if (dom.folderModal && dom.folderModal.style.display !== 'none') return dom.folderModal;
    if (dom.authModal && dom.authModal.style.display !== 'none') return dom.authModal;
    return null;
  }

  closeActiveModal() {
    if (dom.shortcutsModal && dom.shortcutsModal.style.display !== 'none') {
      this.closeShortcutsModal();
      return true;
    }
    if (dom.noteModal && dom.noteModal.style.display !== 'none') {
      closeNoteModal();
      return true;
    }
    if (dom.docModal && dom.docModal.style.display !== 'none') {
      closeDocModal();
      return true;
    }
    if (dom.folderModal && dom.folderModal.style.display !== 'none') {
      closeFolderModal();
      return true;
    }
    return false;
  }

  /**
   * Renders and opens the keyboard shortcuts help modal.
   */
  openShortcutsModal() {
    if (!dom.shortcutsModal) return;
    this.renderShortcutsModalContent();
    dom.shortcutsModal.style.display = 'flex';
  }

  closeShortcutsModal() {
    if (dom.shortcutsModal) {
      dom.shortcutsModal.style.display = 'none';
    }
  }

  toggleShortcutsModal() {
    if (dom.shortcutsModal && dom.shortcutsModal.style.display !== 'none') {
      this.closeShortcutsModal();
    } else {
      this.openShortcutsModal();
    }
  }

  /**
   * Populates the shortcuts modal with dynamically formatted key combos.
   */
  renderShortcutsModalContent() {
    const container = document.getElementById('shortcuts-modal-body');
    if (!container) return;

    const groups = this.getGroupedBindings();
    const categoryOrder = [
      'Navigation',
      'Search & Focus',
      'Preview Pane',
      'Actions & Management',
      'Views & Filtering',
      'Modal Dialogs & Forms',
      'General',
    ];

    // Ensure all registered categories (including plugin categories) are shown
    const allCategories = [
      ...categoryOrder.filter((cat) => groups[cat]),
      ...Object.keys(groups).filter((cat) => !categoryOrder.includes(cat)),
    ];

    let html = '<div class="shortcuts-grid">';
    for (const cat of allCategories) {
      const items = groups[cat];
      if (!items || items.length === 0) continue;

      html += `
        <div class="shortcuts-section">
          <div class="shortcuts-section-title">
            <span>${cat}</span>
          </div>
          <div class="shortcuts-section-list">
      `;

      for (const item of items) {
        const renderedKeys = this.formatKeysForDisplay(item.keys[0]);
        html += `
          <div class="shortcut-row">
            <span class="shortcut-desc">${item.description}</span>
            <div class="shortcut-keys">${renderedKeys}</div>
          </div>
        `;
      }

      html += `
          </div>
        </div>
      `;
    }
    html += '</div>';

    container.innerHTML = html;
  }

  formatKeysForDisplay(keyStr) {
    if (!keyStr) return '';
    // Multi-key chord (e.g. 'g g', 'g n')
    if (keyStr.includes(' ')) {
      return keyStr
        .split(' ')
        .map((k) => `<kbd>${this.capitalizeKey(k)}</kbd>`)
        .join('<span class="key-then">then</span>');
    }

    // Modifier combo (e.g. 'ctrl+enter', 'ctrl+k')
    if (keyStr.includes('+')) {
      return keyStr
        .split('+')
        .map((k) => `<kbd>${this.capitalizeKey(k)}</kbd>`)
        .join('+');
    }

    return `<kbd>${this.capitalizeKey(keyStr)}</kbd>`;
  }

  capitalizeKey(k) {
    const map = {
      ctrl: 'Ctrl',
      meta: 'Cmd',
      cmd: 'Cmd',
      alt: 'Alt',
      shift: 'Shift',
      enter: '↵ Enter',
      escape: 'Esc',
      arrowup: '↑',
      arrowdown: '↓',
      arrowleft: '←',
      arrowright: '→',
      delete: 'Del',
      tab: 'Tab',
      space: 'Space',
    };
    return map[k.toLowerCase()] || k.toUpperCase();
  }

  /**
   * Normalizes an incoming KeyboardEvent to a standard string representation.
   */
  normalizeEvent(e) {
    const parts = [];
    if (e.ctrlKey || e.metaKey) parts.push('ctrl');
    if (e.altKey) parts.push('alt');
    if (e.shiftKey && (parts.length > 0 || e.key.length > 1)) parts.push('shift');

    let key = e.key.toLowerCase();
    if (key === 'control' || key === 'meta' || key === 'alt' || key === 'shift') {
      return '';
    }

    if (parts.length > 0) {
      parts.push(key);
      return parts.join('+');
    }

    return key;
  }

  /**
   * Main keyboard event dispatcher.
   */
  handleKeyDown(e) {
    const activeEl = document.activeElement;
    const isInputFocused =
      activeEl === dom.searchInput ||
      activeEl?.tagName === 'INPUT' ||
      activeEl?.tagName === 'TEXTAREA' ||
      activeEl?.tagName === 'SELECT';

    const isModalOpen = this.isModalOpen();
    const activeModal = this.getActiveModal();

    // ──────────────────────────────────────────────────────────────────────────
    // SPECIAL REQUIREMENT: Two-Stage 'Escape' Key Machine
    // ──────────────────────────────────────────────────────────────────────────
    if (e.key === 'Escape') {
      // 1. If help shortcuts overlay is open, dismiss it
      if (dom.shortcutsModal && dom.shortcutsModal.style.display !== 'none') {
        this.closeShortcutsModal();
        e.preventDefault();
        return;
      }

      // 2. If any other modal is active
      if (isModalOpen && activeModal) {
        if (isInputFocused && activeEl && activeEl.value) {
          // If editing a modal input with text, 1st Escape clears text
          activeEl.value = '';
          activeEl.dispatchEvent(new Event('input', { bubbles: true }));
          e.preventDefault();
          return;
        }
        if (isInputFocused && activeEl) {
          // 2nd Escape blurs the input
          activeEl.blur();
          e.preventDefault();
          return;
        }
        // No input focused: close the modal
        this.closeActiveModal();
        e.preventDefault();
        return;
      }

      // 3. Mobile search dock / detail preview
      if (dom.appContainer && dom.appContainer.classList.contains('mobile-search-open')) {
        if (dom.mobileSearchInput) dom.mobileSearchInput.value = '';
        if (state.currentViewMode === 'docs') {
          handleDocsSearch('');
        } else {
          if (dom.searchInput) dom.searchInput.value = '';
          filterAndRender();
        }
        dom.appContainer.classList.remove('mobile-search-open');
        e.preventDefault();
        return;
      }

      if (dom.appContainer && dom.appContainer.classList.contains('mobile-show-preview')) {
        showMobileList();
        e.preventDefault();
        return;
      }

      // 4. Main Search Input or any focused input field
      if (isInputFocused && activeEl) {
        if (activeEl.value && activeEl.value.length > 0) {
          // Stage 1: Empty the input field first, update filters, keep focus
          activeEl.value = '';
          if (dom.clearSearchBtn) dom.clearSearchBtn.style.display = 'none';
          activeEl.dispatchEvent(new Event('input', { bubbles: true }));

          if (state.currentViewMode === 'docs') {
            handleDocsSearch('');
          } else {
            filterAndRender();
          }
          e.preventDefault();
          return;
        } else {
          // Stage 2: Input is already empty -> go out of focus (blur)
          activeEl.blur();
          e.preventDefault();
          return;
        }
      }

      // 5. If not input-focused but search input has a residual query, clear it
      if (dom.searchInput && dom.searchInput.value) {
        dom.searchInput.value = '';
        if (dom.clearSearchBtn) dom.clearSearchBtn.style.display = 'none';
        if (state.currentViewMode === 'docs') {
          handleDocsSearch('');
        } else {
          filterAndRender();
        }
        e.preventDefault();
        return;
      }
      return;
    }

    // ──────────────────────────────────────────────────────────────────────────
    // SPECIAL REQUIREMENT: '?' Shortcut Help Modal Overlay
    // ──────────────────────────────────────────────────────────────────────────
    if (!isInputFocused && (e.key === '?' || (e.shiftKey && e.key === '/'))) {
      e.preventDefault();
      this.toggleShortcutsModal();
      return;
    }

    // If help modal is open, ignore other keys except Escape
    if (dom.shortcutsModal && dom.shortcutsModal.style.display !== 'none') {
      return;
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Multi-key Chord Tracking ('g g', 'g n', 'g d')
    // ──────────────────────────────────────────────────────────────────────────
    const now = Date.now();
    if (now - this.lastChordTime > this.chordTimeoutMs) {
      this.chordBuffer = [];
    }

    const currentKey = this.normalizeEvent(e);
    if (!currentKey) return;

    if (!isInputFocused && !isModalOpen && currentKey.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
      this.chordBuffer.push(currentKey);
      this.lastChordTime = now;

      if (this.chordBuffer.length === 2) {
        const chord = this.chordBuffer.join(' ');
        this.chordBuffer = [];

        // Check if any registered binding matches this chord
        for (const def of this.bindings.values()) {
          if (def.keys.includes(chord)) {
            if (this.matchesContext(def.context, { isInputFocused, isModalOpen })) {
              e.preventDefault();
              def.handler(e, { registry: this, isInputFocused, isModalOpen });
              return;
            }
          }
        }
      }
    } else {
      this.chordBuffer = [];
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Match Standard Keybindings
    // ──────────────────────────────────────────────────────────────────────────
    const candidates = [];
    for (const def of this.bindings.values()) {
      const matchesKey = def.keys.some((k) => {
        if (k === currentKey) return true;
        // Map cmd+ or meta+ to ctrl+ for cross-platform ease
        if (currentKey.startsWith('ctrl+') && (k.startsWith('cmd+') || k.startsWith('meta+'))) {
          return k.slice(4) === currentKey.slice(5) || k.slice(5) === currentKey.slice(5);
        }
        return false;
      });

      if (matchesKey) {
        if (this.matchesContext(def.context, { isInputFocused, isModalOpen, event: e })) {
          candidates.push(def);
        }
      }
    }

    if (candidates.length > 0) {
      // Sort by priority descending
      candidates.sort((a, b) => (b.priority || 0) - (a.priority || 0));
      const target = candidates[0];
      e.preventDefault();
      target.handler(e, { registry: this, isInputFocused, isModalOpen });
    }
  }

  matchesContext(context, { isInputFocused, isModalOpen, event }) {
    if (typeof context === 'function') {
      return context({ isInputFocused, isModalOpen, event, state });
    }

    switch (context) {
      case 'global':
        // If an input is focused, only permit modified keys (e.g. Ctrl+K, Ctrl+Enter, Ctrl+N)
        if (isInputFocused) {
          return event && (event.ctrlKey || event.metaKey);
        }
        return !isModalOpen;
      case 'list':
        return !isInputFocused && !isModalOpen;
      case 'input':
        return isInputFocused;
      case 'modal':
        return isModalOpen;
      case 'notes':
        return !isInputFocused && !isModalOpen && state.currentViewMode !== 'docs';
      case 'docs':
        return !isInputFocused && !isModalOpen && state.currentViewMode === 'docs';
      default:
        return true;
    }
  }

  init() {
    if (this.initialized) return;
    this.initialized = true;

    // Attach global listener
    document.addEventListener('keydown', this.boundKeyHandler, { capture: false });

    // Setup modal backdrop and button close listeners
    if (dom.shortcutsModalCloseBtn) {
      dom.shortcutsModalCloseBtn.addEventListener('click', () => this.closeShortcutsModal());
    }
    const doneBtn = document.getElementById('shortcuts-modal-done-btn');
    if (doneBtn) {
      doneBtn.addEventListener('click', () => this.closeShortcutsModal());
    }
    if (dom.shortcutsModal) {
      dom.shortcutsModal.addEventListener('click', (e) => {
        if (e.target === dom.shortcutsModal) {
          this.closeShortcutsModal();
        }
      });
    }

    // Footer Help pill trigger
    const helpTrigger = document.getElementById('shortcuts-help-trigger');
    if (helpTrigger) {
      helpTrigger.addEventListener('click', () => this.openShortcutsModal());
    }

    // Register all default core keybindings
    registerCoreKeybindings(this);
  }
}

/**
 * Registers all core application shortcuts into the registry.
 */
function registerCoreKeybindings(registry) {
  // ──────────────────────────────────────────────────────────────────────────
  // Search & Focus
  // ──────────────────────────────────────────────────────────────────────────
  registry.register({
    id: 'core.search.focus',
    keys: ['/', 'ctrl+k'],
    description: 'Focus fuzzy search input',
    category: 'Search & Focus',
    context: (ctx) => !ctx.isModalOpen && (!ctx.isInputFocused || ctx.event?.ctrlKey || ctx.event?.metaKey),
    handler: () => {
      if (dom.searchInput) {
        dom.searchInput.focus();
        dom.searchInput.select();
      }
    },
  });

  // ──────────────────────────────────────────────────────────────────────────
  // Navigation & Lists (Notes & Docs)
  // ──────────────────────────────────────────────────────────────────────────
  registry.register({
    id: 'core.nav.next',
    keys: ['j', 'arrowdown', 'ctrl+n'],
    description: 'Next item / document',
    category: 'Navigation',
    context: (ctx) => !ctx.isModalOpen && (!ctx.isInputFocused || ctx.event?.ctrlKey),
    handler: () => {
      if (state.currentViewMode === 'docs') {
        navigateDocs('next');
      } else {
        if (state.filteredItems && state.filteredItems.length > 0) {
          selectItem(Math.min(state.selectedIndex + 1, state.filteredItems.length - 1));
        }
      }
    },
  });

  registry.register({
    id: 'core.nav.prev',
    keys: ['k', 'arrowup', 'ctrl+p'],
    description: 'Previous item / document',
    category: 'Navigation',
    context: (ctx) => !ctx.isModalOpen && (!ctx.isInputFocused || ctx.event?.ctrlKey),
    handler: () => {
      if (state.currentViewMode === 'docs') {
        navigateDocs('prev');
      } else {
        if (state.filteredItems && state.filteredItems.length > 0) {
          selectItem(Math.max(state.selectedIndex - 1, 0));
        }
      }
    },
  });

  registry.register({
    id: 'core.nav.top',
    keys: 'g g',
    description: 'Jump to top of list / tree',
    category: 'Navigation',
    context: 'list',
    handler: () => {
      if (state.currentViewMode === 'docs') {
        // In docs mode, navigate to first document
        const firstTreeDoc = dom.docsTree?.querySelector('.tree-file-item');
        if (firstTreeDoc) firstTreeDoc.click();
      } else {
        if (state.filteredItems && state.filteredItems.length > 0) {
          selectItem(0);
        }
      }
      showToast('Jumped to top');
    },
  });

  registry.register({
    id: 'core.nav.bottom',
    keys: ['G', 'shift+g'],
    description: 'Jump to bottom of list',
    category: 'Navigation',
    context: 'list',
    handler: () => {
      if (state.currentViewMode === 'docs') {
        const allDocs = dom.docsTree?.querySelectorAll('.tree-file-item');
        if (allDocs && allDocs.length > 0) {
          allDocs[allDocs.length - 1].click();
        }
      } else {
        if (state.filteredItems && state.filteredItems.length > 0) {
          selectItem(state.filteredItems.length - 1);
        }
      }
      showToast('Jumped to bottom');
    },
  });

  // ──────────────────────────────────────────────────────────────────────────
  // Preview Pane & Reading
  // ──────────────────────────────────────────────────────────────────────────
  registry.register({
    id: 'core.preview.scroll-down',
    keys: ['J', 'shift+j', 'ctrl+arrowdown'],
    description: 'Scroll preview pane down',
    category: 'Preview Pane',
    context: 'list',
    handler: () => {
      if (dom.previewPane) {
        dom.previewPane.scrollBy({ top: 180, behavior: 'smooth' });
      }
    },
  });

  registry.register({
    id: 'core.preview.scroll-up',
    keys: ['K', 'shift+k', 'ctrl+arrowup'],
    description: 'Scroll preview pane up',
    category: 'Preview Pane',
    context: 'list',
    handler: () => {
      if (dom.previewPane) {
        dom.previewPane.scrollBy({ top: -180, behavior: 'smooth' });
      }
    },
  });

  registry.register({
    id: 'core.preview.toggle-view',
    keys: 'v',
    description: 'Toggle Raw source / Rendered preview',
    category: 'Preview Pane',
    context: 'list',
    handler: () => {
      toggleViewMode();
    },
  });

  registry.register({
    id: 'core.preview.copy-id',
    keys: ['c', 'y'],
    description: 'Copy item ID or document path',
    category: 'Preview Pane',
    context: 'list',
    handler: () => {
      if (state.currentViewMode === 'docs') {
        if (state.selectedDocPath) {
          navigator.clipboard.writeText(state.selectedDocPath);
          showToast(`Copied Path: ${state.selectedDocPath}`);
        }
      } else {
        const item = state.filteredItems?.[state.selectedIndex];
        if (item) {
          navigator.clipboard.writeText(item.id);
          showToast(`Copied ID: ${item.short_id || item.id.substring(0, 7)}`);
        }
      }
    },
  });

  registry.register({
    id: 'core.preview.copy-markdown',
    keys: 'm',
    description: 'Copy markdown body to clipboard',
    category: 'Preview Pane',
    context: 'list',
    handler: () => {
      if (state.currentViewMode === 'docs') {
        if (state.currentDocContent) {
          navigator.clipboard.writeText(state.currentDocContent);
          showToast('Copied Document Markdown');
        }
      } else {
        const item = state.filteredItems?.[state.selectedIndex];
        if (item) {
          const md = `# ${item.display}\n\n${item.flesh || ''}`;
          navigator.clipboard.writeText(md);
          showToast('Copied Note Markdown');
        }
      }
    },
  });

  // ──────────────────────────────────────────────────────────────────────────
  // Actions & Management
  // ──────────────────────────────────────────────────────────────────────────
  registry.register({
    id: 'core.action.new',
    keys: 'n',
    description: 'Create new Note (or Doc in Docs mode)',
    category: 'Actions & Management',
    context: 'list',
    handler: () => {
      if (state.currentViewMode === 'docs') {
        openNewDocModal();
      } else {
        openNoteModal();
      }
    },
  });

  registry.register({
    id: 'core.action.edit',
    keys: ['e', 'enter'],
    description: 'Edit currently selected note or doc',
    category: 'Actions & Management',
    context: 'list',
    handler: () => {
      if (state.currentViewMode === 'docs') {
        if (state.selectedDocPath) {
          openEditDocModal(state.selectedDocPath);
        }
      } else {
        const item = state.filteredItems?.[state.selectedIndex];
        if (item && !item.is_log) {
          if (item.type === 'doc' || item.is_doc) {
            openEditDocModal(item.doc_path || item.id);
          } else {
            openNoteModal(item);
          }
        }
      }
    },
  });

  registry.register({
    id: 'core.action.toggle-status',
    keys: ['t', 'x'],
    description: 'Toggle Todo status (Active ↔ Completed)',
    category: 'Actions & Management',
    context: 'list',
    handler: () => {
      if (state.currentViewMode !== 'docs') {
        toggleCurrentItemStatus();
      }
    },
  });

  registry.register({
    id: 'core.action.delete',
    keys: ['d', 'delete'],
    description: 'Delete current note or document',
    category: 'Actions & Management',
    context: 'list',
    handler: () => {
      deleteCurrentItem();
    },
  });

  registry.register({
    id: 'core.action.sync',
    keys: 's',
    description: 'Trigger database replication sync',
    category: 'Actions & Management',
    context: 'list',
    handler: async () => {
      if (dom.syncBtn) {
        dom.syncBtn.classList.add('syncing');
        dom.syncBtn.textContent = 'Syncing...';
      }
      showToast('Syncing database...');
      try {
        const res = await triggerSync();
        showToast(res.message || 'Sync complete');
      } catch (err) {
        showToast(`Sync failed: ${err.message}`);
      } finally {
        if (dom.syncBtn) {
          dom.syncBtn.classList.remove('syncing');
          dom.syncBtn.textContent = 'Sync';
        }
      }
    },
  });

  // ──────────────────────────────────────────────────────────────────────────
  // Views & Filtering Chords
  // ──────────────────────────────────────────────────────────────────────────
  registry.register({
    id: 'core.view.notes',
    keys: 'g n',
    description: 'Switch to Notes & Tasks view',
    category: 'Views & Filtering',
    context: 'list',
    handler: () => {
      if (dom.tabNotes) dom.tabNotes.click();
    },
  });

  registry.register({
    id: 'core.view.docs',
    keys: 'g d',
    description: 'Switch to System Docs tree view',
    category: 'Views & Filtering',
    context: 'list',
    handler: () => {
      if (dom.tabDocs) dom.tabDocs.click();
    },
  });

  // Quick category filter pill numbers 1-7
  const filterPillKeys = ['1', '2', '3', '4', '5', '6', '7'];
  filterPillKeys.forEach((key, idx) => {
    registry.register({
      id: `core.filter.${key}`,
      keys: key,
      description: `Switch to filter tab #${idx + 1}`,
      category: 'Views & Filtering',
      context: 'list',
      handler: () => {
        const pills = document.querySelectorAll('#type-filter-pills .pill');
        if (pills && pills[idx]) {
          pills[idx].click();
        }
      },
    });
  });

  // ──────────────────────────────────────────────────────────────────────────
  // Modal Forms & Dialog Ergonomics
  // ──────────────────────────────────────────────────────────────────────────
  registry.register({
    id: 'core.modal.save',
    keys: 'ctrl+enter',
    description: 'Save & Submit active modal form',
    category: 'Modal Dialogs & Forms',
    context: 'modal',
    priority: 10,
    handler: () => {
      const active = registry.getActiveModal();
      if (!active) return;
      const form = active.querySelector('form');
      if (form) {
        if (typeof form.requestSubmit === 'function') {
          form.requestSubmit();
        } else {
          const submitBtn = form.querySelector('[type="submit"]');
          if (submitBtn) submitBtn.click();
        }
      }
    },
  });

  registry.register({
    id: 'core.modal.toggle-tab',
    keys: 'ctrl+p',
    description: 'Toggle Write / Preview editor tabs',
    category: 'Modal Dialogs & Forms',
    context: 'modal',
    priority: 10,
    handler: () => {
      if (dom.noteModal && dom.noteModal.style.display !== 'none') {
        if (dom.modalTabWrite?.classList.contains('active')) {
          dom.modalTabPreview?.click();
        } else {
          dom.modalTabWrite?.click();
        }
      } else if (dom.docModal && dom.docModal.style.display !== 'none') {
        if (dom.docModalTabWrite?.classList.contains('active')) {
          dom.docModalTabPreview?.click();
        } else {
          dom.docModalTabWrite?.click();
        }
      }
    },
  });
}

export const keybindingRegistry = new KeybindingRegistry();

export function initKeybindings() {
  keybindingRegistry.init();
}
