/**
 * KB-CLI Web Fuzzy Finder — Documentation Directory Tree & Viewer Controller
 * Manages the collapsible tree directory view, document selection,
 * markdown/mermaid rendering, filtering, and deep-link routing.
 */

import { state } from './state.js';
import { dom } from './dom.js';
import { fetchDocsTree, fetchDocContent } from './api.js';
import { renderMarkdown, applyPostMarkdownEnhancements } from './markdown.js';
import { showMobileDetail, showMobileList, updateViewModeDisplay } from './finder.js';
import { escapeHtml } from './utils.js';

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

  // 2. Filter input listener
  if (dom.docsSearchInput) {
    dom.docsSearchInput.addEventListener('input', (e) => {
      renderDocsTree(e.target.value.trim());
    });
  }

  // 3. Intra-document link interception in preview markdown body
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

  // 4. Check for URL deep linking
  handleDocsRouting();
  window.addEventListener('popstate', handleDocsRouting);
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

  // Toggle navigation tab active states
  if (dom.tabNotes) dom.tabNotes.classList.toggle('active', !isDocs);
  if (dom.tabDocs) dom.tabDocs.classList.toggle('active', isDocs);

  // Toggle Left Panes & Filter Bar
  if (dom.resultsPane) dom.resultsPane.style.display = isDocs ? 'none' : 'flex';
  if (dom.filterBar) dom.filterBar.style.display = isDocs ? 'none' : 'flex';
  if (dom.docsPane) dom.docsPane.style.display = isDocs ? 'flex' : 'none';

  // Toggle Header action buttons
  if (dom.newNoteBtn) dom.newNoteBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.syncBtn) dom.syncBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.searchInput) {
    dom.searchInput.parentElement.parentElement.style.display = isDocs ? 'none' : 'flex';
  }

  // Toggle Preview action buttons not applicable to docs
  if (dom.editNoteBtn) dom.editNoteBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.deleteNoteBtn) dom.deleteNoteBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.toggleStatusBtn) dom.toggleStatusBtn.style.display = isDocs ? 'none' : 'inline-flex';
  if (dom.previewMetaRow) {
    const metaRow = document.querySelector('.preview-meta-row');
    if (metaRow) metaRow.style.display = isDocs ? 'none' : 'flex';
  }
  if (dom.previewTagsRow) dom.previewTagsRow.style.display = isDocs ? 'none' : 'flex';
  if (dom.docsBreadcrumb) dom.docsBreadcrumb.style.display = isDocs ? 'flex' : 'none';

  if (isDocs) {
    // Load docs tree if empty
    if (!state.docsTree || state.docsTree.length === 0) {
      await loadDocsTree();
    } else {
      renderDocsTree(dom.docsSearchInput?.value || '');
    }

    // Determine document to display
    const docToLoad = targetDocPath || state.selectedDocPath || 'README.md';
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
    // Return mobile view to list if applicable
    showMobileList();
  }
}

/**
 * Loads the documentation directory tree from the backend.
 */
export async function loadDocsTree() {
  try {
    if (dom.docsTree) {
      dom.docsTree.innerHTML = '<div class="loading-state">Loading documentation...</div>';
    }
    state.docsTree = await fetchDocsTree();
    renderDocsTree(dom.docsSearchInput?.value || '');
  } catch (err) {
    console.error('Error loading docs tree:', err);
    if (dom.docsTree) {
      dom.docsTree.innerHTML = `<div class="empty-list">Failed to load documentation: ${escapeHtml(err.message)}</div>`;
    }
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
  let html = '<div class="docs-tree-root">';

  state.docsTree.forEach((folder) => {
    const folderName = folder.name || folder.Name || '';
    const isExpanded = state.docsExpandedFolders.has(folderName) || query.length > 0;

    // Filter children if query is present (supports both lowercase and uppercase properties)
    const childrenList = folder.children || folder.Children || [];
    const visibleChildren = childrenList.filter((item) => {
      const itemTitle = item.title || item.Title || '';
      const itemName = item.name || item.Name || '';
      const itemPath = item.path || item.Path || '';
      if (!query) return true;
      return (
        itemTitle.toLowerCase().includes(query) ||
        itemName.toLowerCase().includes(query) ||
        itemPath.toLowerCase().includes(query)
      );
    });

    if (query && visibleChildren.length === 0) {
      return; // Skip empty folders when filtering
    }

    const folderTitle = folder.title || folder.Title || folderName;
    const folderBadge = visibleChildren.length;

    html += `
      <div class="tree-folder ${isExpanded ? 'expanded' : 'collapsed'}" data-folder="${escapeHtml(folderName)}">
        <div class="tree-folder-header" data-folder="${escapeHtml(folderName)}">
          <span class="tree-chevron">${isExpanded ? '▼' : '▶'}</span>
          <span class="tree-folder-title">${escapeHtml(folderTitle)}</span>
          <span class="tree-count-badge">${folderBadge}</span>
        </div>
        <div class="tree-folder-children" style="display: ${isExpanded ? 'block' : 'none'};">
    `;

    visibleChildren.forEach((file) => {
      const filePath = file.path || file.Path || '';
      const fileTitle = file.title || file.Title || file.name || file.Name || filePath;
      const isSelected = state.selectedDocPath === filePath;

      html += `
        <div class="tree-file-item ${isSelected ? 'active' : ''}" data-path="${escapeHtml(filePath)}">
          <span class="tree-file-title" title="${escapeHtml(fileTitle)}">${escapeHtml(fileTitle)}</span>
        </div>
      `;
    });

    html += `
        </div>
      </div>
    `;
  });

  html += '</div>';
  dom.docsTree.innerHTML = html;

  // Bind click handlers
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
      const folder = el.getAttribute('data-folder');
      if (state.docsExpandedFolders.has(folder)) {
        state.docsExpandedFolders.delete(folder);
      } else {
        state.docsExpandedFolders.add(folder);
      }
      renderDocsTree(dom.docsSearchInput?.value || '');
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

  // Update active item in tree
  if (dom.docsTree) {
    dom.docsTree.querySelectorAll('.tree-file-item').forEach((el) => {
      el.classList.toggle('active', el.getAttribute('data-path') === docPath);
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
    dom.docsBreadcrumbPath.textContent = docPath.replace('/', '  ›  ');
  }

  if (dom.previewShortId) {
    dom.previewShortId.textContent = 'DOC';
  }

  try {
    const data = await fetchDocContent(docPath);
    state.currentDocContent = data.content;

    // Derive title from content or path
    let title = docPath;
    const firstLine = data.content.split('\n')[0];
    if (firstLine.startsWith('# ')) {
      title = firstLine.replace('# ', '').trim();
    }
    if (dom.previewTitle) dom.previewTitle.textContent = title;

    // Render Markdown and Mermaid diagrams
    if (dom.previewMarkdownBody) {
      dom.previewMarkdownBody.innerHTML = renderMarkdown(data.content);
      applyPostMarkdownEnhancements(dom.previewMarkdownBody);
    }

    if (dom.previewRawBody) {
      dom.previewRawBody.textContent = data.content;
    }

    updateViewModeDisplay();
    if (openMobilePreview) {
      showMobileDetail();
    }
  } catch (err) {
    console.error('Error fetching doc content:', err);
    if (dom.previewMarkdownBody) {
      dom.previewMarkdownBody.innerHTML = `<div class="empty-list">Failed to load doc: ${escapeHtml(err.message)}</div>`;
    }
  }
}
