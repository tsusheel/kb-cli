/**
 * KB-CLI Web Fuzzy Finder — Documentation Directory Tree & Viewer Controller
 * Manages the collapsible tree directory view, document selection,
 * markdown/mermaid rendering, filtering, arbitrary nested folders, drag-and-drop import,
 * in-browser document editing, and in-doc search.
 */

import { state } from './state.js';
import { dom } from './dom.js';
import { 
  fetchDocsTree, 
  fetchDocContent, 
  saveDocContent, 
  createDoc, 
  createDocFolder, 
  deleteDoc, 
  importDocsTree, 
  searchDocs 
} from './api.js';
import { renderMarkdown, applyPostMarkdownEnhancements } from './markdown.js';
import { showMobileDetail, showMobileList, updateViewModeDisplay, resetMobileSearch, filterAndRender, loadData, loadTags } from './finder.js';
import { escapeHtml, scrollPreviewToTop, showToast } from './utils.js';
import { searchRegistry } from './search-registry.js';

let isDocsInitialized = false;

/**
 * Initializes the documentation subsystem, loads the tree, and binds listeners.
 */
export async function initDocs() {
  if (isDocsInitialized) return;
  isDocsInitialized = true;

  // 1. Navigation Tab Listeners
  if (dom.tabNotes) {
    dom.tabNotes.addEventListener('click', () => switchViewMode('notes'));
  }
  if (dom.tabDocs) {
    dom.tabDocs.addEventListener('click', () => switchViewMode('docs'));
  }

  // 2. Action Buttons for Documents and Folders
  if (dom.newDocBtn) {
    dom.newDocBtn.addEventListener('click', () => openNewDocModal());
  }
  if (dom.newFolderBtn) {
    dom.newFolderBtn.addEventListener('click', () => openNewFolderModal());
  }
  if (dom.editDocBtn) {
    dom.editDocBtn.addEventListener('click', () => openEditDocModal());
  }
  if (dom.deleteDocBtn) {
    dom.deleteDocBtn.addEventListener('click', async () => {
      if (!state.selectedDocPath) return;
      if (confirm(`Are you sure you want to delete "${state.selectedDocPath}"?`)) {
        try {
          await deleteDoc(state.selectedDocPath);
          showToast(`Deleted "${state.selectedDocPath}"`);
          state.docsTree = await fetchDocsTree();
          state.selectedDocPath = null;
          renderDocsTree('');
          const firstPath = getFirstDocPath(state.docsTree);
          if (firstPath) {
            await selectDoc(firstPath);
          }
          await loadData();
          await loadTags(state.currentViewMode === 'docs' ? 'docs' : 'notes');
        } catch (err) {
          showToast(`Error: ${err.message}`);
        }
      }
    });
  }

  // 5. Doc Modal Tab Toggle (Write / Preview)
  if (dom.docModalTabWrite && dom.docModalTabPreview) {
    dom.docModalTabWrite.addEventListener('click', () => {
      dom.docModalTabWrite.classList.add('active');
      dom.docModalTabPreview.classList.remove('active');
      if (dom.formDocContent) dom.formDocContent.style.display = 'block';
      if (dom.docModalContentPreview) dom.docModalContentPreview.style.display = 'none';
    });

    dom.docModalTabPreview.addEventListener('click', () => {
      dom.docModalTabPreview.classList.add('active');
      dom.docModalTabWrite.classList.remove('active');
      if (dom.formDocContent) dom.formDocContent.style.display = 'none';
      if (dom.docModalContentPreview) {
        dom.docModalContentPreview.style.display = 'block';
        dom.docModalContentPreview.innerHTML = renderMarkdown(dom.formDocContent?.value || '');
      }
    });
  }

  // Close Doc Modal
  if (dom.docModalCloseBtn) {
    dom.docModalCloseBtn.addEventListener('click', closeDocModal);
  }
  if (dom.docModalCancelBtn) {
    dom.docModalCancelBtn.addEventListener('click', closeDocModal);
  }

  // Submit Doc Form
  if (dom.docForm) {
    dom.docForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const pathVal = (dom.formDocPath?.value || '').trim();
      const titleVal = (dom.formDocTitle?.value || '').trim();
      const contentVal = dom.formDocContent?.value || '';
      const tagsVal = (dom.formDocTags?.value || '')
        .split(',')
        .map(t => t.trim().replace(/^#/, ''))
        .filter(Boolean);

      if (!pathVal) {
        alert('Please enter a document path or filename');
        return;
      }

      try {
        if (state.editingDocPath) {
          await saveDocContent(pathVal, titleVal, contentVal, tagsVal);
          showToast(`Saved "${titleVal || pathVal}"`);
        } else {
          await createDoc({ path: pathVal, title: titleVal, content: contentVal, tags: tagsVal });
          showToast(`Created "${titleVal || pathVal}"`);
        }
        closeDocModal();
        state.docsTree = await fetchDocsTree();
        if (state.docsCache) {
          state.docsCache.delete(pathVal);
        }
        renderDocsTree(dom.searchInput?.value || '');
        await selectDoc(pathVal);
        await loadData();
        await loadTags(state.currentViewMode === 'docs' ? 'docs' : 'notes');
      } catch (err) {
        showToast(`Save failed: ${err.message}`);
      }
    });
  }

  // Close Folder Modal
  if (dom.folderModalCloseBtn) {
    dom.folderModalCloseBtn.addEventListener('click', closeFolderModal);
  }
  if (dom.folderModalCancelBtn) {
    dom.folderModalCancelBtn.addEventListener('click', closeFolderModal);
  }

  // Submit Folder Form
  if (dom.folderForm) {
    dom.folderForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const folderPath = (dom.formFolderPath?.value || '').trim();
      const folderTitle = (dom.formFolderTitle?.value || '').trim();
      if (!folderPath) {
        alert('Please enter a folder path');
        return;
      }
      try {
        await createDocFolder({ path: folderPath, title: folderTitle });
        closeFolderModal();
        showToast(`Created folder "${folderTitle || folderPath}"`);
        state.docsExpandedFolders.add(folderPath);
        state.docsTree = await fetchDocsTree();
        renderDocsTree('');
      } catch (err) {
        showToast(`Error: ${err.message}`);
      }
    });
  }

  // 6. Intra-document link interception in preview markdown body
  if (dom.previewMarkdownBody) {
    dom.previewMarkdownBody.addEventListener('click', (e) => {
      if (state.currentViewMode !== 'docs') return;
      const link = e.target.closest('a');
      if (!link) return;
      const href = link.getAttribute('href');
      if (!href) return;
      if (href.startsWith('http://') || href.startsWith('https://') || href.startsWith('#') || href.startsWith('mailto:')) {
        return;
      }
      if (href.endsWith('.md')) {
        e.preventDefault();
        let target = href.replace(/^\.\//, '');
        if (state.selectedDocPath && state.selectedDocPath.startsWith('proposals/')) {
          if (target.startsWith('../')) {
            target = target.replace(/^\.\.\//, '');
          } else if (!target.startsWith('proposals/')) {
            target = 'proposals/' + target;
          }
        }
        selectDoc(target, true);
      }
    });
  }

  // 7. Check for URL deep linking
  handleDocsRouting();
  window.addEventListener('popstate', handleDocsRouting);

  // 8. Initialize Drag-and-Drop Folder Importer
  initDragAndDrop();

  // 9. Pre-load documentation tree and prefetch all documents in background
  loadDocsTree();
}

/**
 * Handles browser history events and deep linking for /docs/...
 */
export function handleDocsRouting() {
  const path = window.location.pathname;
  if (path === '/docs' || path.startsWith('/docs/')) {
    const docPath = path.replace(/^\/docs\/?/, '');
    switchViewMode('docs', docPath || null);
  }
}

/**
 * Switches the main interface between Notes mode and Docs mode.
 */
export async function switchViewMode(mode, targetDocPath = null) {
  state.currentViewMode = mode;
  const isDocs = mode === 'docs';

  // Reset search state across both desktop and mobile
  resetMobileSearch();
  if (dom.searchInput) {
    dom.searchInput.value = '';
    dom.searchInput.placeholder = isDocs ? 'Search docs (/ to focus)...' : 'Search (/ to focus)';
    if (dom.clearSearchBtn) dom.clearSearchBtn.style.display = 'none';
  }

  // Toggle navigation tab active states
  if (dom.tabNotes) dom.tabNotes.classList.toggle('active', !isDocs);
  if (dom.tabDocs) dom.tabDocs.classList.toggle('active', isDocs);

  // Toggle Left Panes & Filter Bar (Filter Bar stays visible in both modes for tags)
  if (dom.resultsPane) dom.resultsPane.style.display = isDocs ? 'none' : 'flex';
  if (dom.filterBar) dom.filterBar.style.display = 'flex';
  if (dom.typeFilterPills) dom.typeFilterPills.style.display = isDocs ? 'none' : 'flex';
  if (dom.docsPane) dom.docsPane.style.display = isDocs ? 'flex' : 'none';

  // Toggle Header action buttons - ALWAYS show sync button and search bar!
  if (dom.syncBtn) dom.syncBtn.style.display = 'inline-flex';
  if (dom.searchInput) {
    dom.searchInput.parentElement.parentElement.style.display = 'flex';
  }
  if (dom.newNoteBtn) dom.newNoteBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.newDocBtn) dom.newDocBtn.style.display = isDocs ? 'inline-flex' : 'none';
  if (dom.newFolderBtn) dom.newFolderBtn.style.display = isDocs ? 'inline-flex' : 'none';

  // Toggle Preview action buttons
  if (dom.editNoteBtn) dom.editNoteBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.deleteNoteBtn) dom.deleteNoteBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.editDocBtn) dom.editDocBtn.style.display = isDocs ? 'inline-flex' : 'none';
  if (dom.deleteDocBtn) dom.deleteDocBtn.style.display = isDocs ? 'inline-flex' : 'none';
  if (dom.toggleStatusBtn) dom.toggleStatusBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.previewMetaRow) {
    const metaRow = document.querySelector('.preview-meta-row');
    if (metaRow) metaRow.style.display = isDocs ? 'none' : 'flex';
  }
  if (dom.previewTagsRow) dom.previewTagsRow.style.display = 'flex';
  if (dom.docsBreadcrumb) dom.docsBreadcrumb.style.display = isDocs ? 'flex' : 'none';

  // Load scope-appropriate tags
  await loadTags(isDocs ? 'docs' : 'notes');

  if (isDocs) {
    // Load docs tree if empty
    if (!state.docsTree || state.docsTree.length === 0) {
      await loadDocsTree();
    } else {
      renderDocsTree('');
    }

    // Determine document to display
    const docToLoad = targetDocPath || state.selectedDocPath || getFirstDocPath(state.docsTree) || 'README.md';
    const isMobile = window.innerWidth <= 768;
    const shouldOpenMobile = !isMobile || Boolean(targetDocPath);
    await selectDoc(docToLoad, Boolean(targetDocPath), shouldOpenMobile);
    if (!shouldOpenMobile) {
      showMobileList();
    }
  } else {
    // Revert URL to root if it was on a /docs path
    if (window.location.pathname.startsWith('/docs')) {
      window.history.pushState({}, '', '/');
    }
    showMobileList();
    scrollPreviewToTop();
    filterAndRender();
  }
}

