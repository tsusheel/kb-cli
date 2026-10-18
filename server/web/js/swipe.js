/**
 * KB-CLI Web Fuzzy Finder — Interactive Swipe & Gesture Navigation Controller
 * Enables real-time 1:1 horizontal drag tracking, elastic boundary resistance,
 * smooth snap physics, and trackpad gestures across notes and documents.
 */

import { dom } from './dom.js';
import { state } from './state.js';
import { canNavigateNotes, navigateNotes, getAdjacentNoteTitle } from './finder.js';
import { canNavigateDocs, navigateDocs, getAdjacentDocTitle } from './docs.js';

let isSwiping = false;
let isLockedVertical = false;
let isTransitioning = false;
let startX = 0;
let startY = 0;
let startTime = 0;
let currentDeltaX = 0;
let wheelDeltaX = 0;
let wheelCooldown = false;
let wheelTimer = null;

/**
 * Checks whether an adjacent note or document exists in the given direction.
 * @param {number} direction -1 for previous, +1 for next
 */
function canNavigate(direction) {
  if (state.currentViewMode === 'docs') {
    return canNavigateDocs(direction);
  }
  return canNavigateNotes(direction);
}

/**
 * Navigates to the adjacent note or document in the given direction.
 * @param {number} direction -1 for previous, +1 for next
 */
async function navigate(direction) {
  if (state.currentViewMode === 'docs') {
    return await navigateDocs(direction);
  }
  return await navigateNotes(direction);
}

/**
 * Retrieves the display title of the adjacent note or document.
 * @param {number} direction -1 for previous, +1 for next
 */
function getAdjacentTitle(direction) {
  if (state.currentViewMode === 'docs') {
    return getAdjacentDocTitle(direction);
  }
  return getAdjacentNoteTitle(direction);
}

/**
 * Determines whether the event target is inside an element that handles horizontal scrolling.
 */
function isInsideHorizontalScrollable(target) {
  let el = target;
  while (el && el !== dom.previewPane && el !== document.body) {
    if (
      el.classList &&
      (el.classList.contains('markdown-table-wrapper') ||
        el.classList.contains('mermaid-viewport') ||
        el.classList.contains('tag-scroll-area') ||
        el.classList.contains('preview-actions') ||
        el.tagName === 'PRE' ||
        el.tagName === 'CODE')
    ) {
      if (el.scrollWidth > el.clientWidth + 4) {
        return true;
      }
    }
    el = el.parentElement;
  }
  return false;
}

/**
 * Checks if any modal or full-screen overlay is currently active.
 */
function isModalActive() {
  const noteModal = document.getElementById('note-modal');
  if (noteModal && noteModal.style.display !== 'none') return true;

  const authModal = document.getElementById('auth-modal');
  if (authModal && authModal.style.display !== 'none') return true;

  if (document.querySelector('.mermaid-fullscreen')) return true;

  return false;
}

/**
 * Truncates long titles to fit cleanly inside the indicator pill and ends with an ellipsis.
 */
function truncateTitle(str, maxLen = 32) {
  if (!str) return '';
  const clean = String(str).replace(/\s+/g, ' ').trim();
  if (clean.length <= maxLen) return clean;
  return clean.substring(0, maxLen - 1).trimEnd() + '…';
}

/**
 * Shows the floating swipe direction indicator badge with trimmed text and arrows.
 */
function showIndicator(direction, text, isBoundary = false) {
  if (!dom.previewSwipeIndicator) return;

  const displayTitle = isBoundary ? text : truncateTitle(text, 28);

  const textEl =
    dom.previewSwipeText ||
    dom.previewSwipeIndicator.querySelector('.preview-swipe-text');
  const arrowLeft =
    dom.swipeArrowLeft ||
    dom.previewSwipeIndicator.querySelector('#swipe-arrow-left');
  const arrowRight =
    dom.swipeArrowRight ||
    dom.previewSwipeIndicator.querySelector('#swipe-arrow-right');

  if (textEl) {
    textEl.textContent = displayTitle;
  }

  if (arrowLeft) {
    arrowLeft.style.display =
      !isBoundary && direction === -1 ? 'inline-flex' : 'none';
  }
  if (arrowRight) {
    arrowRight.style.display =
      !isBoundary && direction === 1 ? 'inline-flex' : 'none';
  }

  dom.previewSwipeIndicator.classList.toggle('at-boundary', isBoundary);
  dom.previewSwipeIndicator.classList.add('visible');
}

