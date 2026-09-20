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
  const searchInput = document.getElementById('search-input');
  const clearSearchBtn = document.getElementById('clear-search-btn');
  const searchCount = document.getElementById('search-count');
  const resultsList = document.getElementById('results-list');
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
  const toast = document.getElementById('toast');

  // Initialize
  async function init() {
    setupEventListeners();
    await loadData();
    await loadTags();
    await loadStats();
  }

  // Event Listeners
  function setupEventListeners() {
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
      }
    });

    // Keyboard Shortcuts
    document.addEventListener('keydown', handleGlobalKeydown);

    // Header Actions
    newNoteBtn.addEventListener('click', () => openNoteModal());
    if (syncBtn) {
      syncBtn.addEventListener('click', async () => {
        syncBtn.classList.add('syncing');
        syncBtn.textContent = 'Syncing...';
        showToast('Syncing data...');
        try {
          const res = await fetch('/api/sync', { method: 'POST' });
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
      const res = await fetch(`/api/items?include_deleted=${includeDeleted}`);
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
      const res = await fetch('/api/tags');
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
      const res = await fetch('/api/stats');
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
        const res = await fetch(`/api/notes/${item.id}`);
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
    if (previewMarkdownBody) previewMarkdownBody.innerHTML = renderMarkdown(rawContent);
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
    if (previewMarkdownBody) previewMarkdownBody.innerHTML = renderMarkdown(rawContent);

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
      codeBlocks.push({ lang: lang.trim(), code: escapeHtml(code) });
      return `\n\n%%BLOCK_CODE_${id}%%\n\n`;
    });

    // Handle code blocks at EOF without trailing newline
    text = text.replace(/(?:^|\n)```([a-zA-Z0-9_+-]*)\n([\s\S]*?)```$/g, (match, lang, code) => {
      const id = codeBlocks.length;
      codeBlocks.push({ lang: lang.trim(), code: escapeHtml(code) });
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
            const langClass = block.lang ? ` class="lang-${block.lang}"` : '';
            out.push(`<pre><code${langClass}>${block.code}</code></pre>`);
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

  // API Mutations
  async function toggleCurrentItemStatus() {
    const item = filteredItems[selectedIndex];
    if (!item || item.is_log) return;

    const newStatus = item.status === 'completed' ? 'active' : 'completed';
    try {
      const res = await fetch(`/api/notes/${item.id}`, {
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
      const res = await fetch(`/api/notes/${item.id}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: 'deleted from web UI' })
      });
      if (res.ok) {
        showToast(`Note soft-deleted`);
        await loadData();
      }
    } catch (err) {
      showToast(`Delete failed: ${err.message}`);
    }
  }

  // Modal Handlers
  function openNoteModal(editItem = null) {
    noteModal.style.display = 'flex';
    if (editItem) {
      const flesh = (currentNoteDetail && (currentNoteDetail.id === editItem.id || currentNoteDetail.id.startsWith(editItem.id))) ? 
        (currentNoteDetail.note_flesh || editItem.flesh || '') : (editItem.flesh || '');
      modalTitle.textContent = 'Edit Note';
      document.getElementById('form-note-id').value = editItem.id;
      document.getElementById('form-title').value = editItem.display;
      document.getElementById('form-type').value = editItem.type || 'note';
      document.getElementById('form-status').value = editItem.status || 'active';
      document.getElementById('form-area').value = editItem.area || '';
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
        const res = await fetch(`/api/notes/${noteId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        if (!res.ok) throw new Error('Failed to update note');
        showToast('Note updated successfully');
      } else {
        // Create new
        const res = await fetch('/api/notes', {
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

  function showToast(message) {
    toast.textContent = message;
    toast.style.display = 'block';
    setTimeout(() => {
      toast.style.display = 'none';
    }, 2500);
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