/**
 * Loads the documentation directory tree from the backend and prefetches all documents.
 */
export async function loadDocsTree() {
  try {
    if (!state.docsTree || state.docsTree.length === 0) {
      if (dom.docsTree) {
        dom.docsTree.innerHTML = '<div class="loading-state">Loading documentation...</div>';
      }
      state.docsTree = await fetchDocsTree();
    }
    renderDocsTree(dom.searchInput?.value || '');
    prefetchAllDocs(state.docsTree);
  } catch (err) {
    console.error('Error loading docs tree:', err);
    if (dom.docsTree) {
      dom.docsTree.innerHTML = `<div class="empty-list">Failed to load documentation: ${escapeHtml(err.message)}</div>`;
    }
  }
}

function getFirstDocPath(tree) {
  if (!tree || !Array.isArray(tree)) return null;
  for (const item of tree) {
    if (item.type === 'file' && item.path) return item.path;
    if (item.children && Array.isArray(item.children)) {
      const childDoc = getFirstDocPath(item.children);
      if (childDoc) return childDoc;
    }
  }
  return null;
}

/**
 * Prefetches all documentation files in the background and caches their content and pre-rendered HTML.
 */
function prefetchAllDocs(tree) {
  if (!tree || !Array.isArray(tree)) return;

  function collectPaths(items) {
    const paths = [];
    for (const item of items) {
      if (item.type === 'file' && item.path) {
        paths.push(item.path);
      }
      if (item.children && Array.isArray(item.children)) {
        paths.push(...collectPaths(item.children));
      }
    }
    return paths;
  }

  const paths = collectPaths(tree);
  paths.forEach(async (docPath) => {
    if (state.docsCache && state.docsCache.has(docPath)) return;
    try {
      const data = await fetchDocContent(docPath);
      if (state.docsCache && !state.docsCache.has(docPath)) {
        const html = renderMarkdown(data.content);
        state.docsCache.set(docPath, {
          content: data.content,
          html,
          tags: data.tags || [],
          title: data.title || '',
        });
      }
    } catch {
      // Ignore background prefetch errors
    }
  });
}