/**
 * Hides the floating swipe direction indicator badge.
 */
function hideIndicator() {
  if (!dom.previewSwipeIndicator) return;
  dom.previewSwipeIndicator.classList.remove('visible');
  const arrowLeft =
    dom.swipeArrowLeft ||
    dom.previewSwipeIndicator.querySelector('#swipe-arrow-left');
  const arrowRight =
    dom.swipeArrowRight ||
    dom.previewSwipeIndicator.querySelector('#swipe-arrow-right');
  if (arrowLeft) arrowLeft.style.display = 'none';
  if (arrowRight) arrowRight.style.display = 'none';
}

/**
 * Smoothly snaps the preview content back to center without changing items.
 */
function resetToCenter() {
  if (!dom.previewContent) return;
  dom.previewContent.style.transition =
    'transform 0.24s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.24s ease';
  dom.previewContent.style.transform = 'translate3d(0, 0, 0)';
  dom.previewContent.style.opacity = '1';

  setTimeout(() => {
    if (!isSwiping && !isTransitioning && dom.previewContent) {
      dom.previewContent.style.willChange = '';
      dom.previewContent.style.transform = '';
      dom.previewContent.style.transition = '';
      dom.previewContent.style.opacity = '';
    }
  }, 250);
}

/**
 * Performs the animated slide-out, item switch, and slide-in transition.
 */
async function performSwipeTransition(direction) {
  if (!dom.previewContent || isTransitioning) return;
  isTransitioning = true;
  hideIndicator();

  // 1. Slide out in swipe direction
  const exitX = direction === 1 ? -120 : 120;
  dom.previewContent.style.transition =
    'transform 0.16s cubic-bezier(0.2, 0.8, 0.2, 1), opacity 0.16s ease';
  dom.previewContent.style.transform = `translate3d(${exitX}px, 0, 0)`;
  dom.previewContent.style.opacity = '0';

  setTimeout(async () => {
    // 2. Load adjacent note / document
    await navigate(direction);

    // 3. Position incoming item on opposite side
    const enterX = direction === 1 ? 90 : -90;
    dom.previewContent.style.transition = 'none';
    dom.previewContent.style.transform = `translate3d(${enterX}px, 0, 0)`;
    dom.previewContent.style.opacity = '0';

    // 4. Force reflow to prepare transition
    void dom.previewContent.offsetWidth;

    // 5. Smoothly glide into center
    dom.previewContent.style.transition =
      'transform 0.24s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.24s ease';
    dom.previewContent.style.transform = 'translate3d(0, 0, 0)';
    dom.previewContent.style.opacity = '1';

    setTimeout(() => {
      isTransitioning = false;
      if (dom.previewContent) {
        dom.previewContent.style.willChange = '';
        dom.previewContent.style.transform = '';
        dom.previewContent.style.transition = '';
        dom.previewContent.style.opacity = '';
      }
    }, 250);
  }, 160);
}

/**
 * Initializes interactive swipe & wheel gestures on the preview pane.
 */
