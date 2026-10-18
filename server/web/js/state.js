/**
 * KB-CLI Web Fuzzy Finder — State Store
 */

export const state = {
  allItems: [],
  filteredItems: [],
  selectedIndex: 0,
  activeTypeFilter: 'all',
  activeTagFilter: null,
  currentNoteDetail: null,
  isRawView: false,
  isAuthenticated: false,
  isAuthRequired: false,

  // Documentation Mode State
  currentViewMode: 'notes', // 'notes' or 'docs'
  docsTree: [],
  selectedDocPath: null,
  currentDocContent: null,
  docsCache: new Map(),
  docsExpandedFolders: new Set(['docs', 'proposals']),
};