/**
 * Recursive rendering of a tree item (folder or document) for arbitrary nested depth.
 */
function renderTreeItem(item, query, depth = 0) {
  const isDirectory = item.type === 'directory';
  const itemPath = item.path || item.Path || '';
  const itemTitle = item.title || item.Title || item.name || item.Name || itemPath;
  const itemName = item.name || item.Name || '';

  const parsed = query ? searchRegistry.parseQuery(query) : null;

  if (isDirectory) {
    const isExpanded = state.docsExpandedFolders.has(itemPath) || state.docsExpandedFolders.has(itemName) || (query && query.length > 0) || Boolean(state.activeDocTagFilter);
    const childrenList = item.children || item.Children || [];

    const renderedChildren = childrenList
      .map((child) => renderTreeItem(child, query, depth + 1))
      .filter(Boolean);

    const matchText = (parsed?.cleanText || query || '').toLowerCase();
    const selfMatches = !matchText || 
      (matchText && (itemTitle.toLowerCase().includes(matchText) || itemName.toLowerCase().includes(matchText) || itemPath.toLowerCase().includes(matchText)));

    if ((query || state.activeDocTagFilter) && !selfMatches && renderedChildren.length === 0) {
      return null;
    }

    const folderBadge = childrenList.length;

    return `
      <div class="tree-folder ${isExpanded ? 'expanded' : 'collapsed'}" data-folder-path="${escapeHtml(itemPath || itemName)}">
        <div class="tree-folder-header" data-folder-path="${escapeHtml(itemPath || itemName)}">
          <span class="tree-chevron">${isExpanded ? '▼' : '▶'}</span>
          <span class="tree-folder-title">${escapeHtml(itemTitle)}</span>
          <span class="tree-count-badge">${folderBadge}</span>
        </div>
        <div class="tree-folder-children" style="display: ${isExpanded ? 'block' : 'none'};">
          ${renderedChildren.length > 0 ? renderedChildren.join('') : '<div class="empty-list" style="padding: 4px 8px; font-size: 11px;">(empty)</div>'}
        </div>
      </div>
    `;
  } else {
    const itemTags = item.tags || item.Tags || [];

    if (parsed) {
      if (parsed.tags.length > 0) {
        const lowerTags = itemTags.map(t => t.toLowerCase());
        const hasAllTags = parsed.tags.every(t => lowerTags.includes(t));
        if (!hasAllTags) return null;
      }
      if (parsed.cleanText) {
        const text = parsed.cleanText.toLowerCase();
        const match = itemTitle.toLowerCase().includes(text) || 
                      itemName.toLowerCase().includes(text) || 
                      itemPath.toLowerCase().includes(text);
        if (!match) return null;
      }
    } else if (query) {
      const match = itemTitle.toLowerCase().includes(query.toLowerCase()) || 
                    itemName.toLowerCase().includes(query.toLowerCase()) || 
                    itemPath.toLowerCase().includes(query.toLowerCase());
      if (!match) return null;
    }

    if (state.activeDocTagFilter) {
      const hasTag = itemTags.some(t => t.toLowerCase() === state.activeDocTagFilter.toLowerCase());
      if (!hasTag) return null;
    }

    const isSelected = state.selectedDocPath === itemPath;
    const tagsHtml = itemTags.length > 0 
      ? `<span class="tree-file-tags">${itemTags.slice(0, 2).map(t => `<span class="tree-tag-badge">#${escapeHtml(t)}</span>`).join('')}</span>` 
      : '';

    return `
      <div class="tree-file-item ${isSelected ? 'active' : ''}" data-path="${escapeHtml(itemPath)}">
        <span class="tree-file-title" title="${escapeHtml(itemTitle)}">${escapeHtml(itemTitle)}</span>
        ${tagsHtml}
      </div>
    `;
  }
}

