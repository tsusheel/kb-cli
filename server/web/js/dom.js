/**
 * KB-CLI Web Fuzzy Finder — DOM References
 */

export const dom = {
  // App Container
  appContainer: document.getElementById('app'),

  // Search Elements
  searchInput: document.getElementById('search-input'),
  clearSearchBtn: document.getElementById('clear-search-btn'),
  searchCount: document.getElementById('search-count'),

  // Results & Navigation
  resultsList: document.getElementById('results-list'),
  mobileBackBtn: document.getElementById('mobile-back-btn'),

  // Preview Pane
  previewEmpty: document.getElementById('preview-empty'),
  previewContent: document.getElementById('preview-content'),
  previewTitle: document.getElementById('preview-title'),
  previewShortId: document.getElementById('preview-short-id'),
  previewTypeBadge: document.getElementById('preview-type-badge'),
  previewStatusBadge: document.getElementById('preview-status-badge'),
  previewAreaBadge: document.getElementById('preview-area-badge'),
  previewDueBadge: document.getElementById('preview-due-badge'),
  previewUpdatedTime: document.getElementById('preview-updated-time'),
  previewTagsRow: document.getElementById('preview-tags-row'),
  previewMarkdownBody: document.getElementById('preview-markdown-body'),
  previewRawBody: document.getElementById('preview-raw-body'),
  previewGraphSection: document.getElementById('preview-graph-section'),
  previewLinksList: document.getElementById('preview-links-list'),
  previewHistorySection: document.getElementById('preview-history-section'),
  previewHistoryList: document.getElementById('preview-history-list'),
  toggleHistoryBtn: document.getElementById('toggle-history-btn'),

  // Filters & Footer
  typeFilterPills: document.getElementById('type-filter-pills'),
  tagPills: document.getElementById('tag-pills'),
  footerStats: document.getElementById('footer-stats'),

  // Header Actions
  newNoteBtn: document.getElementById('new-note-btn'),
  syncBtn: document.getElementById('sync-btn'),
  offlineBadge: document.getElementById('offline-badge'),

  // Preview Action Buttons
  toggleViewBtn: document.getElementById('toggle-view-btn'),
  copyIdBtn: document.getElementById('copy-id-btn'),
  toggleStatusBtn: document.getElementById('toggle-status-btn'),
  editNoteBtn: document.getElementById('edit-note-btn'),
  deleteNoteBtn: document.getElementById('delete-note-btn'),

  // Note Modal Elements
  noteModal: document.getElementById('note-modal'),
  noteForm: document.getElementById('note-form'),
  modalCloseBtn: document.getElementById('modal-close-btn'),
  modalCancelBtn: document.getElementById('modal-cancel-btn'),
  modalTitle: document.getElementById('modal-title'),
  modalTabWrite: document.getElementById('modal-tab-write'),
  modalTabPreview: document.getElementById('modal-tab-preview'),
  modalFleshPreview: document.getElementById('modal-flesh-preview'),
  formNoteId: document.getElementById('form-note-id'),
  formTitle: document.getElementById('form-title'),
  formType: document.getElementById('form-type'),
  formStatus: document.getElementById('form-status'),
  formArea: document.getElementById('form-area'),
  formDue: document.getElementById('form-due'),
  formTags: document.getElementById('form-tags'),
  formFlesh: document.getElementById('form-flesh'),

  // Toast
  toast: document.getElementById('toast'),

  // Auth Elements
  logoutBtn: document.getElementById('logout-btn'),
  authModal: document.getElementById('auth-modal'),
  authForm: document.getElementById('auth-form'),
  authPassword: document.getElementById('auth-password'),
  togglePasswordBtn: document.getElementById('toggle-password-btn'),
  authError: document.getElementById('auth-error'),
  authSubmitBtn: document.getElementById('auth-submit-btn'),
};
