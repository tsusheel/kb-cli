/**
 * KB-CLI Web Fuzzy Finder — Finder UI & Navigation Controller
 * Manages dual-pane list/preview layout, fuzzy search filtering,
 * keyboard shortcuts, preview rendering, and item mutations.
 */

import { state } from './state.js';
import { dom } from './dom.js';
import {
  escapeHtml,
  showToast,
  copyTextFallback,
  triggerCopyFeedback,
  fuzzyMatch,
  highlightPositions,
  scrollPreviewToTop,
} from './utils.js';
import {
  fetchItems,
  fetchTags,
  fetchStats,
  fetchNoteDetail,
  updateNoteStatus,
  deleteNote,
  triggerSync,
} from './api.js';
import { renderMarkdown, applyPostMarkdownEnhancements } from './markdown.js';
import { openNoteModal, closeNoteModal } from './modal.js';

export function showMobileDetail() {
  if (window.innerWidth <= 768 && dom.appContainer) {
    dom.appContainer.classList.remove('mobile-search-open');
    dom.appContainer.classList.add('mobile-show-preview');
    scrollPreviewToTop();
  }
}

export function showMobileList() {
  if (dom.appContainer) {
    dom.appContainer.classList.remove('mobile-show-preview');
    if (!dom.mobileSearchInput || !dom.mobileSearchInput.value.trim()) {
      dom.appContainer.classList.remove('mobile-search-open');
    }
  }
}

export async function loadData() {
  try {
    const includeDeleted = state.activeTypeFilter === 'archived';
    state.allItems = await fetchItems(includeDeleted);
    updateCounts();
    filterAndRender();
  } catch (err) {
    console.error(err);
    if (dom.resultsList) {
      dom.resultsList.innerHTML = `<div class="empty-list">Error loading data: ${escapeHtml(err.message)}</div>`;
    }
  }
}

export async function loadTags() {
  try {
    const tags = await fetchTags();
    if (!dom.tagPills) return;
    if (!tags || tags.length === 0) {
      dom.tagPills.innerHTML = '';
      return;
    }
    dom.tagPills.innerHTML = tags
      .map(
        (t) => `
        <button class="tag-chip ${state.activeTagFilter === t.name ? 'active' : ''}" data-tag="${escapeHtml(t.name)}">
          #${escapeHtml(t.name)} (${t.count})
        </button>
      `
      )
      .join('');
  } catch (err) {
    console.error(err);
  }
}

export async function loadStats() {
  try {
    const stats = await fetchStats();
    if (stats && dom.footerStats) {
      const streak = stats.consecutive_streak_days || 0;
      dom.footerStats.textContent = `${stats.total_active_notes || 0} notes · ${stats.total_daily_logs || 0} logs · ${streak}d streak`;
    }
  } catch (err) {
    console.error(err);
  }
}

