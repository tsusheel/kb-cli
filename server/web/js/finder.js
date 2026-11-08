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
  deleteDoc,
  fetchDocContent,
  triggerSync,
} from './api.js';
import { renderMarkdown, applyPostMarkdownEnhancements } from './markdown.js';
import { openNoteModal, closeNoteModal } from './modal.js';
import { handleDocsSearch, openEditDocModal, renderDocsTree } from './docs.js';
import { searchRegistry } from './search-registry.js';
import { searchTokens } from './search-tokens.js';

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

export async function loadTags(scope = null) {
  try {
    const effectiveScope = scope || (state.currentViewMode === 'docs' ? 'docs' : 'notes');
    const tags = await fetchTags(effectiveScope);
    if (!dom.tagPills) return;
    if (!tags || tags.length === 0) {
      dom.tagPills.innerHTML = '';
      return;
    }
    const currentActiveTag = state.currentViewMode === 'docs' ? state.activeDocTagFilter : state.activeTagFilter;
    dom.tagPills.innerHTML = tags
      .map(
        (t) => `
        <button class="tag-chip ${currentActiveTag === t.name ? 'active' : ''}" data-tag="${escapeHtml(t.name)}">
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
  const counts = { all: 0, note: 0, todo: 0, project: 0, idea: 0, log: 0, doc: 0, archived: 0 };
  state.allItems.forEach((item) => {
    if (item.deleted_at) {
      counts.archived++;
    } else {
      counts.all++;
      if (item.is_log) {
        counts.log++;
      } else if (item.type === 'doc' || item.is_doc) {
        counts.doc++;
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
  setElText('count-doc', counts.doc);
  setElText('count-archived', counts.archived);
}

export function appendTagToSearch(tag) {
  searchTokens.toggleTag(tag);
}

export function filterAndRender() {
  const query = searchTokens.getFullQueryString();
  const parsed = searchRegistry.parseQuery(query);

  // 1. Structured filtering with tab & tag support
  let items = state.allItems.filter((item) => {
    // If query specifies archived status, honor query
    if (parsed.statuses.includes('archived')) {
      if (!item.deleted_at) return false;
    } else if (state.activeTypeFilter === 'archived') {
      return !!item.deleted_at;
    } else {
      if (item.deleted_at) return false;
    }

    // Active type filter pill (unless query explicitly specifies @types)
    if (parsed.types.length === 0) {
      if (state.activeTypeFilter === 'all') {
        // all active notes + logs
      } else if (state.activeTypeFilter === 'log') {
        if (!item.is_log) return false;
      } else if (state.activeTypeFilter !== 'archived') {
        if (item.is_log || (item.type || '').toLowerCase() !== state.activeTypeFilter) {
          return false;
        }
      }
    }

    // Active tag filter pill (unless query explicitly specifies #tags)
    if (state.activeTagFilter && parsed.tags.length === 0) {
      if (!item.tags || !item.tags.includes(state.activeTagFilter)) return false;
    }

    // 2. Structured query evaluation (#tags, @types, is:status, +area, plugin qualifiers)
    return searchRegistry.filterItem(item, parsed);
  });

  // 3. Fuzzy rank if freeform clean text exists
  const textQuery = parsed.cleanText;
  if (textQuery) {
    const scored = [];
    items.forEach((item) => {
      const titleMatch = fuzzyMatch(textQuery, item.display);
      const fleshMatch = fuzzyMatch(textQuery, item.flesh);
      const tagMatch = item.tags ? fuzzyMatch(textQuery, item.tags.join(' ')) : null;

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
  renderList(textQuery || query);

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
    if (item.type === 'doc' || item.is_doc) {
      try {
        const docData = await fetchDocContent(item.doc_path || item.id);
        if (state.filteredItems[state.selectedIndex]?.id === item.id) {
          state.currentDocContent = docData.content;
          state.currentDocTags = docData.tags || [];
          if (dom.previewRawBody) dom.previewRawBody.textContent = docData.content;
          if (dom.previewMarkdownBody) {
            dom.previewMarkdownBody.innerHTML = renderMarkdown(docData.content);
            applyPostMarkdownEnhancements(dom.previewMarkdownBody);
          }
          if (dom.previewTagsRow) {
            if (docData.tags && docData.tags.length > 0) {
              dom.previewTagsRow.innerHTML = docData.tags
                .map((t) => `<span class="preview-tag-chip" data-tag="${escapeHtml(t)}">#${escapeHtml(t)}</span>`)
                .join('');
              dom.previewTagsRow.style.display = 'flex';
            } else {
              dom.previewTagsRow.innerHTML = '';
            }
          }
          scrollPreviewToTop();
        }
      } catch (err) {
        console.error('Error fetching doc content in notes preview:', err);
      }
    } else {
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
}

