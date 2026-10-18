/**
 * KB-CLI Web Fuzzy Finder — Note Editor Modal
 * Handles creation and editing of notes, form validation, and live write/preview switching.
 */

import { state } from './state.js';
import { dom } from './dom.js';
import { showToast } from './utils.js';
import { saveNote } from './api.js';
import { renderMarkdown, applyPostMarkdownEnhancements } from './markdown.js';

let noteSavedCallback = null;

export function resetModalTabs() {
  if (dom.modalTabWrite && dom.modalTabPreview && dom.formFlesh && dom.modalFleshPreview) {
    dom.modalTabPreview.classList.remove('active');
    dom.modalTabWrite.classList.add('active');
    dom.modalFleshPreview.style.display = 'none';
    dom.formFlesh.style.display = 'block';
    dom.modalFleshPreview.innerHTML = '';
  }
}

export function openNoteModal(editItem = null) {
  if (!dom.noteModal) return;
  dom.noteModal.style.display = 'flex';
  resetModalTabs();

  if (editItem) {
    const detail = state.currentNoteDetail;
    const isMatchingDetail = detail && (detail.id === editItem.id || detail.id.startsWith(editItem.id));

    const flesh = isMatchingDetail ? (detail.note_flesh || editItem.flesh || '') : (editItem.flesh || '');
    const dueVal = isMatchingDetail && detail.target_date_time
      ? detail.target_date_time.split('T')[0]
      : (editItem.due || (editItem.target_date_time ? editItem.target_date_time.split('T')[0] : ''));

    if (dom.modalTitle) dom.modalTitle.textContent = 'Edit Note';
    if (dom.formNoteId) dom.formNoteId.value = editItem.id;
    if (dom.formTitle) dom.formTitle.value = editItem.display;
    if (dom.formType) dom.formType.value = editItem.type || 'note';
    if (dom.formStatus) dom.formStatus.value = editItem.status || 'active';
    if (dom.formArea) dom.formArea.value = editItem.area || '';
    if (dom.formDue) dom.formDue.value = dueVal;
    if (dom.formTags) dom.formTags.value = (editItem.tags || []).join(', ');
    if (dom.formFlesh) dom.formFlesh.value = flesh;
  } else {
    if (dom.modalTitle) dom.modalTitle.textContent = 'Create New Note';
    if (dom.noteForm) dom.noteForm.reset();
    if (dom.formNoteId) dom.formNoteId.value = '';
  }

  if (dom.formTitle) dom.formTitle.focus();
}

export function closeNoteModal() {
  if (!dom.noteModal) return;
  dom.noteModal.style.display = 'none';
  if (dom.noteForm) dom.noteForm.reset();
  resetModalTabs();
}

async function handleNoteFormSubmit(e) {
  e.preventDefault();
  const noteId = dom.formNoteId ? dom.formNoteId.value : '';
  const title = dom.formTitle ? dom.formTitle.value.trim() : '';
  const type = dom.formType ? dom.formType.value : 'note';
  const status = dom.formStatus ? dom.formStatus.value : 'active';
  const area = dom.formArea ? dom.formArea.value.trim() : '';
  const due = dom.formDue ? dom.formDue.value.trim() : '';
  const tags = dom.formTags
    ? dom.formTags.value.split(',').map((t) => t.trim()).filter(Boolean)
    : [];
  const flesh = dom.formFlesh ? dom.formFlesh.value : '';

  const payload = {
    note: title,
    content: flesh,
    type,
    status,
    area,
    due,
    tags,
  };

  try {
    if (noteId) {
      await saveNote(payload, noteId);
      showToast('Note updated successfully');
    } else {
      await saveNote(payload);
      showToast('Note created successfully');
    }
    closeNoteModal();
    if (typeof noteSavedCallback === 'function') {
      await noteSavedCallback();
    }
  } catch (err) {
    showToast(`Error: ${err.message}`);
  }
}

/**
 * Initializes modal event listeners.
 */
export function setupModalListeners({ onNoteSaved } = {}) {
  noteSavedCallback = onNoteSaved;

  if (dom.modalCloseBtn) dom.modalCloseBtn.addEventListener('click', closeNoteModal);
  if (dom.modalCancelBtn) dom.modalCancelBtn.addEventListener('click', closeNoteModal);
  if (dom.noteForm) dom.noteForm.addEventListener('submit', handleNoteFormSubmit);

  if (dom.noteModal) {
    dom.noteModal.addEventListener('click', (e) => {
      if (e.target === dom.noteModal) closeNoteModal();
    });
  }

  // Write / Preview Tabs
  if (dom.modalTabWrite && dom.modalTabPreview && dom.formFlesh && dom.modalFleshPreview) {
    dom.modalTabWrite.addEventListener('click', () => {
      dom.modalTabPreview.classList.remove('active');
      dom.modalTabWrite.classList.add('active');
      dom.modalFleshPreview.style.display = 'none';
      dom.formFlesh.style.display = 'block';
      dom.formFlesh.focus();
    });

    dom.modalTabPreview.addEventListener('click', () => {
      dom.modalTabWrite.classList.remove('active');
      dom.modalTabPreview.classList.add('active');
      dom.formFlesh.style.display = 'none';
      dom.modalFleshPreview.style.display = 'block';
      const content = dom.formFlesh.value;
      dom.modalFleshPreview.innerHTML = renderMarkdown(content);
      applyPostMarkdownEnhancements(dom.modalFleshPreview);
    });
  }
}
