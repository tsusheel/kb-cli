/**
 * KB-CLI Web Fuzzy Finder — API Client
 * Manages all HTTP communication with backend REST endpoints.
 */

import { state } from './state.js';
import { dom } from './dom.js';

let onUnauthorizedCallback = null;

export function setOnUnauthorized(cb) {
  onUnauthorizedCallback = cb;
}

/**
 * Standard fetch wrapper with automatic 401 session expiration handling.
 */
export async function apiFetch(url, options = {}) {
  const res = await fetch(url, options);
  if (res.status === 401) {
    state.isAuthenticated = false;
    if (dom.logoutBtn) dom.logoutBtn.style.display = 'none';
    if (typeof onUnauthorizedCallback === 'function') {
      onUnauthorizedCallback();
    }
    throw new Error('Authentication required');
  }
  return res;
}

/**
 * Checks server auth configuration and session validity.
 */
export async function checkAuthStatus() {
  try {
    const res = await fetch('/api/auth/status');
    if (res.ok) {
      const data = await res.json();
      state.isAuthenticated = data.authenticated;
      state.isAuthRequired = data.auth_required;
      if (dom.logoutBtn) {
        dom.logoutBtn.style.display =
          state.isAuthenticated && state.isAuthRequired ? 'inline-flex' : 'none';
      }
      return {
        isAuthenticated: state.isAuthenticated,
        isAuthRequired: state.isAuthRequired,
      };
    }
  } catch (e) {
    console.warn('Auth status check failed:', e);
  }
  return { isAuthenticated: false, isAuthRequired: false };
}

/**
 * Authenticates against the backend.
 */
export async function login(password) {
  const res = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  });
  const data = await res.json();
  return { ok: res.ok, data };
}

/**
 * Logs out of current session.
 */
export async function logout() {
  try {
    await fetch('/api/auth/logout', { method: 'POST' });
  } catch (e) {
    console.warn('Logout request failed:', e);
  }
  state.isAuthenticated = false;
}

/**
 * Fetches notes and logs for fuzzy search.
 */
export async function fetchItems(includeDeleted = false) {
  const res = await apiFetch(`/api/items?include_deleted=${includeDeleted}`);
  if (!res.ok) throw new Error('Failed to load items');
  return res.json();
}

/**
 * Fetches all unique tags with usage counts (optional scope: 'notes', 'docs', or all).
 */
export async function fetchTags(scope = null) {
  const url = scope ? `/api/tags?scope=${encodeURIComponent(scope)}` : '/api/tags';
  const res = await apiFetch(url);
  if (!res.ok) return [];
  return res.json();
}

/**
 * Fetches system stats (note counts, streak, daily logs).
 */
export async function fetchStats() {
  const res = await apiFetch('/api/stats');
  if (!res.ok) return null;
  return res.json();
}

/**
 * Fetches detailed metadata for a single note (links, revisions).
 */
export async function fetchNoteDetail(noteId) {
  const res = await apiFetch(`/api/notes/${noteId}`);
  if (!res.ok) throw new Error('Failed to load note detail');
  return res.json();
}

/**
 * Creates or updates a note.
 */
export async function saveNote(payload, noteId = null) {
  const url = noteId ? `/api/notes/${noteId}` : '/api/notes';
  const method = noteId ? 'PUT' : 'POST';
  const res = await apiFetch(url, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error(noteId ? 'Failed to update note' : 'Failed to create note');
  return res.json();
}

/**
 * Toggles a note's status between active and completed.
 */
export async function updateNoteStatus(noteId, newStatus) {
  const res = await apiFetch(`/api/notes/${noteId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status: newStatus }),
  });
  if (!res.ok) throw new Error('Failed to update status');
  return res.json();
}

/**
 * Soft-deletes a note with a reason.
 */
export async function deleteNote(noteId, reason = 'deleted from web UI') {
  const res = await apiFetch(`/api/notes/${noteId}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ reason }),
  });
  if (!res.ok) throw new Error('Failed to delete note');
  return res.json();
}

/**
 * Triggers database sync.
 */
export async function triggerSync() {
  const res = await apiFetch('/api/sync', { method: 'POST' });
  return res.json();
}

/**
 * Fetches the documentation directory tree.
 */
export async function fetchDocsTree() {
  const res = await apiFetch('/api/docs');
  if (!res.ok) throw new Error('Failed to load documentation tree');
  return res.json();
}

/**
 * Fetches the raw markdown content of a documentation file.
 */
export async function fetchDocContent(docPath) {
  const res = await apiFetch(`/api/docs/content?path=${encodeURIComponent(docPath)}`);
  if (!res.ok) throw new Error('Failed to load document content');
  return res.json();
}

/**
 * Updates an existing document's content, title, and/or tags.
 */
export async function saveDocContent(docPath, title, content, tags = null) {
  const payload = { path: docPath, title, content };
  if (tags !== null) payload.tags = tags;
  const res = await apiFetch('/api/docs/content', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to save document');
  return res.json();
}

/**
 * Creates a new document.
 */
export async function createDoc(payload) {
  const res = await apiFetch('/api/docs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to create document');
  return res.json();
}

/**
 * Creates a new documentation folder.
 */
export async function createDocFolder(payload) {
  const res = await apiFetch('/api/docs/folders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw new Error('Failed to create folder');
  return res.json();
}

/**
 * Soft-deletes a document or folder.
 */
export async function deleteDoc(docPath, reason = 'deleted from web UI') {
  const res = await apiFetch(`/api/docs?path=${encodeURIComponent(docPath)}&reason=${encodeURIComponent(reason)}`, {
    method: 'DELETE',
  });
  if (!res.ok) throw new Error('Failed to delete document');
  return res.json();
}

/**
 * Batch-imports a tree of folders and markdown files.
 */
export async function importDocsTree(items, overwrite = true) {
  const res = await apiFetch('/api/docs/import-tree', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ items, overwrite }),
  });
  if (!res.ok) throw new Error('Failed to import documentation tree');
  return res.json();
}

/**
 * Performs full-text search across documentation.
 */
export async function searchDocs(query) {
  const res = await apiFetch(`/api/docs/search?q=${encodeURIComponent(query)}`);
  if (!res.ok) throw new Error('Documentation search failed');
  return res.json();
}

