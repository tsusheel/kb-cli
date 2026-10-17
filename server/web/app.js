/**
 * KB-CLI Web Fuzzy Finder — Client Application
 */

(function () {
  'use strict';

  // State
  let allItems = [];
  let filteredItems = [];
  let selectedIndex = 0;
  let activeTypeFilter = 'all';
  let activeTagFilter = null;
  let currentNoteDetail = null;
  let isRawView = false;

  // DOM Elements
  const appContainer = document.getElementById('app');
  const searchInput = document.getElementById('search-input');
  const clearSearchBtn = document.getElementById('clear-search-btn');
  const searchCount = document.getElementById('search-count');
  const resultsList = document.getElementById('results-list');
  const mobileBackBtn = document.getElementById('mobile-back-btn');
  const previewEmpty = document.getElementById('preview-empty');
  const previewContent = document.getElementById('preview-content');
  const previewTitle = document.getElementById('preview-title');
  const previewShortId = document.getElementById('preview-short-id');
  const previewTypeBadge = document.getElementById('preview-type-badge');
  const previewStatusBadge = document.getElementById('preview-status-badge');
  const previewAreaBadge = document.getElementById('preview-area-badge');
  const previewDueBadge = document.getElementById('preview-due-badge');
  const previewUpdatedTime = document.getElementById('preview-updated-time');
  const previewTagsRow = document.getElementById('preview-tags-row');
  const previewMarkdownBody = document.getElementById('preview-markdown-body');
  const previewRawBody = document.getElementById('preview-raw-body');
  const previewGraphSection = document.getElementById('preview-graph-section');
  const previewLinksList = document.getElementById('preview-links-list');
  const previewHistorySection = document.getElementById('preview-history-section');
  const previewHistoryList = document.getElementById('preview-history-list');
  const toggleHistoryBtn = document.getElementById('toggle-history-btn');
  const typeFilterPills = document.getElementById('type-filter-pills');
  const tagPills = document.getElementById('tag-pills');
  const footerStats = document.getElementById('footer-stats');
  const newNoteBtn = document.getElementById('new-note-btn');
  const syncBtn = document.getElementById('sync-btn');
  const toggleViewBtn = document.getElementById('toggle-view-btn');
  const copyIdBtn = document.getElementById('copy-id-btn');
  const toggleStatusBtn = document.getElementById('toggle-status-btn');
  const editNoteBtn = document.getElementById('edit-note-btn');
  const deleteNoteBtn = document.getElementById('delete-note-btn');
  const noteModal = document.getElementById('note-modal');
  const noteForm = document.getElementById('note-form');
  const modalCloseBtn = document.getElementById('modal-close-btn');
  const modalCancelBtn = document.getElementById('modal-cancel-btn');
  const modalTitle = document.getElementById('modal-title');
  const modalTabWrite = document.getElementById('modal-tab-write');
  const modalTabPreview = document.getElementById('modal-tab-preview');
  const modalFleshPreview = document.getElementById('modal-flesh-preview');
  const formFlesh = document.getElementById('form-flesh');
  const toast = document.getElementById('toast');
  let toastTimer = null;

  // PWA Elements & State
  const pwaInstallBtn = document.getElementById('pwa-install-btn');
  const offlineBadge = document.getElementById('offline-badge');
  let deferredInstallPrompt = null;

  // Auth Elements & State
  const logoutBtn = document.getElementById('logout-btn');
  const authModal = document.getElementById('auth-modal');
  const authForm = document.getElementById('auth-form');
  const authPassword = document.getElementById('auth-password');
  const togglePasswordBtn = document.getElementById('toggle-password-btn');
  const authError = document.getElementById('auth-error');
  const authSubmitBtn = document.getElementById('auth-submit-btn');
  let isAuthenticated = false;
  let isAuthRequired = false;

  function showMobileDetail() {
    if (window.innerWidth <= 768 && appContainer) {
      appContainer.classList.add('mobile-show-preview');
    }
  }

  function showMobileList() {
    if (appContainer) {
      appContainer.classList.remove('mobile-show-preview');
    }
  }

  function showAuthModal() {
    if (authModal) {
      authModal.style.display = 'flex';
      if (authError) authError.style.display = 'none';
      if (authPassword) {
        authPassword.value = '';
        setTimeout(() => authPassword.focus(), 50);
      }
    }
  }

  function hideAuthModal() {
    if (authModal) {
      authModal.style.display = 'none';
    }
  }

  async function apiFetch(url, options = {}) {
    const res = await fetch(url, options);
    if (res.status === 401) {
      isAuthenticated = false;
      if (logoutBtn) logoutBtn.style.display = 'none';
      showAuthModal();
      throw new Error('Authentication required');
    }
    return res;
  }

  async function checkAuthStatus() {
    try {
      const res = await fetch('/api/auth/status');
      if (res.ok) {
        const data = await res.json();
        isAuthenticated = data.authenticated;
        isAuthRequired = data.auth_required;
        if (logoutBtn) {
          logoutBtn.style.display = (isAuthenticated && isAuthRequired) ? 'inline-flex' : 'none';
        }
        if (!isAuthenticated && isAuthRequired) {
          showAuthModal();
          return false;
        }
      }
    } catch (e) {
      console.warn('Auth status check failed:', e);
    }
    return true;
  }

  // Initialize
  async function init() {
    initMarkdownLibraries();
    initPWA();
    setupEventListeners();
    const canLoad = await checkAuthStatus();
    if (canLoad) {
      await loadData();
      await loadTags();
      await loadStats();
      handlePWAUrlAction();
    }
  }

  function initPWA() {
    // 1. Service Worker Registration
    if ('serviceWorker' in navigator) {
      window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js', { scope: '/' })
          .then(registration => {
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
          .catch(err => {
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

    // 2. Install Prompt Handling
    window.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault();
      deferredInstallPrompt = e;
      if (pwaInstallBtn) {
        pwaInstallBtn.style.display = 'inline-flex';
      }
    });

    window.addEventListener('appinstalled', () => {
      if (pwaInstallBtn) {
        pwaInstallBtn.style.display = 'none';
      }
      deferredInstallPrompt = null;
      showToast('KB App installed successfully! 🎉');
    });

    // 3. Online/Offline Network Monitoring
    const updateOnlineStatus = () => {
      const isOnline = navigator.onLine;
      if (offlineBadge) {
        offlineBadge.style.display = isOnline ? 'none' : 'inline-flex';
      }
    };

    window.addEventListener('online', () => {
      updateOnlineStatus();
      showToast('Connected — syncing latest data');
      loadData();
      loadTags();
      loadStats();
    });

    window.addEventListener('offline', () => {
      updateOnlineStatus();
      showToast('Working offline — cached notes available');
    });

    updateOnlineStatus();
  }

  function handlePWAUrlAction() {
    try {
      const params = new URLSearchParams(window.location.search);
      const action = params.get('action');
      if (action === 'new') {
        setTimeout(() => openNoteModal(), 200);
      } else if (action === 'search') {
        if (searchInput) searchInput.focus();
      }
    } catch (_) {}
  }

  // Event Listeners
  function setupEventListeners() {
    if (mobileBackBtn) {
      mobileBackBtn.addEventListener('click', () => {
        showMobileList();
      });
    }

    searchInput.addEventListener('input', () => {
      clearSearchBtn.style.display = searchInput.value ? 'block' : 'none';
      filterAndRender();
    });

    clearSearchBtn.addEventListener('click', () => {
      searchInput.value = '';
      clearSearchBtn.style.display = 'none';
      searchInput.focus();
      filterAndRender();
    });

    typeFilterPills.addEventListener('click', (e) => {
      const pill = e.target.closest('.pill');
      if (!pill) return;
      document.querySelectorAll('#type-filter-pills .pill').forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
      activeTypeFilter = pill.dataset.filter;
      filterAndRender();
    });

    tagPills.addEventListener('click', (e) => {
      const chip = e.target.closest('.tag-chip');
      if (!chip) return;
      const tag = chip.dataset.tag;
      if (activeTagFilter === tag) {
        activeTagFilter = null;
        chip.classList.remove('active');
      } else {
        document.querySelectorAll('.tag-chip').forEach(c => c.classList.remove('active'));
        activeTagFilter = tag;
        chip.classList.add('active');
      }
      filterAndRender();
    });

    resultsList.addEventListener('click', (e) => {
      const itemEl = e.target.closest('.list-item');
      if (!itemEl) return;
      const index = parseInt(itemEl.dataset.index, 10);
      if (!isNaN(index)) {
        selectItem(index);
        showMobileDetail();
      }
    });

    // Keyboard Shortcuts
    document.addEventListener('keydown', handleGlobalKeydown);

    // Header Actions
    if (pwaInstallBtn) {
      pwaInstallBtn.addEventListener('click', async () => {
        if (!deferredInstallPrompt) return;
        deferredInstallPrompt.prompt();
        const choice = await deferredInstallPrompt.userChoice;
        if (choice.outcome === 'accepted') {
          showToast('Installing KB App...');
        }
        deferredInstallPrompt = null;
        pwaInstallBtn.style.display = 'none';
      });
    }

    newNoteBtn.addEventListener('click', () => openNoteModal());
    if (syncBtn) {
      syncBtn.addEventListener('click', async () => {
        syncBtn.classList.add('syncing');
        syncBtn.textContent = 'Syncing...';
        showToast('Syncing data...');
        try {
          const res = await apiFetch('/api/sync', { method: 'POST' });
          const data = await res.json();
          if (data.success) {
            showToast(data.message || 'Sync complete');
          } else {
            showToast(`Sync failed: ${data.error || 'unknown error'}`);
          }
        } catch (err) {
          showToast(`Sync error: ${err.message}`);
        } finally {
          syncBtn.classList.remove('syncing');
          syncBtn.textContent = 'Sync';
          await loadData();
          await loadTags();
          await loadStats();
        }
      });
    }

    // Auth Listeners
    if (logoutBtn) {
      logoutBtn.addEventListener('click', async () => {
        try {
          await fetch('/api/auth/logout', { method: 'POST' });
        } catch (e) {
          console.warn(e);
        }
        isAuthenticated = false;
        logoutBtn.style.display = 'none';
        allItems = [];
        filteredItems = [];
        resultsList.innerHTML = '<div class="empty-list">Locked. Please authenticate to view notes.</div>';
        renderEmptyPreview();
        showAuthModal();
        showToast('Signed out');
      });
    }

    if (togglePasswordBtn && authPassword) {
      togglePasswordBtn.addEventListener('click', () => {
        const isPassword = authPassword.type === 'password';
        authPassword.type = isPassword ? 'text' : 'password';
        togglePasswordBtn.textContent = isPassword ? '🔒' : '👁';
      });
    }

    if (authForm) {
      authForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const password = authPassword.value;
        if (!password) return;

        authSubmitBtn.disabled = true;
        authSubmitBtn.textContent = 'Unlocking...';
        if (authError) authError.style.display = 'none';

        try {
          const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ password })
          });
          const data = await res.json();

          if (res.ok && data.authenticated) {
            isAuthenticated = true;
            isAuthRequired = data.auth_required !== false;
            hideAuthModal();
            if (logoutBtn) {
              logoutBtn.style.display = isAuthRequired ? 'inline-flex' : 'none';
            }
            showToast('Knowledge Base unlocked');
            await loadData();
            await loadTags();
            await loadStats();
            if (searchInput) searchInput.focus();
          } else {
            if (authError) {
              authError.textContent = data.error || 'Incorrect password. Please try again.';
              authError.style.display = 'block';
            }
            if (authPassword) {
              authPassword.select();
              authPassword.focus();
            }
          }
        } catch (err) {
          if (authError) {
            authError.textContent = `Connection error: ${err.message}`;
            authError.style.display = 'block';
          }
        } finally {
          authSubmitBtn.disabled = false;
          authSubmitBtn.textContent = 'Unlock';
        }
      });
    }

    // Preview Actions
    if (toggleViewBtn) {
      toggleViewBtn.addEventListener('click', () => toggleViewMode());
    }

    copyIdBtn.addEventListener('click', () => {
      const item = filteredItems[selectedIndex];
      if (item) {
        navigator.clipboard.writeText(item.id);
        showToast(`Copied ID: ${item.short_id}`);
      }
    });

    toggleStatusBtn.addEventListener('click', () => toggleCurrentItemStatus());
    editNoteBtn.addEventListener('click', () => {
      const item = filteredItems[selectedIndex];
      if (item && !item.is_log) {
        openNoteModal(item);
      }
    });

    deleteNoteBtn.addEventListener('click', () => deleteCurrentItem());

    toggleHistoryBtn.addEventListener('click', () => {
      const isVisible = previewHistoryList.style.display !== 'none';
      previewHistoryList.style.display = isVisible ? 'none' : 'flex';
      toggleHistoryBtn.textContent = isVisible ? 'Show' : 'Hide';
    });

    // Modal Actions
    modalCloseBtn.addEventListener('click', closeNoteModal);
    modalCancelBtn.addEventListener('click', closeNoteModal);
    noteForm.addEventListener('submit', handleNoteFormSubmit);
    noteModal.addEventListener('click', (e) => {
      if (e.target === noteModal) closeNoteModal();
    });

    // Modal Write / Preview Tabs
    if (modalTabWrite && modalTabPreview && formFlesh && modalFleshPreview) {
      modalTabWrite.addEventListener('click', () => {
        modalTabPreview.classList.remove('active');
        modalTabWrite.classList.add('active');
        modalFleshPreview.style.display = 'none';
        formFlesh.style.display = 'block';
        formFlesh.focus();
      });

      modalTabPreview.addEventListener('click', () => {
        modalTabWrite.classList.remove('active');
        modalTabPreview.classList.add('active');
        formFlesh.style.display = 'none';
        modalFleshPreview.style.display = 'block';
        const content = formFlesh.value;
        modalFleshPreview.innerHTML = renderMarkdown(content);
        applyPostMarkdownEnhancements(modalFleshPreview);
      });
    }

    // Delegated Code Block & Diagram Copy Handler
    document.addEventListener('click', async (e) => {
      const copyBtn = e.target.closest('.code-copy-btn');
      if (!copyBtn) return;
      e.preventDefault();
      e.stopPropagation();

      const wrapper = copyBtn.closest('.code-block-wrapper, .mermaid-block-wrapper');
      if (!wrapper) return;

      let textToCopy = '';
      const mermaidDiag = wrapper.querySelector('.mermaid-diagram');
      if (mermaidDiag) {
        textToCopy = mermaidDiag.getAttribute('data-raw-code') || mermaidDiag.textContent;
      } else {
        const codeEl = wrapper.querySelector('pre code');
        textToCopy = codeEl ? codeEl.textContent : '';
      }

      if (!textToCopy) return;

      try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
          await navigator.clipboard.writeText(textToCopy);
        } else {
          copyTextFallback(textToCopy);
        }
        triggerCopyFeedback(copyBtn);
      } catch (err) {
        copyTextFallback(textToCopy);
        triggerCopyFeedback(copyBtn);
      }
    });
  }

  function handleGlobalKeydown(e) {
    const isModalOpen = noteModal.style.display !== 'none';
    const isInputFocused = document.activeElement === searchInput || 
                           document.activeElement.tagName === 'INPUT' || 
                           document.activeElement.tagName === 'TEXTAREA' ||
                           document.activeElement.tagName === 'SELECT';

    if (e.key === 'Escape') {
      if (isModalOpen) {
        closeNoteModal();
        e.preventDefault();
      } else if (appContainer && appContainer.classList.contains('mobile-show-preview')) {
        showMobileList();
        e.preventDefault();
      } else if (searchInput.value) {
        searchInput.value = '';
        clearSearchBtn.style.display = 'none';
        filterAndRender();
        e.preventDefault();
      }
      return;
    }

    if (isModalOpen) return;

    // Focus search on '/' or 'Ctrl+K' / 'Cmd+K'
    if ((e.key === '/' && !isInputFocused) || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k')) {
      e.preventDefault();
      searchInput.focus();
      searchInput.select();
      return;
    }

    // Navigation (Arrow keys or j/k or Ctrl+N/Ctrl+P)
    if (e.key === 'ArrowDown' || (isInputFocused && (e.ctrlKey && e.key === 'n')) || (!isInputFocused && e.key === 'j')) {
      e.preventDefault();
      if (filteredItems.length > 0) {
        selectItem(Math.min(selectedIndex + 1, filteredItems.length - 1));
      }
      return;
    }

    if (e.key === 'ArrowUp' || (isInputFocused && (e.ctrlKey && e.key === 'p')) || (!isInputFocused && e.key === 'k')) {
      e.preventDefault();
      if (filteredItems.length > 0) {
        selectItem(Math.max(selectedIndex - 1, 0));
      }
      return;
    }

    // Actions when not typing in an input
    if (!isInputFocused) {
      if (e.key === 'v' || e.key === 'V') {
        e.preventDefault();
        toggleViewMode();
      } else if (e.key === 'n' || e.key === 'N') {
        e.preventDefault();
        openNoteModal();
      } else if (e.key === 't' || e.key === 'T') {
        e.preventDefault();
        toggleCurrentItemStatus();
      } else if (e.key === 'c' || e.key === 'C') {
        e.preventDefault();
        const item = filteredItems[selectedIndex];
        if (item) {
          navigator.clipboard.writeText(item.id);
          showToast(`Copied ID: ${item.short_id}`);
        }
      } else if (e.key === 'm' || e.key === 'M') {
        e.preventDefault();
        const item = filteredItems[selectedIndex];
        if (item) {
          const md = `# ${item.display}\n\n${item.flesh || ''}`;
          navigator.clipboard.writeText(md);
          showToast('Copied Note Markdown');
        }
      } else if (e.key === 'e' || e.key === 'E') {
        e.preventDefault();
        const item = filteredItems[selectedIndex];
        if (item && !item.is_log) {
          openNoteModal(item);
        }
      }
    }
  }

  // Data Loading
  async function loadData() {
    try {
      const includeDeleted = activeTypeFilter === 'archived';
      const res = await apiFetch(`/api/items?include_deleted=${includeDeleted}`);
      if (!res.ok) throw new Error('Failed to load items');
      allItems = await res.json();
      updateCounts();
      filterAndRender();
    } catch (err) {
      console.error(err);
      resultsList.innerHTML = `<div class="empty-list">Error loading data: ${err.message}</div>`;
    }
  }

  async function loadTags() {
    try {
      const res = await apiFetch('/api/tags');
      if (!res.ok) return;
      const tags = await res.json();
      if (!tags || tags.length === 0) {
        tagPills.innerHTML = '';
        return;
      }
      tagPills.innerHTML = tags.map(t => `
        <button class="tag-chip ${activeTagFilter === t.name ? 'active' : ''}" data-tag="${escapeHtml(t.name)}">
          #${escapeHtml(t.name)} (${t.count})
        </button>
      `).join('');
    } catch (err) {
      console.error(err);
    }
  }

  async function loadStats() {
    try {
      const res = await apiFetch('/api/stats');
      if (!res.ok) return;
      const stats = await res.json();
      if (stats) {
        const streak = stats.consecutive_streak_days || 0;
        footerStats.textContent = `${stats.total_active_notes || 0} notes · ${stats.total_daily_logs || 0} logs · ${streak}d streak`;
      }
    } catch (err) {
      console.error(err);
    }
  }

  function updateCounts() {
    const counts = { all: 0, note: 0, todo: 0, project: 0, idea: 0, log: 0, archived: 0 };
    allItems.forEach(item => {
      if (item.deleted_at) {
        counts.archived++;
      } else {
        counts.all++;
        if (item.is_log) {
          counts.log++;
        } else {
          const t = (item.type || 'note').toLowerCase();
          if (counts[t] !== undefined) counts[t]++;
        }
      }
    });

    document.getElementById('count-all').textContent = counts.all;
    document.getElementById('count-note').textContent = counts.note;
    document.getElementById('count-todo').textContent = counts.todo;
    document.getElementById('count-project').textContent = counts.project;
    document.getElementById('count-idea').textContent = counts.idea;
    document.getElementById('count-log').textContent = counts.log;
    document.getElementById('count-archived').textContent = counts.archived;
  }

  // Fuzzy Search Scoring Algorithm
  function fuzzyMatch(pattern, str) {
    if (!pattern) return { score: 1, positions: [] };
    if (!str) return null;

    pattern = pattern.toLowerCase();
    const target = str.toLowerCase();

    let patternIdx = 0;
    let targetIdx = 0;
    const positions = [];
    let score = 0;
    let consecutiveBonus = 0;

    while (patternIdx < pattern.length && targetIdx < target.length) {
      if (pattern[patternIdx] === target[targetIdx]) {
        positions.push(targetIdx);
        score += 10 + consecutiveBonus;
        consecutiveBonus += 5;
        if (targetIdx === 0 || target[targetIdx - 1] === ' ' || target[targetIdx - 1] === '/' || target[targetIdx - 1] === '-') {
          score += 15; // Word boundary bonus
        }
        patternIdx++;
      } else {
        consecutiveBonus = 0;
      }
      targetIdx++;
    }

    if (patternIdx === pattern.length) {
      return { score, positions };
    }
    return null;
  }

  function filterAndRender() {
    const query = searchInput.value.trim();

    // 1. Filter by tab & tag
    let items = allItems.filter(item => {
      if (activeTypeFilter === 'archived') {
        return !!item.deleted_at;
      }
      if (item.deleted_at) return false;

      if (activeTypeFilter === 'all') {
        // include all active notes + logs
      } else if (activeTypeFilter === 'log') {
        if (!item.is_log) return false;
      } else {
        if (item.is_log || (item.type || '').toLowerCase() !== activeTypeFilter) return false;
      }

      if (activeTagFilter) {
        if (!item.tags || !item.tags.includes(activeTagFilter)) return false;
      }

      return true;
    });

    // 2. Fuzzy rank if query exists
    if (query) {
      const scored = [];
      items.forEach(item => {
        const titleMatch = fuzzyMatch(query, item.display);
        const fleshMatch = fuzzyMatch(query, item.flesh);
        const tagMatch = item.tags ? fuzzyMatch(query, item.tags.join(' ')) : null;

        let bestScore = -1;
        let positions = [];

        if (titleMatch) {
          bestScore = Math.max(bestScore, titleMatch.score * 2);
          positions = titleMatch.positions;
        }
        if (tagMatch && tagMatch.score * 1.5 > bestScore) {
          bestScore = tagMatch.score * 1.5;
        }
        if (fleshMatch && fleshMatch.score > bestScore) {
          bestScore = fleshMatch.score;
        }

        if (bestScore > 0) {
          scored.push({ item, score: bestScore, positions });
        }
      });

      scored.sort((a, b) => b.score - a.score);
      filteredItems = scored.map(s => s.item);
    } else {
      filteredItems = items;
    }

    searchCount.textContent = `${filteredItems.length} item${filteredItems.length === 1 ? '' : 's'}`;
    renderList(query);

    if (filteredItems.length > 0) {
      selectedIndex = Math.min(selectedIndex, filteredItems.length - 1);
      selectItem(selectedIndex, false);
    } else {
      renderEmptyPreview();
    }
  }

  function renderList(query) {
    if (filteredItems.length === 0) {
      resultsList.innerHTML = `<div class="empty-list">No matching items found</div>`;
      return;
    }

    resultsList.innerHTML = filteredItems.map((item, idx) => {
      const isSelected = idx === selectedIndex;
      const type = item.is_log ? 'log' : (item.type || 'note');
      const badgeClass = `badge-${type.toLowerCase()}`;
      
      let highlightedTitle = escapeHtml(item.display);
      if (query) {
        const match = fuzzyMatch(query, item.display);
        if (match && match.positions.length > 0) {
          highlightedTitle = highlightPositions(item.display, match.positions);
        }
      }

      const tagsHtml = (item.tags || []).map(t => `<span class="mini-tag">#${escapeHtml(t)}</span>`).join(' ');

      return `
        <div class="list-item ${isSelected ? 'selected' : ''}" data-index="${idx}">
          <div class="list-item-header">
            <span class="item-badge ${badgeClass}">${type}</span>
            <span class="item-title">${highlightedTitle}</span>
            <span class="item-time">${escapeHtml(item.timestamp || '')}</span>
          </div>
          ${item.flesh ? `<div class="list-item-snippet">${escapeHtml(item.flesh.substring(0, 100))}</div>` : ''}
          ${tagsHtml ? `<div class="list-item-tags">${tagsHtml}</div>` : ''}
        </div>
      `;
    }).join('');

    // Ensure selected item is scrolled into view
    const selectedEl = resultsList.querySelector('.list-item.selected');
    if (selectedEl) {
      selectedEl.scrollIntoView({ block: 'nearest' });
    }
  }

  function highlightPositions(str, positions) {
    const posSet = new Set(positions);
    let out = '';
    for (let i = 0; i < str.length; i++) {
      if (posSet.has(i)) {
        out += `<span class="highlight-match">${escapeHtml(str[i])}</span>`;
      } else {
        out += escapeHtml(str[i]);
      }
    }
    return out;
  }

  // Item Selection & Preview
  async function selectItem(index, scroll = true) {
    selectedIndex = index;
    const items = resultsList.querySelectorAll('.list-item');
    items.forEach((el, i) => {
      el.classList.toggle('selected', i === selectedIndex);
    });

    if (scroll && items[selectedIndex]) {
      items[selectedIndex].scrollIntoView({ block: 'nearest' });
    }

    const item = filteredItems[selectedIndex];
    if (!item) {
      renderEmptyPreview();
      return;
    }

    renderPreviewBasic(item);

    if (!item.is_log) {
      try {
        const res = await apiFetch(`/api/notes/${item.id}`);
        if (res.ok) {
          currentNoteDetail = await res.json();
          renderPreviewDetail(currentNoteDetail);
        }
      } catch (err) {
        console.error('Error fetching note detail:', err);
      }
    }
  }

  function toggleViewMode() {
    isRawView = !isRawView;
    updateViewModeDisplay();
  }

  function updateViewModeDisplay() {
    if (toggleViewBtn) {
      toggleViewBtn.textContent = isRawView ? 'Preview' : 'Raw';
      toggleViewBtn.classList.toggle('active', isRawView);
      toggleViewBtn.title = isRawView ? 'Show Rendered Markdown (V)' : 'Show Raw Text (V)';
    }
    if (previewMarkdownBody && previewRawBody) {
      previewMarkdownBody.style.display = isRawView ? 'none' : 'block';
      previewRawBody.style.display = isRawView ? 'block' : 'none';
    }
  }

  function renderPreviewBasic(item) {
    previewEmpty.style.display = 'none';
    previewContent.style.display = 'block';

    previewTitle.textContent = item.display;
    previewShortId.textContent = item.short_id || item.id.substring(0, 7);

    const type = item.is_log ? 'log' : (item.type || 'note');
    previewTypeBadge.textContent = type;
    previewTypeBadge.className = `type-badge badge-${type.toLowerCase()}`;

    previewStatusBadge.textContent = item.status || (item.is_log ? 'recorded' : 'active');
    previewStatusBadge.className = `status-badge ${item.status === 'completed' ? 'status-completed' : ''}`;

    if (item.area) {
      previewAreaBadge.style.display = 'inline-block';
      previewAreaBadge.textContent = item.area;
    } else {
      previewAreaBadge.style.display = 'none';
    }

    previewDueBadge.style.display = 'none';
    previewUpdatedTime.textContent = item.timestamp ? `Updated ${item.timestamp}` : '';

    if (item.tags && item.tags.length > 0) {
      previewTagsRow.innerHTML = item.tags.map(t => `
        <span class="preview-tag-chip" data-tag="${escapeHtml(t)}">#${escapeHtml(t)}</span>
      `).join('');
    } else {
      previewTagsRow.innerHTML = '';
    }

    const rawContent = (item.flesh || item.display || '').trim();
    if (previewRawBody) previewRawBody.textContent = rawContent;
    if (previewMarkdownBody) {
      previewMarkdownBody.innerHTML = renderMarkdown(rawContent);
      applyPostMarkdownEnhancements(previewMarkdownBody);
    }
    updateViewModeDisplay();

    previewGraphSection.style.display = 'none';
    previewHistorySection.style.display = 'none';

    toggleStatusBtn.textContent = item.status === 'completed' ? 'Reopen' : 'Done';
    toggleStatusBtn.style.display = (item.type === 'todo' || item.type === 'project') ? 'inline-flex' : 'none';
    editNoteBtn.style.display = item.is_log ? 'none' : 'inline-flex';
    if (toggleViewBtn) toggleViewBtn.style.display = 'inline-flex';
  }

  function renderPreviewDetail(note) {
    if (note.target_date_time) {
      previewDueBadge.style.display = 'inline-block';
      previewDueBadge.textContent = `Due: ${note.target_date_time.split('T')[0]}`;
    }

    if (note.tags && note.tags.length > 0) {
      previewTagsRow.innerHTML = note.tags.map(t => `
        <span class="preview-tag-chip" data-tag="${escapeHtml(t)}">#${escapeHtml(t)}</span>
      `).join('');
    }

    const rawContent = (note.note_flesh || note.note || '').trim();
    if (previewRawBody) previewRawBody.textContent = rawContent;
    if (previewMarkdownBody) {
      previewMarkdownBody.innerHTML = renderMarkdown(rawContent);
      applyPostMarkdownEnhancements(previewMarkdownBody);
    }

    // Graph Links
    if (note.links && note.links.length > 0) {
      previewGraphSection.style.display = 'block';
      previewLinksList.innerHTML = note.links.map(l => `
        <div class="link-row" data-note-id="${escapeHtml(l.other_note_id)}">
          <span class="link-arrow">${l.direction === 'outgoing' ? '→' : '←'}</span>
          <span class="link-type">[${escapeHtml(l.type)}]</span>
          <span class="link-title">${escapeHtml(l.other_note || l.other_note_id.substring(0, 7))}</span>
        </div>
      `).join('');

      previewLinksList.querySelectorAll('.link-row').forEach(el => {
        el.addEventListener('click', () => {
          const targetId = el.dataset.noteId;
          const idx = filteredItems.findIndex(i => i.id === targetId || i.id.startsWith(targetId));
          if (idx !== -1) {
            selectItem(idx);
          } else {
            searchInput.value = targetId.substring(0, 7);
            filterAndRender();
          }
        });
      });
    } else {
      previewGraphSection.style.display = 'none';
    }

    // Revision History
    if (note.history && note.history.length > 0) {
      previewHistorySection.style.display = 'block';
      previewHistoryList.innerHTML = note.history.map(h => `
        <div class="history-item">
          <div>
            <span class="history-action">${escapeHtml(h.action)}</span>
            <span class="history-summary">${escapeHtml(h.changes_summary || '')}</span>
          </div>
          <span class="history-time">${escapeHtml(h.created_at ? h.created_at.split('T')[0] : '')}</span>
        </div>
      `).join('');
    } else {
      previewHistorySection.style.display = 'none';
    }
  }

  function renderEmptyPreview() {
    previewEmpty.style.display = 'block';
    previewContent.style.display = 'none';
  }

  // Markdown Parser
  function renderMarkdown(text) {
    if (!text || !text.trim()) {
      return '<p><em>(no content)</em></p>';
    }

    // Step 1: Normalize line endings
    text = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');

    // Step 2: Extract code blocks
    const codeBlocks = [];
    text = text.replace(/(?:^|\n)```([a-zA-Z0-9_+-]*)\n([\s\S]*?)\n```/g, (match, lang, code) => {
      const id = codeBlocks.length;
      codeBlocks.push({ lang: lang.trim(), rawCode: code, escapedCode: escapeHtml(code) });
      return `\n\n%%BLOCK_CODE_${id}%%\n\n`;
    });

    // Handle code blocks at EOF without trailing newline
    text = text.replace(/(?:^|\n)```([a-zA-Z0-9_+-]*)\n([\s\S]*?)```$/g, (match, lang, code) => {
      const id = codeBlocks.length;
      codeBlocks.push({ lang: lang.trim(), rawCode: code, escapedCode: escapeHtml(code) });
      return `\n\n%%BLOCK_CODE_${id}%%\n\n`;
    });

    // Step 3: Extract inline code
    const inlineCodes = [];
    text = text.replace(/`([^`\n]+)`/g, (match, code) => {
      const id = inlineCodes.length;
      inlineCodes.push(escapeHtml(code));
      return `%%INLINE_CODE_${id}%%`;
    });

    // Inline formatting processor
    function parseInline(str) {
      if (!str) return '';

      let s = escapeHtml(str);

      // Links: [text](url)
      s = s.replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
      s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');

      // Auto links: https://...
      s = s.replace(/(^|[\s(])(https?:\/\/[^\s)<]+)/g, '$1<a href="$2" target="_blank" rel="noopener noreferrer">$2</a>');

      // Bold: **text** or __text__
      s = s.replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>');
      s = s.replace(/__([^_\n]+)__/g, '<strong>$1</strong>');

      // Italic: *text* or _text_
      s = s.replace(/\*([^*\n]+)\*/g, '<em>$1</em>');
      s = s.replace(/(^|[^\w])_([^_\n]+)_(?=[^\w]|$)/g, '$1<em>$2</em>');

      // Strikethrough: ~~text~~
      s = s.replace(/~~([^~\n]+)~~/g, '<del>$1</del>');

      return s;
    }

    // Block-level processor
    function parseBlocks(src) {
      const lines = src.split('\n');
      const out = [];
      let i = 0;

      while (i < lines.length) {
        const line = lines[i];

        // 1. Code block token placeholder
        const codeBlockMatch = line.trim().match(/^%%BLOCK_CODE_(\d+)%%$/);
        if (codeBlockMatch) {
          const id = parseInt(codeBlockMatch[1], 10);
          const block = codeBlocks[id];
          if (block) {
            const rawLang = (block.lang || '').toLowerCase();
            if (rawLang === 'mermaid') {
              const safeRaw = escapeHtml(block.rawCode.trim());
              out.push(`
                <div class="mermaid-block-wrapper">
                  <div class="mermaid-block-header">
                    <div class="mermaid-header-left">
                      <span class="mermaid-tag">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg>
                        Mermaid Diagram
                      </span>
                      <span class="zoom-level-badge">100%</span>
                    </div>
                    <div class="mermaid-header-actions">
                      <button type="button" class="diagram-btn zoom-in-btn" title="Zoom In (+)" aria-label="Zoom In">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                      </button>
                      <button type="button" class="diagram-btn zoom-out-btn" title="Zoom Out (-)" aria-label="Zoom Out">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                      </button>
                      <button type="button" class="diagram-btn zoom-reset-btn" title="Reset View (Double-click)" aria-label="Reset View">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path><path d="M3 3v5h5"></path></svg>
                      </button>
                      <button type="button" class="diagram-btn fullscreen-btn" title="Expand Fullscreen (⛶)" aria-label="Fullscreen">
                        <svg class="fs-expand-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/></svg>
                        <svg class="fs-compress-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="display:none;"><path d="M4 14h6v6M20 10h-6V4M14 10l7-7M10 14l-7 7"/></svg>
                      </button>
                      <button type="button" class="code-copy-btn" title="Copy Diagram Source" aria-label="Copy Diagram Source">
                        <svg class="copy-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                        <svg class="check-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="display:none;"><polyline points="20 6 9 17 4 12"></polyline></svg>
                        <span class="copy-text">Copy</span>
                      </button>
                    </div>
                  </div>
                  <div class="mermaid-viewport" data-raw-code="${safeRaw}">
                    <div class="mermaid-diagram" data-raw-code="${safeRaw}">${safeRaw}</div>
                  </div>
                </div>
              `);
            } else {
              const langDisplay = block.lang || 'code';
              const langClass = block.lang ? ` class="hljs language-${escapeHtml(block.lang)}"` : ' class="hljs"';
              out.push(`
                <div class="code-block-wrapper">
                  <div class="code-block-header">
                    <span class="code-lang-tag">${escapeHtml(langDisplay)}</span>
                    <button type="button" class="code-copy-btn" title="Copy Code" aria-label="Copy Code">
                      <svg class="copy-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                      <svg class="check-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none;"><polyline points="20 6 9 17 4 12"></polyline></svg>
                      <span class="copy-text">Copy</span>
                    </button>
                  </div>
                  <pre><code${langClass}>${block.escapedCode}</code></pre>
                </div>
              `);
            }
          }
          i++;
          continue;
        }

        // 2. Blank line
        if (line.trim() === '') {
          i++;
          continue;
        }

        // 3. Headers (# to ######)
        const headerMatch = line.match(/^(#{1,6})[ \t]+(.*)$/);
        if (headerMatch) {
          const level = headerMatch[1].length;
          const content = parseInline(headerMatch[2].trim());
          out.push(`<h${level}>${content}</h${level}>`);
          i++;
          continue;
        }

        // 4. Horizontal rule
        if (/^[ \t]*(?:[-*_][ \t]*){3,}$/.test(line)) {
          out.push('<hr>');
          i++;
          continue;
        }

        // 5. Blockquote (groups contiguous > lines)
        if (/^[ \t]*>/.test(line)) {
          const quoteLines = [];
          while (i < lines.length && /^[ \t]*>/.test(lines[i])) {
            quoteLines.push(lines[i].replace(/^[ \t]*>[ \t]?/, ''));
            i++;
          }
          const innerHtml = parseBlocks(quoteLines.join('\n'));
          out.push(`<blockquote>${innerHtml}</blockquote>`);
          continue;
        }

        // 6. Table (lines starting and ending with |)
        if (/^[ \t]*\|.+?\|[ \t]*$/.test(line)) {
          const tableLines = [];
          while (i < lines.length && /^[ \t]*\|.+?\|[ \t]*$/.test(lines[i])) {
            tableLines.push(lines[i].trim());
            i++;
          }
          if (tableLines.length >= 1) {
            const parseRow = (l) => l.slice(1, -1).split('|').map(c => parseInline(c.trim()));
            let startIdx = 0;
            let hasHeaderDivider = false;
            if (tableLines.length >= 2 && /^\|(?:\s*:?-+:?\s*\|)+$/.test(tableLines[1])) {
              hasHeaderDivider = true;
            }

            let tHtml = '<table>';
            if (hasHeaderDivider) {
              const headers = parseRow(tableLines[0]);
              tHtml += '<thead><tr>' + headers.map(h => `<th>${h}</th>`).join('') + '</tr></thead><tbody>';
              startIdx = 2;
            } else {
              tHtml += '<tbody>';
              startIdx = 0;
            }

            for (let r = startIdx; r < tableLines.length; r++) {
              if (/^\|(?:\s*:?-+:?\s*\|)+$/.test(tableLines[r])) continue;
              const cells = parseRow(tableLines[r]);
              tHtml += '<tr>' + cells.map(c => `<td>${c}</td>`).join('') + '</tr>';
            }
            tHtml += '</tbody></table>';
            out.push(tHtml);
            continue;
          }
        }

        // 7. Task list or Unordered list
        if (/^[ \t]*[-*][ \t]/.test(line)) {
          const listItems = [];
          let isTaskList = false;
          while (i < lines.length && /^[ \t]*[-*][ \t]/.test(lines[i])) {
            const curLine = lines[i];
            const taskMatch = curLine.match(/^[ \t]*[-*][ \t]\[([ xX])\][ \t]+(.*)$/);
            if (taskMatch) {
              isTaskList = true;
              const checked = taskMatch[1].toLowerCase() === 'x';
              const content = parseInline(taskMatch[2].trim());
              listItems.push(`<li class="task-item"><input type="checkbox" ${checked ? 'checked' : ''} disabled> <span>${content}</span></li>`);
            } else {
              const content = parseInline(curLine.replace(/^[ \t]*[-*][ \t]+/, '').trim());
              listItems.push(`<li>${content}</li>`);
            }
            i++;
          }
          const listClass = isTaskList ? ' class="task-list"' : '';
          out.push(`<ul${listClass}>${listItems.join('')}</ul>`);
          continue;
        }

        // 8. Ordered list
        if (/^[ \t]*\d+\.[ \t]/.test(line)) {
          const listItems = [];
          while (i < lines.length && /^[ \t]*\d+\.[ \t]/.test(lines[i])) {
            const content = parseInline(lines[i].replace(/^[ \t]*\d+\.[ \t]+/, '').trim());
            listItems.push(`<li>${content}</li>`);
            i++;
          }
          out.push(`<ol>${listItems.join('')}</ol>`);
          continue;
        }

        // 9. Regular paragraph (group contiguous non-block lines)
        const paraLines = [];
        while (i < lines.length && lines[i].trim() !== '' &&
               !/^(#{1,6})[ \t]+/.test(lines[i]) &&
               !/^[ \t]*>/.test(lines[i]) &&
               !/^[ \t]*\|.+?\|[ \t]*$/.test(lines[i]) &&
               !/^[ \t]*[-*][ \t]/.test(lines[i]) &&
               !/^[ \t]*\d+\.[ \t]/.test(lines[i]) &&
               !/^[ \t]*(?:[-*_][ \t]*){3,}$/.test(lines[i]) &&
               !/^%%BLOCK_CODE_/.test(lines[i].trim())) {
          paraLines.push(parseInline(lines[i].trim()));
          i++;
        }
        if (paraLines.length > 0) {
          out.push(`<p>${paraLines.join('<br>')}</p>`);
        }
      }

      return out.join('\n');
    }

    // Step 4: Parse blocks
    let html = parseBlocks(text);

    // Step 5: Restore inline code
    html = html.replace(/%%INLINE_CODE_(\d+)%%/g, (match, id) => {
      const code = inlineCodes[parseInt(id, 10)] || '';
      return `<code>${code}</code>`;
    });

    return html;
  }

  // Initialize Markdown libraries
  function initMarkdownLibraries() {
    if (window.mermaid) {
      try {
        window.mermaid.initialize({
          startOnLoad: false,
          theme: 'dark',
          securityLevel: 'loose',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
          themeVariables: {
            darkMode: true,
            background: '#0b0e14',
            primaryColor: '#1f6feb',
            primaryTextColor: '#c9d1d9',
            primaryBorderColor: '#30363d',
            lineColor: '#8b949e',
            secondaryColor: '#161b22',
            tertiaryColor: '#21262d',
            fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif'
          }
        });
      } catch (e) {
        console.warn('Failed to initialize mermaid:', e);
      }
    }
  }

  // Post-render enhancements (Highlight.js, Mermaid.js)
  async function applyPostMarkdownEnhancements(container) {
    if (!container) return;

    // 1. Syntax highlighting with Highlight.js
    if (window.hljs) {
      container.querySelectorAll('.code-block-wrapper pre code').forEach((block) => {
        const wrapper = block.closest('.code-block-wrapper');
        const langTag = wrapper ? wrapper.querySelector('.code-lang-tag') : null;
        const classes = Array.from(block.classList);
        const hasSpecificLang = classes.some(c => c.startsWith('language-'));

        if (hasSpecificLang) {
          try {
            window.hljs.highlightElement(block);
          } catch (e) {
            console.debug('Highlight error:', e);
          }
        } else {
          // Auto-detect language
          const codeText = block.textContent;
          if (codeText && codeText.trim().length > 0) {
            try {
              const res = window.hljs.highlightAuto(codeText);
              if (res && res.value) {
                block.innerHTML = res.value;
                block.classList.add('hljs');
                if (res.language && langTag && (langTag.textContent === 'code' || !langTag.textContent)) {
                  langTag.textContent = res.language;
                }
              }
            } catch (e) {
              try { window.hljs.highlightElement(block); } catch (e2) {}
            }
          }
        }
      });
    }

    // 2. Mermaid diagram rendering
    if (window.mermaid) {
      const diagrams = container.querySelectorAll('.mermaid-diagram:not([data-processed="true"])');
      for (let idx = 0; idx < diagrams.length; idx++) {
        const diag = diagrams[idx];
        const rawCode = diag.getAttribute('data-raw-code') || diag.textContent;
        const uniqueId = `mermaid-svg-${Date.now()}-${idx}-${Math.floor(Math.random() * 10000)}`;

        try {
          const { svg } = await window.mermaid.render(uniqueId, rawCode.trim());
          diag.innerHTML = svg;
          diag.setAttribute('data-processed', 'true');
          const wrapper = diag.closest('.mermaid-block-wrapper');
          if (wrapper) {
            setupMermaidPanZoom(wrapper);
          }
        } catch (err) {
          console.warn('Mermaid rendering failed for snippet:', err);
          diag.setAttribute('data-processed', 'error');
          diag.innerHTML = `
            <div class="mermaid-error">
              <div class="mermaid-error-title">⚠️ Diagram Render Error</div>
              <pre class="mermaid-error-raw"><code>${escapeHtml(rawCode.trim())}</code></pre>
            </div>
          `;
          const orphaned = document.getElementById(uniqueId) || document.getElementById(`d${uniqueId}`);
          if (orphaned) orphaned.remove();
        }
      }
    }
  }

  // Interactive Native SVG Pan & Zoom Controller for Mermaid diagrams
  function setupMermaidPanZoom(wrapper) {
    if (!wrapper || wrapper.dataset.panzoomInitialized) return;
    wrapper.dataset.panzoomInitialized = 'true';

    const viewport = wrapper.querySelector('.mermaid-viewport');
    const diag = wrapper.querySelector('.mermaid-diagram');
    const zoomBadge = wrapper.querySelector('.zoom-level-badge');
    const zoomInBtn = wrapper.querySelector('.zoom-in-btn');
    const zoomOutBtn = wrapper.querySelector('.zoom-out-btn');
    const resetBtn = wrapper.querySelector('.zoom-reset-btn');
    const fullscreenBtn = wrapper.querySelector('.fullscreen-btn');

    if (!viewport || !diag) return;

    const svg = diag.querySelector('svg');
    if (!svg) return;

    // 1. Read initial viewBox to obtain true vector boundaries
    let minX = 0;
    let minY = 0;
    let origW = 800;
    let origH = 400;

    const vb = svg.getAttribute('viewBox');
    if (vb) {
      const parts = vb.trim().split(/[\s,]+/).map(parseFloat);
      if (parts.length >= 4 && parts[2] > 0 && parts[3] > 0) {
        minX = parts[0] || 0;
        minY = parts[1] || 0;
        origW = parts[2];
        origH = parts[3];
      }
    } else {
      try {
        const bbox = svg.getBBox();
        if (bbox && bbox.width > 0 && bbox.height > 0) {
          minX = bbox.x || 0;
          minY = bbox.y || 0;
          origW = bbox.width;
          origH = bbox.height;
        }
      } catch (_) {}
    }

    // 2. Prepare SVG for native vector viewport (remove max-width and fixed viewBox)
    svg.removeAttribute('viewBox');
    svg.removeAttribute('width');
    svg.removeAttribute('height');
    svg.style.maxWidth = 'none';
    svg.style.maxHeight = 'none';
    svg.style.width = '100%';
    svg.style.height = '100%';
    svg.style.display = 'block';
    svg.style.overflow = 'hidden';

    // 3. Wrap diagram elements into a dedicated SVG vector panzoom layer
    let panzoomLayer = svg.querySelector(':scope > g.mermaid-panzoom-layer');
    if (!panzoomLayer) {
      panzoomLayer = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      panzoomLayer.setAttribute('class', 'mermaid-panzoom-layer');

      const children = Array.from(svg.childNodes);
      children.forEach((child) => {
        const tag = child.tagName ? child.tagName.toLowerCase() : '';
        // Keep styles and defs at the root of the SVG
        if (tag === 'style' || tag === 'defs' || tag === 'metadata') {
          return;
        }
        panzoomLayer.appendChild(child);
      });
      svg.appendChild(panzoomLayer);
    }

    let scale = 1;
    let tx = 0;
    let ty = 0;
    let isDragging = false;
    let startX = 0;
    let startY = 0;
    const MIN_SCALE = 0.2;
    const MAX_SCALE = 6.0;

    function applyTransform(smooth = false) {
      panzoomLayer.classList.toggle('smooth-transition', smooth);
      panzoomLayer.setAttribute('transform', `matrix(${scale}, 0, 0, ${scale}, ${tx}, ${ty})`);
      if (zoomBadge) {
        zoomBadge.textContent = `${Math.round(scale * 100)}%`;
      }
    }

    function clampPan() {
      const vRect = viewport.getBoundingClientRect();
      if (vRect.width <= 0 || vRect.height <= 0) return;
      const margin = 50;
      const diagramW = origW * scale;
      const diagramH = origH * scale;

      const minTx = margin - (minX * scale + diagramW);
      const maxTx = vRect.width - margin - (minX * scale);
      const minTy = margin - (minY * scale + diagramH);
      const maxTy = vRect.height - margin - (minY * scale);

      if (minTx <= maxTx) {
        tx = Math.min(Math.max(tx, minTx), maxTx);
      }
      if (minTy <= maxTy) {
        ty = Math.min(Math.max(ty, minTy), maxTy);
      }
    }

    function fitDiagram(smooth = false) {
      const vRect = viewport.getBoundingClientRect();
      if (vRect.width <= 0 || vRect.height <= 0) return;

      const availW = Math.max(vRect.width - 60, 100);
      const availH = Math.max(vRect.height - 60, 100);

      const ratioW = availW / origW;
      const ratioH = availH / origH;
      const fitScale = Math.min(ratioW, ratioH, 1.0);

      scale = Math.min(Math.max(fitScale, MIN_SCALE), 1.0);
      tx = (vRect.width - origW * scale) / 2 - minX * scale;
      ty = (vRect.height - origH * scale) / 2 - minY * scale;

      clampPan();
      applyTransform(smooth);
    }

    // Zoom centered strictly on a focal point (focalX, focalY) relative to viewport
    function zoomTowards(newTargetScale, focalX, focalY, smooth = false) {
      const clampedScale = Math.min(Math.max(newTargetScale, MIN_SCALE), MAX_SCALE);
      if (clampedScale === scale) return;

      // Mathematical focal point preservation:
      // The point under (focalX, focalY) remains stationary
      tx = focalX - (focalX - tx) * (clampedScale / scale);
      ty = focalY - (focalY - ty) * (clampedScale / scale);
      scale = clampedScale;

      clampPan();
      applyTransform(smooth);
    }

    // Mouse wheel zoom centered exactly on cursor position with smooth exponential curve
    viewport.addEventListener('wheel', (e) => {
      e.preventDefault();
      e.stopPropagation();

      const vRect = viewport.getBoundingClientRect();
      const cursorX = e.clientX - vRect.left;
      const cursorY = e.clientY - vRect.top;

      const deltaMultiplier = e.deltaMode === 1 ? 24 : (e.deltaMode === 2 ? 400 : 1);
      const delta = -e.deltaY * deltaMultiplier;
      const zoomFactor = Math.min(Math.max(Math.exp(delta * 0.0015), 0.75), 1.35);

      zoomTowards(scale * zoomFactor, cursorX, cursorY, false);
    }, { passive: false });

    // Pointer-based Panning (Mouse & Single Touch via Pointer Events)
    viewport.addEventListener('pointerdown', (e) => {
      if (e.button !== 0) return;
      isDragging = true;
      startX = e.clientX - tx;
      startY = e.clientY - ty;
      viewport.classList.add('is-dragging');
      panzoomLayer.classList.remove('smooth-transition');
      try { viewport.setPointerCapture(e.pointerId); } catch (_) {}
    });

    viewport.addEventListener('pointermove', (e) => {
      if (!isDragging) return;
      tx = e.clientX - startX;
      ty = e.clientY - startY;
      clampPan();
      applyTransform(false);
    });

    const stopDragging = (e) => {
      if (isDragging) {
        isDragging = false;
        viewport.classList.remove('is-dragging');
        try { viewport.releasePointerCapture(e.pointerId); } catch (_) {}
      }
    };

    viewport.addEventListener('pointerup', stopDragging);
    viewport.addEventListener('pointercancel', stopDragging);

    // Double-click to reset view
    viewport.addEventListener('dblclick', (e) => {
      e.preventDefault();
      fitDiagram(true);
    });

    // Toolbar Buttons
    if (zoomInBtn) {
      zoomInBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const vRect = viewport.getBoundingClientRect();
        zoomTowards(scale * 1.25, vRect.width / 2, vRect.height / 2, true);
      });
    }

    if (zoomOutBtn) {
      zoomOutBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        const vRect = viewport.getBoundingClientRect();
        zoomTowards(scale * 0.8, vRect.width / 2, vRect.height / 2, true);
      });
    }

    if (resetBtn) {
      resetBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        fitDiagram(true);
      });
    }

    // Fullscreen toggle
    function toggleFullscreen() {
      const isFs = wrapper.classList.toggle('mermaid-fullscreen');
      fullscreenBtn.title = isFs ? 'Exit Fullscreen (Esc)' : 'Expand Fullscreen (⛶)';
      setTimeout(() => {
        fitDiagram(false);
      }, 60);
    }

    if (fullscreenBtn) {
      fullscreenBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        toggleFullscreen();
      });
    }

    // Two-finger pinch-to-zoom on touch devices
    let initialPinchDist = 0;
    let initialPinchScale = 1;

    viewport.addEventListener('touchstart', (e) => {
      if (e.touches.length === 2) {
        isDragging = false;
        initialPinchDist = Math.hypot(
          e.touches[0].clientX - e.touches[1].clientX,
          e.touches[0].clientY - e.touches[1].clientY
        );
        initialPinchScale = scale;
      }
    }, { passive: true });

    viewport.addEventListener('touchmove', (e) => {
      if (e.touches.length === 2 && initialPinchDist > 0) {
        const currentDist = Math.hypot(
          e.touches[0].clientX - e.touches[1].clientX,
          e.touches[0].clientY - e.touches[1].clientY
        );
        const vRect = viewport.getBoundingClientRect();
        const midX = (e.touches[0].clientX + e.touches[1].clientX) / 2 - vRect.left;
        const midY = (e.touches[0].clientY + e.touches[1].clientY) / 2 - vRect.top;
        const factor = currentDist / initialPinchDist;
        zoomTowards(initialPinchScale * factor, midX, midY, false);
      }
    }, { passive: true });

    viewport.addEventListener('touchend', (e) => {
      if (e.touches.length < 2) {
        initialPinchDist = 0;
      }
    });

    // Esc key exits fullscreen
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape' && wrapper.classList.contains('mermaid-fullscreen')) {
        wrapper.classList.remove('mermaid-fullscreen');
        if (fullscreenBtn) fullscreenBtn.title = 'Expand Fullscreen (⛶)';
        setTimeout(() => fitDiagram(false), 60);
      }
    });

    // Dynamic resize handling
    let hasFitted = false;
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const cr = entry.contentRect;
        if (cr.width > 0 && cr.height > 0) {
          if (!hasFitted) {
            hasFitted = true;
            fitDiagram(false);
          }
        }
      }
    });
    ro.observe(viewport);

    // Initial fit
    requestAnimationFrame(() => {
      fitDiagram(false);
    });
  }

  function copyTextFallback(text) {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    try {
      document.execCommand('copy');
    } catch (e) {
      showToast('Could not copy to clipboard');
    }
    document.body.removeChild(ta);
  }

  function triggerCopyFeedback(btn) {
    const textSpan = btn.querySelector('.copy-text');
    const origText = textSpan ? textSpan.textContent : 'Copy';
    btn.classList.add('copied');
    if (textSpan) textSpan.textContent = 'Copied!';

    setTimeout(() => {
      btn.classList.remove('copied');
      if (textSpan) textSpan.textContent = origText;
    }, 2000);
  }

  function resetModalTabs() {
    if (modalTabWrite && modalTabPreview && formFlesh && modalFleshPreview) {
      modalTabPreview.classList.remove('active');
      modalTabWrite.classList.add('active');
      modalFleshPreview.style.display = 'none';
      formFlesh.style.display = 'block';
      modalFleshPreview.innerHTML = '';
    }
  }

  // API Mutations
  async function toggleCurrentItemStatus() {
    const item = filteredItems[selectedIndex];
    if (!item || item.is_log) return;

    const newStatus = item.status === 'completed' ? 'active' : 'completed';
    try {
      const res = await apiFetch(`/api/notes/${item.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus })
      });
      if (res.ok) {
        item.status = newStatus;
        showToast(`Status updated to: ${newStatus}`);
        selectItem(selectedIndex, false);
      }
    } catch (err) {
      showToast(`Failed to update status: ${err.message}`);
    }
  }

  async function deleteCurrentItem() {
    const item = filteredItems[selectedIndex];
    if (!item) return;

    if (!confirm(`Are you sure you want to soft-delete "${item.display}"?`)) return;

    try {
      const res = await apiFetch(`/api/notes/${item.id}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: 'deleted from web UI' })
      });
      if (res.ok) {
        showToast(`Note soft-deleted`);
        showMobileList();
        await loadData();
      }
    } catch (err) {
      showToast(`Delete failed: ${err.message}`);
    }
  }

  // Modal Handlers
  function openNoteModal(editItem = null) {
    noteModal.style.display = 'flex';
    resetModalTabs();
    if (editItem) {
      const flesh = (currentNoteDetail && (currentNoteDetail.id === editItem.id || currentNoteDetail.id.startsWith(editItem.id))) ? 
        (currentNoteDetail.note_flesh || editItem.flesh || '') : (editItem.flesh || '');
      const dueVal = (currentNoteDetail && (currentNoteDetail.id === editItem.id || currentNoteDetail.id.startsWith(editItem.id)) && currentNoteDetail.target_date_time) ? 
        currentNoteDetail.target_date_time.split('T')[0] : (editItem.due || (editItem.target_date_time ? editItem.target_date_time.split('T')[0] : ''));
      modalTitle.textContent = 'Edit Note';
      document.getElementById('form-note-id').value = editItem.id;
      document.getElementById('form-title').value = editItem.display;
      document.getElementById('form-type').value = editItem.type || 'note';
      document.getElementById('form-status').value = editItem.status || 'active';
      document.getElementById('form-area').value = editItem.area || '';
      document.getElementById('form-due').value = dueVal;
      document.getElementById('form-tags').value = (editItem.tags || []).join(', ');
      document.getElementById('form-flesh').value = flesh;
    } else {
      modalTitle.textContent = 'Create New Note';
      noteForm.reset();
      document.getElementById('form-note-id').value = '';
    }
    document.getElementById('form-title').focus();
  }

  function closeNoteModal() {
    noteModal.style.display = 'none';
    noteForm.reset();
    resetModalTabs();
  }

  async function handleNoteFormSubmit(e) {
    e.preventDefault();
    const noteId = document.getElementById('form-note-id').value;
    const title = document.getElementById('form-title').value.trim();
    const type = document.getElementById('form-type').value;
    const status = document.getElementById('form-status').value;
    const area = document.getElementById('form-area').value.trim();
    const due = document.getElementById('form-due').value.trim();
    const tags = document.getElementById('form-tags').value.split(',').map(t => t.trim()).filter(Boolean);
    const flesh = document.getElementById('form-flesh').value;

    const payload = {
      note: title,
      content: flesh,
      type: type,
      status: status,
      area: area,
      due: due,
      tags: tags
    };

    try {
      if (noteId) {
        // Edit existing
        const res = await apiFetch(`/api/notes/${noteId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        if (!res.ok) throw new Error('Failed to update note');
        showToast('Note updated successfully');
      } else {
        // Create new
        const res = await apiFetch('/api/notes', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        if (!res.ok) throw new Error('Failed to create note');
        showToast('Note created successfully');
      }
      closeNoteModal();
      await loadData();
      await loadTags();
    } catch (err) {
      showToast(`Error: ${err.message}`);
    }
  }

  function showToast(message, actionText, actionCallback) {
    if (toastTimer) {
      clearTimeout(toastTimer);
      toastTimer = null;
    }
    toast.innerHTML = '';
    const span = document.createElement('span');
    span.textContent = message;
    toast.appendChild(span);

    if (actionText && actionCallback) {
      const btn = document.createElement('button');
      btn.className = 'toast-action-btn';
      btn.textContent = actionText;
      btn.addEventListener('click', () => {
        toast.style.display = 'none';
        actionCallback();
      });
      toast.appendChild(btn);
    }

    toast.style.display = 'flex';
    if (!actionText) {
      toastTimer = setTimeout(() => {
        toast.style.display = 'none';
      }, 3000);
    }
  }

  function escapeHtml(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // Start app
  window.addEventListener('DOMContentLoaded', init);
})();