export function initSwipeNavigation() {
  if (!dom.previewPane) return;

  // Touch Start
  dom.previewPane.addEventListener(
    'touchstart',
    (e) => {
      if (e.touches.length !== 1 || isTransitioning) return;
      if (isModalActive()) return;

      // Ensure preview is active and visible
      if (dom.previewEmpty && dom.previewEmpty.style.display !== 'none') return;
      if (dom.previewContent && dom.previewContent.style.display === 'none') return;

      // On mobile, only swipe when viewing preview
      if (
        window.innerWidth <= 768 &&
        dom.appContainer &&
        !dom.appContainer.classList.contains('mobile-show-preview')
      ) {
        return;
      }

      // Do not intercept action buttons or links
      if (e.target.closest('button') || e.target.closest('a') || e.target.closest('.preview-actions')) {
        return;
      }

      // Do not intercept scrollable tables, pre code blocks, or mermaid diagrams
      if (isInsideHorizontalScrollable(e.target)) return;

      startX = e.touches[0].clientX;
      startY = e.touches[0].clientY;
      startTime = Date.now();
      isSwiping = false;
      isLockedVertical = false;
      currentDeltaX = 0;
    },
    { passive: true }
  );

  // Touch Move
  dom.previewPane.addEventListener(
    'touchmove',
    (e) => {
      if (e.touches.length !== 1 || isTransitioning || isLockedVertical) return;
      if (!dom.previewContent) return;

      const curX = e.touches[0].clientX;
      const curY = e.touches[0].clientY;
      const dx = curX - startX;
      const dy = curY - startY;

      // 1. Direction lock detection
      if (!isSwiping) {
        if (Math.abs(dy) > Math.abs(dx) && Math.abs(dy) > 7) {
          isLockedVertical = true;
          return;
        }
        if (Math.abs(dx) > Math.abs(dy) * 1.2 && Math.abs(dx) > 7) {
          isSwiping = true;
          dom.previewContent.style.willChange = 'transform, opacity';
          dom.previewContent.style.transition = 'none';
        }
      }

      // 2. Active 1:1 Horizontal Drag
      if (isSwiping) {
        if (e.cancelable) e.preventDefault();

        // Boundary resistance (rubber-band) when no adjacent item exists
        if (dx > 0) {
          const hasPrev = canNavigate(-1);
          currentDeltaX = hasPrev ? dx : dx * 0.22;
        } else {
          const hasNext = canNavigate(1);
          currentDeltaX = hasNext ? dx : dx * 0.22;
        }

        // Apply real-time 1:1 transform and subtle fade
        dom.previewContent.style.transform = `translate3d(${currentDeltaX}px, 0, 0)`;
        const opacity = Math.max(0.68, 1 - Math.abs(currentDeltaX) / (window.innerWidth * 1.6));
        dom.previewContent.style.opacity = `${opacity}`;

        // Update floating swipe direction indicator
        if (Math.abs(currentDeltaX) >= 24) {
          const dir = currentDeltaX < 0 ? 1 : -1;
          const hasItem = canNavigate(dir);
          const title = getAdjacentTitle(dir);
          if (hasItem && title) {
            showIndicator(dir, title, false);
          } else {
            showIndicator(dir, dir === 1 ? 'No next item' : 'No previous item', true);
          }
        } else {
          hideIndicator();
        }
      }
    },
    { passive: false }
  );

  // Touch End
  const handleTouchEnd = () => {
    if (!isSwiping || isTransitioning) {
      isSwiping = false;
      return;
    }
    isSwiping = false;
    hideIndicator();

    const absDx = Math.abs(currentDeltaX);
    const dt = Math.max(1, Date.now() - startTime);
    const velocity = absDx / dt;
    const isFlick = velocity > 0.42 && absDx > 30;
    const isDrag = absDx > Math.min(window.innerWidth * 0.22, 85);
    const dir = currentDeltaX < 0 ? 1 : -1;

    if ((isFlick || isDrag) && canNavigate(dir)) {
      performSwipeTransition(dir);
    } else {
      resetToCenter();
    }
  };

  dom.previewPane.addEventListener('touchend', handleTouchEnd, { passive: true });
  dom.previewPane.addEventListener('touchcancel', handleTouchEnd, { passive: true });

  // Wheel / Trackpad Two-Finger Horizontal Scroll
  dom.previewPane.addEventListener(
    'wheel',
    (e) => {
      // Disregard vertical scrolling gestures
      if (Math.abs(e.deltaY) >= Math.abs(e.deltaX) * 0.8) {
        wheelDeltaX = 0;
        return;
      }
      if (Math.abs(e.deltaX) < 14) return;
      if (isTransitioning || wheelCooldown || isModalActive()) return;

      // Disregard when preview is empty or inside scrollable table/code
      if (dom.previewEmpty && dom.previewEmpty.style.display !== 'none') return;
      if (dom.previewContent && dom.previewContent.style.display === 'none') return;
      if (isInsideHorizontalScrollable(e.target)) return;

      wheelDeltaX += e.deltaX;
      clearTimeout(wheelTimer);
      wheelTimer = setTimeout(() => {
        wheelDeltaX = 0;
      }, 180);

      // Threshold for trackpad flick (deltaX > 0 is scroll right -> next item)
      if (Math.abs(wheelDeltaX) >= 42) {
        const dir = wheelDeltaX > 0 ? 1 : -1;
        wheelDeltaX = 0;
        if (canNavigate(dir)) {
          wheelCooldown = true;
          setTimeout(() => {
            wheelCooldown = false;
          }, 380);
          performSwipeTransition(dir);
        }
      }
    },
    { passive: true }
  );
}
