/**
 * KB-CLI Web Fuzzy Finder — Utilities & Helpers
 */

import { dom } from './dom.js';

let toastTimer = null;

/**
 * Escapes special HTML characters to prevent XSS.
 */
export function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

/**
 * Displays a non-blocking toast notification.
 */
export function showToast(message, actionText, actionCallback) {
  if (toastTimer) {
    clearTimeout(toastTimer);
    toastTimer = null;
  }
  if (!dom.toast) return;

  dom.toast.innerHTML = '';
  const span = document.createElement('span');
  span.textContent = message;
  dom.toast.appendChild(span);

  if (actionText && actionCallback) {
    const btn = document.createElement('button');
    btn.className = 'toast-action-btn';
    btn.textContent = actionText;
    btn.addEventListener('click', () => {
      dom.toast.style.display = 'none';
      actionCallback();
    });
    dom.toast.appendChild(btn);
  }

  dom.toast.style.display = 'flex';
  if (!actionText) {
    toastTimer = setTimeout(() => {
      dom.toast.style.display = 'none';
    }, 3000);
  }
}

/**
 * Fallback clipboard copy for browsers without navigator.clipboard API.
 */
export function copyTextFallback(text) {
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

/**
 * Triggers temporary visual "Copied!" feedback on a copy button.
 */
export function triggerCopyFeedback(btn) {
  const textSpan = btn.querySelector('.copy-text');
  const origText = textSpan ? textSpan.textContent : 'Copy';
  btn.classList.add('copied');
  if (textSpan) textSpan.textContent = 'Copied!';

  setTimeout(() => {
    btn.classList.remove('copied');
    if (textSpan) textSpan.textContent = origText;
  }, 2000);
}

/**
 * Fuzzy matching scoring algorithm with word-boundary and consecutive character bonuses.
 */
export function fuzzyMatch(pattern, str) {
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
      if (
        targetIdx === 0 ||
        target[targetIdx - 1] === ' ' ||
        target[targetIdx - 1] === '/' ||
        target[targetIdx - 1] === '-'
      ) {
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

/**
 * Highlights matched positions in a string with a stylized span.
 */
export function highlightPositions(str, positions) {
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

/**
 * Scrolls the preview container (and window/document) to the top immediately.
 */
export function scrollPreviewToTop() {
  if (dom.previewPane) {
    if (typeof dom.previewPane.scrollTo === 'function') {
      dom.previewPane.scrollTo({ top: 0, left: 0, behavior: 'instant' });
    }
    dom.previewPane.scrollTop = 0;
  }
  window.scrollTo({ top: 0, left: 0, behavior: 'instant' });
  if (document.documentElement) document.documentElement.scrollTop = 0;
  if (document.body) document.body.scrollTop = 0;
}