/**
 * Renders the interactive directory tree into the left pane.
 */
export function renderDocsTree(filterText = '') {
  if (!dom.docsTree) return;
  if (!state.docsTree || state.docsTree.length === 0) {
    dom.docsTree.innerHTML = '<div class="empty-list">No documentation available.</div>';
    return;
  }

  const query = filterText.toLowerCase();
  const renderedNodes = state.docsTree
    .map((item) => renderTreeItem(item, query, 0))
    .filter(Boolean);

  if (renderedNodes.length === 0) {
    dom.docsTree.innerHTML = `<div class="empty-list">No docs matching "${escapeHtml(filterText)}"</div>`;
    return;
  }

  dom.docsTree.innerHTML = `<div class="docs-tree-root">${renderedNodes.join('')}</div>`;
  bindTreeEvents();
}

/**
 * Attaches click event listeners to tree folder headers and document items.
 */
function bindTreeEvents() {
  if (!dom.docsTree) return;

  // Folder header toggle
  dom.docsTree.querySelectorAll('.tree-folder-header').forEach((el) => {
    el.addEventListener('click', (e) => {
      e.stopPropagation();
      const folderPath = el.getAttribute('data-folder-path');
      if (state.docsExpandedFolders.has(folderPath)) {
        state.docsExpandedFolders.delete(folderPath);
      } else {
        state.docsExpandedFolders.add(folderPath);
      }
      renderDocsTree(dom.searchInput?.value || '');
    });
  });

  // File item selection
  dom.docsTree.querySelectorAll('.tree-file-item').forEach((el) => {
    el.addEventListener('click', (e) => {
      e.stopPropagation();
      const docPath = el.getAttribute('data-path');
      selectDoc(docPath, true);
    });
  });
}