export function updateCounts() {
  const counts = { all: 0, note: 0, todo: 0, project: 0, idea: 0, log: 0, archived: 0 };
  state.allItems.forEach((item) => {
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

  const setElText = (id, val) => {
    const el = document.getElementById(id);
    if (el) el.textContent = val;
  };

  setElText('count-all', counts.all);
  setElText('count-note', counts.note);
  setElText('count-todo', counts.todo);
  setElText('count-project', counts.project);
  setElText('count-idea', counts.idea);
  setElText('count-log', counts.log);
  setElText('count-archived', counts.archived);
}

export function filterAndRender() {
  const query = dom.searchInput ? dom.searchInput.value.trim() : '';

  // 1. Filter by active tab & tag
  let items = state.allItems.filter((item) => {
    if (state.activeTypeFilter === 'archived') {
      return !!item.deleted_at;
    }
    if (item.deleted_at) return false;

    if (state.activeTypeFilter === 'all') {
      // include all active notes + logs
    } else if (state.activeTypeFilter === 'log') {
      if (!item.is_log) return false;
    } else {
      if (item.is_log || (item.type || '').toLowerCase() !== state.activeTypeFilter) {
        return false;
      }
    }

    if (state.activeTagFilter) {
      if (!item.tags || !item.tags.includes(state.activeTagFilter)) return false;
    }

    return true;
  });

  // 2. Fuzzy rank if query exists
  if (query) {
    const scored = [];
    items.forEach((item) => {
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
    state.filteredItems = scored.map((s) => s.item);
  } else {
    state.filteredItems = items;
  }

  if (dom.searchCount) {
    dom.searchCount.textContent = `${state.filteredItems.length} item${state.filteredItems.length === 1 ? '' : 's'}`;
  }
  renderList(query);

  if (state.filteredItems.length > 0) {
    state.selectedIndex = Math.min(state.selectedIndex, state.filteredItems.length - 1);
    selectItem(state.selectedIndex, false);
  } else {
    renderEmptyPreview();
  }
}

export function renderList(query) {
  if (!dom.resultsList) return;

  if (state.filteredItems.length === 0) {
    dom.resultsList.innerHTML = `<div class="empty-list">No matching items found</div>`;
    return;
  }

  dom.resultsList.innerHTML = state.filteredItems
    .map((item, idx) => {
      const isSelected = idx === state.selectedIndex;
      const type = item.is_log ? 'log' : item.type || 'note';
      const badgeClass = `badge-${type.toLowerCase()}`;

      let highlightedTitle = escapeHtml(item.display);
      if (query) {
        const match = fuzzyMatch(query, item.display);
        if (match && match.positions.length > 0) {
          highlightedTitle = highlightPositions(item.display, match.positions);
        }
      }

      const tagsHtml = (item.tags || [])
        .map((t) => `<span class="mini-tag">#${escapeHtml(t)}</span>`)
        .join(' ');

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
    })
    .join('');

  // Ensure selected item is scrolled into view
  const selectedEl = dom.resultsList.querySelector('.list-item.selected');
  if (selectedEl) {
    selectedEl.scrollIntoView({ block: 'nearest' });
  }
}

export async function selectItem(index, scroll = true) {
  state.selectedIndex = index;
  if (!dom.resultsList) return;

  const items = dom.resultsList.querySelectorAll('.list-item');
  items.forEach((el, i) => {
    el.classList.toggle('selected', i === state.selectedIndex);
  });

  if (scroll && items[state.selectedIndex]) {
    items[state.selectedIndex].scrollIntoView({ block: 'nearest' });
  }

  const item = state.filteredItems[state.selectedIndex];
  if (!item) {
    renderEmptyPreview();
    return;
  }

  scrollPreviewToTop();
  renderPreviewBasic(item);

  if (!item.is_log) {
    try {
      state.currentNoteDetail = await fetchNoteDetail(item.id);
      if (state.filteredItems[state.selectedIndex]?.id === item.id) {
        renderPreviewDetail(state.currentNoteDetail);
        scrollPreviewToTop();
      }
    } catch (err) {
      console.error('Error fetching note detail:', err);
    }
  }
}

export function toggleViewMode() {
  state.isRawView = !state.isRawView;
  updateViewModeDisplay();
}

export function updateViewModeDisplay() {
  if (dom.toggleViewBtn) {
    dom.toggleViewBtn.textContent = state.isRawView ? 'Preview' : 'Raw';
    dom.toggleViewBtn.classList.toggle('active', state.isRawView);
    dom.toggleViewBtn.title = state.isRawView
      ? 'Show Rendered Markdown (V)'
      : 'Show Raw Text (V)';
  }
  if (dom.previewMarkdownBody && dom.previewRawBody) {
    dom.previewMarkdownBody.style.display = state.isRawView ? 'none' : 'block';
    dom.previewRawBody.style.display = state.isRawView ? 'block' : 'none';
  }
}

export function renderPreviewBasic(item) {
  if (dom.previewEmpty) dom.previewEmpty.style.display = 'none';
  if (dom.previewContent) dom.previewContent.style.display = 'block';

  if (dom.previewTitle) dom.previewTitle.textContent = item.display;
  if (dom.previewShortId) {
    dom.previewShortId.textContent = item.short_id || item.id.substring(0, 7);
  }

  const type = item.is_log ? 'log' : item.type || 'note';
  if (dom.previewTypeBadge) {
    dom.previewTypeBadge.textContent = type;
    dom.previewTypeBadge.className = `type-badge badge-${type.toLowerCase()}`;
  }

  if (dom.previewStatusBadge) {
    dom.previewStatusBadge.textContent = item.status || (item.is_log ? 'recorded' : 'active');
    dom.previewStatusBadge.className = `status-badge ${item.status === 'completed' ? 'status-completed' : ''}`;
  }

  if (dom.previewAreaBadge) {
    if (item.area) {
      dom.previewAreaBadge.style.display = 'inline-block';
      dom.previewAreaBadge.textContent = item.area;
    } else {
      dom.previewAreaBadge.style.display = 'none';
    }
  }

  if (dom.previewDueBadge) dom.previewDueBadge.style.display = 'none';
  if (dom.previewUpdatedTime) {
    dom.previewUpdatedTime.textContent = item.timestamp ? `Updated ${item.timestamp}` : '';
  }

  if (dom.previewTagsRow) {
    if (item.tags && item.tags.length > 0) {
      dom.previewTagsRow.innerHTML = item.tags
        .map((t) => `<span class="preview-tag-chip" data-tag="${escapeHtml(t)}">#${escapeHtml(t)}</span>`)
        .join('');
    } else {
      dom.previewTagsRow.innerHTML = '';
    }
  }

  const rawContent = (item.flesh || item.display || '').trim();
  if (dom.previewRawBody) dom.previewRawBody.textContent = rawContent;
  if (dom.previewMarkdownBody) {
    dom.previewMarkdownBody.innerHTML = renderMarkdown(rawContent);
    applyPostMarkdownEnhancements(dom.previewMarkdownBody);
  }
  scrollPreviewToTop();
  updateViewModeDisplay();

  if (dom.previewGraphSection) dom.previewGraphSection.style.display = 'none';
  if (dom.previewHistorySection) dom.previewHistorySection.style.display = 'none';

  if (dom.toggleStatusBtn) {
    dom.toggleStatusBtn.textContent = item.status === 'completed' ? 'Reopen' : 'Done';
    dom.toggleStatusBtn.style.display =
      item.type === 'todo' || item.type === 'project' ? 'inline-flex' : 'none';
  }
  if (dom.editNoteBtn) {
    dom.editNoteBtn.style.display = item.is_log ? 'none' : 'inline-flex';
  }
  if (dom.toggleViewBtn) {
    dom.toggleViewBtn.style.display = 'inline-flex';
  }
}

export function renderPreviewDetail(note) {
  if (dom.previewDueBadge) {
    if (note.target_date_time) {
      dom.previewDueBadge.style.display = 'inline-block';
      dom.previewDueBadge.textContent = `Due: ${note.target_date_time.split('T')[0]}`;
    } else {
      dom.previewDueBadge.style.display = 'none';
    }
  }

  if (dom.previewTagsRow && note.tags && note.tags.length > 0) {
    dom.previewTagsRow.innerHTML = note.tags
      .map((t) => `<span class="preview-tag-chip" data-tag="${escapeHtml(t)}">#${escapeHtml(t)}</span>`)
      .join('');
  }

  const rawContent = (note.note_flesh || note.note || '').trim();
  if (dom.previewRawBody) dom.previewRawBody.textContent = rawContent;
  if (dom.previewMarkdownBody) {
    dom.previewMarkdownBody.innerHTML = renderMarkdown(rawContent);
    applyPostMarkdownEnhancements(dom.previewMarkdownBody);
  }
  scrollPreviewToTop();

  // Graph Links
  if (dom.previewGraphSection && dom.previewLinksList) {
    if (note.links && note.links.length > 0) {
      dom.previewGraphSection.style.display = 'block';
      dom.previewLinksList.innerHTML = note.links
        .map(
          (l) => `
          <div class="link-row" data-note-id="${escapeHtml(l.other_note_id)}">
            <span class="link-arrow">${l.direction === 'outgoing' ? '→' : '←'}</span>
            <span class="link-type">[${escapeHtml(l.type)}]</span>
            <span class="link-title">${escapeHtml(l.other_note || l.other_note_id.substring(0, 7))}</span>
          </div>
        `
        )
        .join('');

      dom.previewLinksList.querySelectorAll('.link-row').forEach((el) => {
        el.addEventListener('click', () => {
          const targetId = el.dataset.noteId;
          const idx = state.filteredItems.findIndex(
            (i) => i.id === targetId || i.id.startsWith(targetId)
          );
          if (idx !== -1) {
            selectItem(idx);
          } else if (dom.searchInput) {
            dom.searchInput.value = targetId.substring(0, 7);
            filterAndRender();
          }
        });
      });
    } else {
      dom.previewGraphSection.style.display = 'none';
    }
  }

  // Revision History
  if (dom.previewHistorySection && dom.previewHistoryList) {
    if (note.history && note.history.length > 0) {
      dom.previewHistorySection.style.display = 'block';
      dom.previewHistoryList.innerHTML = note.history
        .map(
          (h) => `
          <div class="history-item">
            <div>
              <span class="history-action">${escapeHtml(h.action)}</span>
              <span class="history-summary">${escapeHtml(h.changes_summary || '')}</span>
            </div>
            <span class="history-time">${escapeHtml(h.created_at ? h.created_at.split('T')[0] : '')}</span>
          </div>
        `
        )
        .join('');
    } else {
      dom.previewHistorySection.style.display = 'none';
    }
  }
}

export function renderEmptyPreview() {
  if (dom.previewEmpty) dom.previewEmpty.style.display = 'block';
  if (dom.previewContent) dom.previewContent.style.display = 'none';
}

export async function toggleCurrentItemStatus() {
  const item = state.filteredItems[state.selectedIndex];
  if (!item || item.is_log) return;

  const newStatus = item.status === 'completed' ? 'active' : 'completed';
  try {
    await updateNoteStatus(item.id, newStatus);
    item.status = newStatus;
    showToast(`Status updated to: ${newStatus}`);
    selectItem(state.selectedIndex, false);
  } catch (err) {
    showToast(`Failed to update status: ${err.message}`);
  }
}

export async function deleteCurrentItem() {
  const item = state.filteredItems[state.selectedIndex];
  if (!item) return;

  if (!confirm(`Are you sure you want to soft-delete "${item.display}"?`)) return;

  try {
    await deleteNote(item.id);
    showToast('Note soft-deleted');
    showMobileList();
    await loadData();
  } catch (err) {
    showToast(`Delete failed: ${err.message}`);
  }
}

export function handleGlobalKeydown(e) {
  const isModalOpen = dom.noteModal && dom.noteModal.style.display !== 'none';
  const isInputFocused =
    document.activeElement === dom.searchInput ||
    document.activeElement.tagName === 'INPUT' ||
    document.activeElement.tagName === 'TEXTAREA' ||
    document.activeElement.tagName === 'SELECT';

  if (e.key === 'Escape') {
    if (isModalOpen) {
      closeNoteModal();
      e.preventDefault();
    } else if (dom.appContainer && dom.appContainer.classList.contains('mobile-search-open')) {
      if (dom.mobileSearchInput) dom.mobileSearchInput.value = '';
      if (state.currentViewMode === 'docs') {
        if (dom.docsSearchInput) {
          dom.docsSearchInput.value = '';
          dom.docsSearchInput.dispatchEvent(new Event('input', { bubbles: true }));
        }
      } else {
        if (dom.searchInput) {
          dom.searchInput.value = '';
          dom.searchInput.dispatchEvent(new Event('input', { bubbles: true }));
        }
      }
      dom.appContainer.classList.remove('mobile-search-open');
      if (dom.mobileSearchClose) dom.mobileSearchClose.textContent = 'Done';
      e.preventDefault();
    } else if (dom.appContainer && dom.appContainer.classList.contains('mobile-show-preview')) {
      showMobileList();
      e.preventDefault();
    } else if (dom.searchInput && dom.searchInput.value) {
      dom.searchInput.value = '';
      if (dom.clearSearchBtn) dom.clearSearchBtn.style.display = 'none';
      filterAndRender();
      e.preventDefault();
    }
    return;
  }

  if (isModalOpen) return;

  // Focus search on '/' or 'Ctrl+K' / 'Cmd+K'
  if (
    (e.key === '/' && !isInputFocused) ||
    ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k')
  ) {
    e.preventDefault();
    if (dom.searchInput) {
      dom.searchInput.focus();
      dom.searchInput.select();
    }
    return;
  }

  // Navigation (Arrow keys or j/k or Ctrl+N/Ctrl+P)
  if (
    e.key === 'ArrowDown' ||
    (isInputFocused && e.ctrlKey && e.key === 'n') ||
    (!isInputFocused && e.key === 'j')
  ) {
    e.preventDefault();
    if (state.filteredItems.length > 0) {
      selectItem(Math.min(state.selectedIndex + 1, state.filteredItems.length - 1));
    }
    return;
  }

  if (
    e.key === 'ArrowUp' ||
    (isInputFocused && e.ctrlKey && e.key === 'p') ||
    (!isInputFocused && e.key === 'k')
  ) {
    e.preventDefault();
    if (state.filteredItems.length > 0) {
      selectItem(Math.max(state.selectedIndex - 1, 0));
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
      const item = state.filteredItems[state.selectedIndex];
      if (item) {
        navigator.clipboard.writeText(item.id);
        showToast(`Copied ID: ${item.short_id || item.id.substring(0, 7)}`);
      }
    } else if (e.key === 'm' || e.key === 'M') {
      e.preventDefault();
      const item = state.filteredItems[state.selectedIndex];
      if (item) {
        const md = `# ${item.display}\n\n${item.flesh || ''}`;
        navigator.clipboard.writeText(md);
        showToast('Copied Note Markdown');
      }
    } else if (e.key === 'e' || e.key === 'E') {
      e.preventDefault();
      const item = state.filteredItems[state.selectedIndex];
      if (item && !item.is_log) {
        openNoteModal(item);
      }
    }
  }
}

/**
 * Initializes all search, list, filter, button, and shortcut listeners.
 */
export function setupFinderListeners() {
  if (dom.mobileBackBtn) {
    dom.mobileBackBtn.addEventListener('click', () => {
      showMobileList();
    });
  }

  // Mobile Bottom Search Handlers
  function openMobileSearch(e) {
    if (e && e.type === 'touchend') {
      e.preventDefault();
    }
    if (!dom.appContainer || !dom.mobileSearchInput) return;
    dom.appContainer.classList.add('mobile-search-open');
    const isDocs = state.currentViewMode === 'docs';
    dom.mobileSearchInput.placeholder = isDocs ? 'Search documentation...' : 'Search notes & tasks...';
    dom.mobileSearchInput.value = isDocs ? (dom.docsSearchInput?.value || '') : (dom.searchInput?.value || '');
    if (dom.mobileSearchClose) {
      dom.mobileSearchClose.textContent = 'Done';
    }
    // Synchronous focus directly in user gesture stack for iPhone / iOS Safari
    dom.mobileSearchInput.focus();
    if (typeof dom.mobileSearchInput.setSelectionRange === 'function') {
      const len = dom.mobileSearchInput.value.length;
      dom.mobileSearchInput.setSelectionRange(len, len);
    }
  }

  function closeMobileSearch() {
    if (!dom.appContainer) return;
    dom.appContainer.classList.remove('mobile-search-open');
    if (dom.mobileSearchClose) {
      dom.mobileSearchClose.textContent = 'Done';
    }
  }

  function clearMobileSearch() {
    if (dom.mobileSearchInput) {
      dom.mobileSearchInput.value = '';
    }
    if (state.currentViewMode === 'docs') {
      if (dom.docsSearchInput) {
        dom.docsSearchInput.value = '';
        dom.docsSearchInput.dispatchEvent(new Event('input', { bubbles: true }));
      }
    } else {
      if (dom.searchInput) {
        dom.searchInput.value = '';
        dom.searchInput.dispatchEvent(new Event('input', { bubbles: true }));
      }
    }
    closeMobileSearch();
  }

  if (dom.mobileSearchBtn) {
    dom.mobileSearchBtn.addEventListener('click', openMobileSearch);
    dom.mobileSearchBtn.addEventListener('touchend', openMobileSearch, { passive: false });
  }

  if (dom.mobileSearchClose) {
    dom.mobileSearchClose.addEventListener('click', () => {
      const btnText = (dom.mobileSearchClose.textContent || '').trim();
      if (btnText === 'Done') {
        // Blur to dismiss keyboard
        if (dom.mobileSearchInput) {
          dom.mobileSearchInput.blur();
        }
        // If there is query text, keep search dock open at bottom and switch to 'Clear'
        if (dom.mobileSearchInput && dom.mobileSearchInput.value.trim()) {
          dom.mobileSearchClose.textContent = 'Clear';
        } else {
          // If empty, close search dock and restore floating search FAB
          closeMobileSearch();
        }
      } else {
        // Button says 'Clear': wipe input, reset search, close dock, restore search FAB
        clearMobileSearch();
      }
    });
  }

  if (dom.mobileSearchInput) {
    dom.mobileSearchInput.addEventListener('focus', () => {
      if (dom.mobileSearchClose) {
        dom.mobileSearchClose.textContent = 'Done';
      }
    });

    dom.mobileSearchInput.addEventListener('blur', () => {
      if (dom.mobileSearchClose) {
        dom.mobileSearchClose.textContent = dom.mobileSearchInput.value.trim() ? 'Clear' : 'Done';
      }
    });

    dom.mobileSearchInput.addEventListener('input', () => {
      const val = dom.mobileSearchInput.value;
      if (dom.mobileSearchClose) {
        if (document.activeElement === dom.mobileSearchInput) {
          dom.mobileSearchClose.textContent = 'Done';
        } else {
          dom.mobileSearchClose.textContent = val.trim() ? 'Clear' : 'Done';
        }
      }
      if (state.currentViewMode === 'docs') {
        if (dom.docsSearchInput) {
          dom.docsSearchInput.value = val;
          dom.docsSearchInput.dispatchEvent(new Event('input', { bubbles: true }));
        }
      } else {
        if (dom.searchInput) {
          dom.searchInput.value = val;
          dom.searchInput.dispatchEvent(new Event('input', { bubbles: true }));
        }
      }
    });
  }

  if (dom.searchInput) {
    dom.searchInput.addEventListener('input', () => {
      if (dom.clearSearchBtn) {
        dom.clearSearchBtn.style.display = dom.searchInput.value ? 'block' : 'none';
      }
      filterAndRender();
    });
  }

  if (dom.clearSearchBtn) {
    dom.clearSearchBtn.addEventListener('click', () => {
      if (dom.searchInput) {
        dom.searchInput.value = '';
        dom.clearSearchBtn.style.display = 'none';
        dom.searchInput.focus();
      }
      filterAndRender();
    });
  }

  if (dom.typeFilterPills) {
    dom.typeFilterPills.addEventListener('click', (e) => {
      const pill = e.target.closest('.pill');
      if (!pill) return;
      document
        .querySelectorAll('#type-filter-pills .pill')
        .forEach((p) => p.classList.remove('active'));
      pill.classList.add('active');
      state.activeTypeFilter = pill.dataset.filter;
      filterAndRender();
    });
  }

  if (dom.tagPills) {
    dom.tagPills.addEventListener('click', (e) => {
      const chip = e.target.closest('.tag-chip');
      if (!chip) return;
      const tag = chip.dataset.tag;
      if (state.activeTagFilter === tag) {
        state.activeTagFilter = null;
        chip.classList.remove('active');
      } else {
        document.querySelectorAll('.tag-chip').forEach((c) => c.classList.remove('active'));
        state.activeTagFilter = tag;
        chip.classList.add('active');
      }
      filterAndRender();
    });
  }

  if (dom.resultsList) {
    dom.resultsList.addEventListener('click', (e) => {
      const itemEl = e.target.closest('.list-item');
      if (!itemEl) return;
      const index = parseInt(itemEl.dataset.index, 10);
      if (!isNaN(index)) {
        selectItem(index);
        showMobileDetail();
      }
    });
  }

  // Keyboard Navigation
  document.addEventListener('keydown', handleGlobalKeydown);

  // New Note Header Button
  if (dom.newNoteBtn) {
    dom.newNoteBtn.addEventListener('click', () => openNoteModal());
  }

  // Database Sync Button
  if (dom.syncBtn) {
    dom.syncBtn.addEventListener('click', async () => {
      dom.syncBtn.classList.add('syncing');
      dom.syncBtn.textContent = 'Syncing...';
      showToast('Syncing data...');
      try {
        const data = await triggerSync();
        if (data.success) {
          showToast(data.message || 'Sync complete');
        } else {
          showToast(`Sync failed: ${data.error || 'unknown error'}`);
        }
      } catch (err) {
        showToast(`Sync error: ${err.message}`);
      } finally {
        dom.syncBtn.classList.remove('syncing');
        dom.syncBtn.textContent = 'Sync';
        await loadData();
        await loadTags();
        await loadStats();
      }
    });
  }

  // Preview Actions
  if (dom.toggleViewBtn) {
    dom.toggleViewBtn.addEventListener('click', () => toggleViewMode());
  }

  if (dom.copyIdBtn) {
    dom.copyIdBtn.addEventListener('click', () => {
      const item = state.filteredItems[state.selectedIndex];
      if (item) {
        navigator.clipboard.writeText(item.id);
        showToast(`Copied ID: ${item.short_id || item.id.substring(0, 7)}`);
      }
    });
  }

  if (dom.toggleStatusBtn) {
    dom.toggleStatusBtn.addEventListener('click', () => toggleCurrentItemStatus());
  }

  if (dom.editNoteBtn) {
    dom.editNoteBtn.addEventListener('click', () => {
      const item = state.filteredItems[state.selectedIndex];
      if (item && !item.is_log) {
        openNoteModal(item);
      }
    });
  }

  if (dom.deleteNoteBtn) {
    dom.deleteNoteBtn.addEventListener('click', () => deleteCurrentItem());
  }

  if (dom.toggleHistoryBtn && dom.previewHistoryList) {
    dom.toggleHistoryBtn.addEventListener('click', () => {
      const isVisible = dom.previewHistoryList.style.display !== 'none';
      dom.previewHistoryList.style.display = isVisible ? 'none' : 'flex';
      dom.toggleHistoryBtn.textContent = isVisible ? 'Show' : 'Hide';
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