export function canNavigateNotes(direction) {
  if (!state.filteredItems || state.filteredItems.length === 0) return false;
  const targetIndex = state.selectedIndex + direction;
  return targetIndex >= 0 && targetIndex < state.filteredItems.length;
}

export async function navigateNotes(direction) {
  if (!canNavigateNotes(direction)) return false;
  const targetIndex = state.selectedIndex + direction;
  await selectItem(targetIndex, true);
  return true;
}

export function getAdjacentNoteTitle(direction) {
  if (!state.filteredItems) return null;
  const targetIndex = state.selectedIndex + direction;
  if (targetIndex >= 0 && targetIndex < state.filteredItems.length) {
    const item = state.filteredItems[targetIndex];
    return item ? item.display : null;
  }
  return null;
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

  const isDoc = item.type === 'doc' || item.is_doc;
  if (dom.previewShortId) {
    dom.previewShortId.textContent = isDoc ? 'DOC' : (item.short_id || item.id.substring(0, 7));
  }

  const type = item.is_log ? 'log' : item.type || 'note';
  if (dom.previewTypeBadge) {
    dom.previewTypeBadge.textContent = type;
    dom.previewTypeBadge.className = `type-badge badge-${type.toLowerCase()}`;
  }

  if (dom.previewStatusBadge) {
    dom.previewStatusBadge.textContent = item.status || (item.is_log ? 'recorded' : 'active');
    dom.previewStatusBadge.className = `status-badge ${item.status === 'completed' ? 'status-completed' : ''}`;
    dom.previewStatusBadge.style.display = isDoc ? 'none' : 'inline-block';
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
      !isDoc && (item.type === 'todo' || item.type === 'project') ? 'inline-flex' : 'none';
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

  if (item.type === 'doc' || item.is_doc) {
    if (!confirm(`Are you sure you want to delete doc "${item.display || item.doc_path}"?`)) return;
    try {
      await deleteDoc(item.doc_path || item.id);
      showToast('Document deleted');
      showMobileList();
      await loadData();
      await loadTags();
    } catch (err) {
      showToast(`Delete failed: ${err.message}`);
    }
    return;
  }

  if (!confirm(`Are you sure you want to soft-delete "${item.display}"?`)) return;

  try {
    await deleteNote(item.id);
    showToast('Note soft-deleted');
    showMobileList();
    await loadData();
    await loadTags();
  } catch (err) {
    showToast(`Delete failed: ${err.message}`);
  }
}

export function handleGlobalKeydown(e) {
  // Centralized key navigation is now managed by keybindingRegistry
}

// Mobile Bottom Search Handlers & State
let isMobileSearchFocused = false;

export function setMobileSearchButtonState(mode) {
  if (!dom.mobileSearchClose) return;
  if (mode === 'clear') {
    dom.mobileSearchClose.textContent = 'Clear';
    dom.mobileSearchClose.classList.add('is-clear');
  } else {
    dom.mobileSearchClose.textContent = 'Done';
    dom.mobileSearchClose.classList.remove('is-clear');
  }
}

/**
 * Dynamically pins the mobile search dock directly above the software keyboard
 * using the modern window.visualViewport API across iOS Safari & Android Chrome.
 */
export function updateMobileSearchPosition() {
  if (!dom.mobileBottomSearch || !dom.appContainer?.classList.contains('mobile-search-open')) {
    return;
  }

  if (window.visualViewport) {
    const vv = window.visualViewport;
    // Calculate how much the keyboard / visual viewport offset covers from the bottom of the screen.
    // In CSS layout, fixed bottom: 0 is at window.innerHeight.
    // The bottom edge of the visual viewport is at (vv.offsetTop + vv.height).
    const offsetFromBottom = Math.max(
      0,
      Math.round(window.innerHeight - (vv.offsetTop + vv.height))
    );

    dom.mobileBottomSearch.style.bottom = `${offsetFromBottom}px`;

    // When lifted above the keyboard, safe area padding is covered by the keyboard,
    // so we use a clean compact 10px padding.
    if (offsetFromBottom > 10) {
      dom.mobileBottomSearch.style.paddingBottom = '10px';
    } else {
      dom.mobileBottomSearch.style.paddingBottom = '';
    }

    // Ensure list content can be scrolled comfortably above the search bar
    const dockHeight = dom.mobileBottomSearch.offsetHeight || 62;
    const totalBottomPadding = `${offsetFromBottom + dockHeight + 10}px`;
    if (dom.resultsList) {
      dom.resultsList.style.paddingBottom = totalBottomPadding;
    }
    if (dom.docsTree) {
      dom.docsTree.style.paddingBottom = totalBottomPadding;
    }
  } else {
    dom.mobileBottomSearch.style.bottom = '0px';
  }
}

export function resetMobileSearchPosition() {
  if (dom.mobileBottomSearch) {
    dom.mobileBottomSearch.style.bottom = '';
    dom.mobileBottomSearch.style.paddingBottom = '';
  }
  if (dom.resultsList) {
    dom.resultsList.style.paddingBottom = '';
  }
  if (dom.docsTree) {
    dom.docsTree.style.paddingBottom = '';
  }
}

export function resetMobileSearch() {
  if (dom.mobileSearchInput) {
    dom.mobileSearchInput.value = '';
    if (typeof dom.mobileSearchInput.blur === 'function') {
      dom.mobileSearchInput.blur();
    }
  }
  setMobileSearchButtonState('done');
  if (dom.appContainer) {
    dom.appContainer.classList.remove('mobile-search-open');
  }
  resetMobileSearchPosition();
  isMobileSearchFocused = false;
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

  function openMobileSearch(e) {
    if (e && e.type === 'touchend') {
      e.preventDefault();
    }
    if (!dom.appContainer || !dom.mobileSearchInput) return;
    dom.appContainer.classList.add('mobile-search-open');
    const isDocs = state.currentViewMode === 'docs';
    dom.mobileSearchInput.placeholder = isDocs ? 'Search documentation...' : 'Search notes & tasks...';
    dom.mobileSearchInput.value = dom.searchInput?.value || '';
    isMobileSearchFocused = true;
    setMobileSearchButtonState('done');

    updateMobileSearchPosition();

    // Focus input directly with preventScroll to stop iOS from jumping
    try {
      dom.mobileSearchInput.focus({ preventScroll: true });
    } catch {
      dom.mobileSearchInput.focus();
    }

    if (typeof dom.mobileSearchInput.setSelectionRange === 'function') {
      const len = dom.mobileSearchInput.value.length;
      dom.mobileSearchInput.setSelectionRange(len, len);
    }

    // Schedule position updates as keyboard opens and finishes animation
    requestAnimationFrame(updateMobileSearchPosition);
    setTimeout(updateMobileSearchPosition, 50);
    setTimeout(updateMobileSearchPosition, 150);
    setTimeout(updateMobileSearchPosition, 300);
    setTimeout(updateMobileSearchPosition, 500);
  }

  function closeMobileSearch() {
    if (!dom.appContainer) return;
    dom.appContainer.classList.remove('mobile-search-open');
    isMobileSearchFocused = false;
    setMobileSearchButtonState('done');
    resetMobileSearchPosition();
  }

  function clearMobileSearch() {
    if (dom.mobileSearchInput) {
      dom.mobileSearchInput.value = '';
    }
    if (dom.searchInput) {
      dom.searchInput.value = '';
      dom.searchInput.dispatchEvent(new Event('input', { bubbles: true }));
    }
    closeMobileSearch();
  }

  // Dynamic visualViewport tracking for software keyboard repositioning
  if (window.visualViewport) {
    window.visualViewport.addEventListener('resize', updateMobileSearchPosition);
    window.visualViewport.addEventListener('scroll', updateMobileSearchPosition);
  }
  window.addEventListener('resize', updateMobileSearchPosition);

  if (dom.mobileSearchBtn) {
    dom.mobileSearchBtn.addEventListener('click', openMobileSearch);
    dom.mobileSearchBtn.addEventListener('touchend', openMobileSearch, { passive: false });
  }

  if (dom.mobileSearchClose) {
    // Prevent input blur before click handler processes state
    const preventPrematureBlur = (e) => e.preventDefault();
    dom.mobileSearchClose.addEventListener('pointerdown', preventPrematureBlur);
    dom.mobileSearchClose.addEventListener('mousedown', preventPrematureBlur);

    dom.mobileSearchClose.addEventListener('click', () => {
      const hasText = Boolean(dom.mobileSearchInput && dom.mobileSearchInput.value.trim());

      if (isMobileSearchFocused) {
        // User tapped "Done" while input was active
        if (dom.mobileSearchInput) {
          dom.mobileSearchInput.blur();
        }
        isMobileSearchFocused = false;

        if (hasText) {
          // Keep search dock open at bottom showing query; switch button to 'Clear' with distinct color
          setMobileSearchButtonState('clear');
        } else {
          // Empty input: close search dock and restore floating search FAB
          closeMobileSearch();
        }
      } else {
        // User tapped "Clear" while input was blurred (or empty Done)
        if (hasText) {
          clearMobileSearch();
        } else {
          closeMobileSearch();
        }
      }
    });
  }

  if (dom.mobileSearchInput) {
    dom.mobileSearchInput.addEventListener('focus', () => {
      isMobileSearchFocused = true;
      setMobileSearchButtonState('done');
      updateMobileSearchPosition();
      setTimeout(updateMobileSearchPosition, 100);
      setTimeout(updateMobileSearchPosition, 300);
    });

    dom.mobileSearchInput.addEventListener('blur', () => {
      isMobileSearchFocused = false;
      setMobileSearchButtonState(dom.mobileSearchInput.value.trim() ? 'clear' : 'done');
      updateMobileSearchPosition();
      setTimeout(updateMobileSearchPosition, 100);
      setTimeout(updateMobileSearchPosition, 300);
    });

    dom.mobileSearchInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        dom.mobileSearchInput.blur();
        isMobileSearchFocused = false;
        if (dom.mobileSearchInput.value.trim()) {
          setMobileSearchButtonState('clear');
        } else {
          closeMobileSearch();
        }
      }
    });

    dom.mobileSearchInput.addEventListener('input', () => {
      const val = dom.mobileSearchInput.value;
      if (isMobileSearchFocused) {
        setMobileSearchButtonState('done');
      } else {
        setMobileSearchButtonState(val.trim() ? 'clear' : 'done');
      }
      if (dom.searchInput) {
        dom.searchInput.value = val;
        dom.searchInput.dispatchEvent(new Event('input', { bubbles: true }));
      }
    });
  }

  if (dom.searchInput) {
    dom.searchInput.addEventListener('input', () => {
      if (dom.clearSearchBtn) {
        dom.clearSearchBtn.style.display = dom.searchInput.value ? 'block' : 'none';
      }
      if (state.currentViewMode === 'docs') {
        handleDocsSearch(searchTokens.getFullQueryString());
        return;
      }
      filterAndRender();
    });
  }

  if (dom.clearSearchBtn) {
    dom.clearSearchBtn.addEventListener('click', () => {
      searchTokens.clearTokens(false);
      if (dom.searchInput) {
        dom.searchInput.value = '';
        dom.clearSearchBtn.style.display = 'none';
        dom.searchInput.focus();
      }
      searchTokens.renderPills();
      if (state.currentViewMode === 'docs') {
        handleDocsSearch('');
        return;
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
      if (tag) {
        appendTagToSearch(tag);
      }
    });
  }

  if (dom.previewTagsRow) {
    dom.previewTagsRow.addEventListener('click', (e) => {
      const chip = e.target.closest('.preview-tag-chip');
      if (!chip) return;
      const tag = chip.dataset.tag;
      if (tag) {
        appendTagToSearch(tag);
      }
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

  // Keyboard Navigation is initialized by keybindingRegistry.init()

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
        if (item.type === 'doc' || item.is_doc) {
          openEditDocModal(item.doc_path || item.id);
        } else {
          openNoteModal(item);
        }
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