/**
 * Loads and displays the selected documentation file in the preview pane.
 */
export async function selectDoc(docPath, updateUrl = true, openMobilePreview = true) {
  state.selectedDocPath = docPath;
  scrollPreviewToTop();

  // Auto-expand all ancestor folders of this doc path
  if (docPath && docPath.includes('/')) {
    const parts = docPath.split('/');
    let cur = '';
    for (let i = 0; i < parts.length - 1; i++) {
      cur = cur ? cur + '/' + parts[i] : parts[i];
      state.docsExpandedFolders.add(cur);
    }
    renderDocsTree(dom.searchInput?.value || '');
  }

  // Update active item in tree
  if (dom.docsTree) {
    dom.docsTree.querySelectorAll('.tree-file-item').forEach((el) => {
      const isActive = el.getAttribute('data-path') === docPath;
      el.classList.toggle('active', isActive);
      if (isActive) {
        el.scrollIntoView({ block: 'nearest' });
      }
    });
  }

  // Update URL if requested
  if (updateUrl) {
    window.history.pushState({}, '', `/docs/${docPath}`);
  }

  // Display in preview
  if (dom.previewEmpty) dom.previewEmpty.style.display = 'none';
  if (dom.previewContent) dom.previewContent.style.display = 'block';

  // Update Breadcrumb
  if (dom.docsBreadcrumbPath) {
    dom.docsBreadcrumbPath.textContent = docPath.replace(/\//g, '  ›  ');
  }

  if (dom.previewShortId) {
    dom.previewShortId.textContent = 'DOC';
  }

  // Fast path: if document is cached in memory, render immediately with zero flicker
  if (state.docsCache && state.docsCache.has(docPath)) {
    const cachedData = state.docsCache.get(docPath);
    renderLoadedDoc(docPath, cachedData.content, cachedData.html, openMobilePreview, cachedData.tags || []);
    return;
  }

  if (dom.previewPane) {
    dom.previewPane.classList.add('doc-transitioning');
  }

  try {
    const data = await fetchDocContent(docPath);
    if (state.selectedDocPath !== docPath) return;
    const html = renderMarkdown(data.content);
    if (state.docsCache) {
      state.docsCache.set(docPath, {
        content: data.content,
        html,
        tags: data.tags || [],
        title: data.title || '',
      });
    }
    renderLoadedDoc(docPath, data.content, html, openMobilePreview, data.tags || []);
  } catch (err) {
    console.error('Error fetching doc content:', err);
    if (dom.previewMarkdownBody) {
      dom.previewMarkdownBody.innerHTML = `<div class="empty-list">Failed to load doc: ${escapeHtml(err.message)}</div>`;
    }
  } finally {
    if (dom.previewPane) {
      dom.previewPane.classList.remove('doc-transitioning');
    }
  }
}

/**
 * Renders loaded document markdown and triggers post-render enhancements.
 */
function renderLoadedDoc(docPath, content, html, openMobilePreview, tags = []) {
  state.currentDocContent = content;
  state.currentDocTags = tags;

  // Derive title from content or path
  let title = docPath;
  const firstLine = content.split('\n')[0];
  if (firstLine.startsWith('# ')) {
    title = firstLine.replace('# ', '').trim();
  }
  if (dom.previewTitle) dom.previewTitle.textContent = title;

  // Render tags row
  if (dom.previewTagsRow) {
    if (tags && tags.length > 0) {
      dom.previewTagsRow.innerHTML = tags
        .map((t) => `<span class="preview-tag-chip" data-tag="${escapeHtml(t)}">#${escapeHtml(t)}</span>`)
        .join('');
      dom.previewTagsRow.style.display = 'flex';
    } else {
      dom.previewTagsRow.innerHTML = '';
      dom.previewTagsRow.style.display = 'none';
    }
  }

  // Render Markdown and Mermaid diagrams
  if (dom.previewMarkdownBody) {
    dom.previewMarkdownBody.innerHTML = html || renderMarkdown(content);
    applyPostMarkdownEnhancements(dom.previewMarkdownBody);
  }

  if (dom.previewRawBody) {
    dom.previewRawBody.textContent = content;
  }

  scrollPreviewToTop();
  updateViewModeDisplay();
  if (openMobilePreview) {
    showMobileDetail();
  }
}

/**
 * Main Search handler when in Docs mode.
 */
export function handleDocsSearch(query) {
  query = (query || '').trim();
  renderDocsTree(query);
  countDocsTreeMatches(query);
}

/**
 * Highlights text occurrences inside the active document preview.
 */
function highlightInDoc(query) {
  if (!dom.previewMarkdownBody || !state.currentDocContent) return;

  if (!query) {
    dom.previewMarkdownBody.innerHTML = renderMarkdown(state.currentDocContent);
    applyPostMarkdownEnhancements(dom.previewMarkdownBody);
    if (dom.searchCount) dom.searchCount.textContent = '0';
    return;
  }

  // Re-render fresh HTML
  dom.previewMarkdownBody.innerHTML = renderMarkdown(state.currentDocContent);
  applyPostMarkdownEnhancements(dom.previewMarkdownBody);

  const regex = new RegExp(`(${escapeRegex(query)})`, 'gi');
  let matchCount = 0;

  function walk(node) {
    if (node.nodeType === Node.TEXT_NODE) {
      if (regex.test(node.nodeValue)) {
        const span = document.createElement('span');
        span.innerHTML = node.nodeValue.replace(regex, (m) => {
          matchCount++;
          return `<mark class="highlight-match">${escapeHtml(m)}</mark>`;
        });
        node.parentNode.replaceChild(span, node);
      }
    } else if (node.nodeType === Node.ELEMENT_NODE && node.tagName !== 'SCRIPT' && node.tagName !== 'STYLE') {
      Array.from(node.childNodes).forEach(walk);
    }
  }

  walk(dom.previewMarkdownBody);

  if (dom.searchCount) {
    dom.searchCount.textContent = `${matchCount} in doc`;
  }

  const firstHighlight = dom.previewMarkdownBody.querySelector('.highlight-match');
  if (firstHighlight) {
    firstHighlight.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }
}

function countDocsTreeMatches(query) {
  if (!dom.searchCount) return;
  if (!query) {
    dom.searchCount.textContent = '0';
    return;
  }
  const count = dom.docsTree?.querySelectorAll('.tree-file-item').length || 0;
  dom.searchCount.textContent = `${count} docs`;
}

function escapeRegex(string) {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Drag and Drop Folder Importer using HTML5 webkitGetAsEntry.
 */
export function initDragAndDrop() {
  if (!dom.docsPane) return;

  ['dragenter', 'dragover'].forEach((name) => {
    dom.docsPane.addEventListener(name, (e) => {
      e.preventDefault();
      e.stopPropagation();
      dom.docsPane.classList.add('drag-over');
    });
  });

  ['dragleave', 'dragend'].forEach((name) => {
    dom.docsPane.addEventListener(name, (e) => {
      e.preventDefault();
      e.stopPropagation();
      dom.docsPane.classList.remove('drag-over');
    });
  });

  dom.docsPane.addEventListener('drop', async (e) => {
    e.preventDefault();
    e.stopPropagation();
    dom.docsPane.classList.remove('drag-over');

    const items = e.dataTransfer.items;
    if (!items || items.length === 0) return;

    showToast('Scanning dropped folders and Markdown files...');
    const collected = [];

    async function traverseEntry(entry, currentPath = '') {
      if (entry.isFile) {
        if (entry.name.toLowerCase().endsWith('.md') || entry.name.toLowerCase().endsWith('.markdown')) {
          const file = await new Promise((resolve) => entry.file(resolve));
          const text = await file.text();
          const docPath = currentPath ? `${currentPath}/${entry.name}` : entry.name;
          collected.push({
            path: docPath,
            type: 'doc',
            content: text,
          });
        }
      } else if (entry.isDirectory) {
        const dirPath = currentPath ? `${currentPath}/${entry.name}` : entry.name;
        collected.push({
          path: dirPath,
          type: 'folder',
        });
        const reader = entry.createReader();
        const readAllEntries = async () => {
          return new Promise((resolve) => {
            reader.readEntries(async (entries) => {
              if (entries.length === 0) {
                resolve();
              } else {
                for (const child of entries) {
                  await traverseEntry(child, dirPath);
                }
                await readAllEntries();
                resolve();
              }
            });
          });
        };
        await readAllEntries();
      }
    }

    try {
      for (let i = 0; i < items.length; i++) {
        const entry = items[i].webkitGetAsEntry ? items[i].webkitGetAsEntry() : null;
        if (entry) {
          await traverseEntry(entry);
        }
      }

      if (collected.length === 0) {
        showToast('No Markdown (.md) files found in dropped items');
        return;
      }

      showToast(`Importing ${collected.length} items into database...`);
      const res = await importDocsTree(collected, true);
      state.docsTree = res.tree || (await fetchDocsTree());
      renderDocsTree('');
      showToast(`Imported ${res.created || 0} created, ${res.updated || 0} updated documents!`);

      const firstDoc = collected.find((it) => it.type === 'doc');
      if (firstDoc) {
        await selectDoc(firstDoc.path);
      }
    } catch (err) {
      console.error('Drag and drop import failed:', err);
      showToast(`Import failed: ${err.message}`);
    }
  });

  // Clicking dropzone opens standard file dialog fallback
  if (dom.docsDropzone) {
    dom.docsDropzone.addEventListener('click', () => {
      openNewDocModal();
    });
  }
}

export function openNewDocModal(targetFolder = '') {
  state.editingDocPath = null;
  if (dom.docModalTitle) dom.docModalTitle.textContent = 'New Document';
  if (dom.formDocPath) {
    dom.formDocPath.disabled = false;
    dom.formDocPath.value = targetFolder ? `${targetFolder}/` : '';
  }
  if (dom.formDocTitle) dom.formDocTitle.value = '';
  if (dom.formDocContent) dom.formDocContent.value = '# Title\n\n';
  if (dom.formDocTags) dom.formDocTags.value = '';
  if (dom.docModal) dom.docModal.style.display = 'flex';
  setTimeout(() => dom.formDocTitle?.focus(), 50);
}

export async function openEditDocModal(docPath = null) {
  const target = docPath || state.selectedDocPath;
  if (!target) return;
  state.editingDocPath = target;
  if (dom.docModalTitle) dom.docModalTitle.textContent = 'Edit Document';
  if (dom.formDocPath) {
    dom.formDocPath.value = target;
    dom.formDocPath.disabled = true;
  }

  let content = state.currentDocContent || '';
  let title = dom.previewTitle?.textContent || target;
  let tags = state.currentDocTags || [];

  if (target !== state.selectedDocPath || !content) {
    try {
      const data = await fetchDocContent(target);
      content = data.content || '';
      title = data.title || target;
      tags = data.tags || [];
    } catch (e) {
      console.error('Failed to fetch doc for editing:', e);
    }
  }

  if (dom.formDocTitle) dom.formDocTitle.value = title;
  if (dom.formDocContent) dom.formDocContent.value = content;
  if (dom.formDocTags) dom.formDocTags.value = tags.join(', ');
  if (dom.docModal) dom.docModal.style.display = 'flex';
  setTimeout(() => dom.formDocContent?.focus(), 50);
}

export function closeDocModal() {
  if (dom.docModal) dom.docModal.style.display = 'none';
  state.editingDocPath = null;
}

export function openNewFolderModal(targetParent = '') {
  if (dom.formFolderPath) dom.formFolderPath.value = targetParent ? `${targetParent}/` : '';
  if (dom.formFolderTitle) dom.formFolderTitle.value = '';
  if (dom.folderModal) dom.folderModal.style.display = 'flex';
  setTimeout(() => dom.formFolderPath?.focus(), 50);
}

export function closeFolderModal() {
  if (dom.folderModal) dom.folderModal.style.display = 'none';
}

export function getOrderedDocPaths() {
  const paths = [];
  function collect(items) {
    if (!items || !Array.isArray(items)) return;
    for (const item of items) {
      const p = item.path || item.Path;
      if (item.type === 'file' && p) {
        paths.push(p);
      }
      if (item.children && Array.isArray(item.children)) {
        collect(item.children);
      }
    }
  }
  collect(state.docsTree);
  return paths;
}

export function canNavigateDocs(direction) {
  const paths = getOrderedDocPaths();
  if (paths.length === 0 || !state.selectedDocPath) return false;
  const idx = paths.indexOf(state.selectedDocPath);
  if (idx === -1) return false;
  if (direction === 'prev') return idx > 0;
  if (direction === 'next') return idx < paths.length - 1;
  return false;
}

export async function navigateDocs(direction) {
  const paths = getOrderedDocPaths();
  if (paths.length === 0 || !state.selectedDocPath) return false;
  const idx = paths.indexOf(state.selectedDocPath);
  if (idx === -1) return false;

  let targetIdx = idx;
  if (direction === 'prev' && idx > 0) targetIdx--;
  if (direction === 'next' && idx < paths.length - 1) targetIdx++;

  if (targetIdx !== idx) {
    await selectDoc(paths[targetIdx], true);
    return true;
  }
  return false;
}

export function getAdjacentDocTitle(direction) {
  const paths = getOrderedDocPaths();
  if (paths.length === 0 || !state.selectedDocPath) return null;
  const idx = paths.indexOf(state.selectedDocPath);
  if (idx === -1) return null;
  if (direction === 'prev' && idx > 0) return paths[idx - 1];
  if (direction === 'next' && idx < paths.length - 1) return paths[idx + 1];
  return null;
}
