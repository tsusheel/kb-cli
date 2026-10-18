/**
 * KB-CLI Web Fuzzy Finder — Mermaid Interactive Pan & Zoom Controller
 * Provides native SVG vector transformation, cursor focal-point zooming,
 * pointer drag panning, touch pinch-to-zoom, and fullscreen capabilities.
 */

export function setupMermaidPanZoom(wrapper) {
  if (!wrapper || wrapper.dataset.panzoomInitialized) return;
  wrapper.dataset.panzoomInitialized = 'true';

  const viewport = wrapper.querySelector('.mermaid-viewport');
  const diag = wrapper.querySelector('.mermaid-diagram');
  const zoomBadge = wrapper.querySelector('.zoom-level-badge');
  const zoomInBtn = wrapper.querySelector('.zoom-in-btn');
  const zoomOutBtn = wrapper.querySelector('.zoom-out-btn');
  const resetBtn = wrapper.querySelector('.zoom-reset-btn');
  const fullscreenBtn = wrapper.querySelector('.fullscreen-btn');

  if (!viewport || !diag) return;

  const svg = diag.querySelector('svg');
  if (!svg) return;

  // 1. Read initial viewBox to obtain true vector boundaries
  let minX = 0;
  let minY = 0;
  let origW = 800;
  let origH = 400;

  const vb = svg.getAttribute('viewBox');
  if (vb) {
    const parts = vb.trim().split(/[\s,]+/).map(parseFloat);
    if (parts.length >= 4 && parts[2] > 0 && parts[3] > 0) {
      minX = parts[0] || 0;
      minY = parts[1] || 0;
      origW = parts[2];
      origH = parts[3];
    }
  } else {
    try {
      const bbox = svg.getBBox();
      if (bbox && bbox.width > 0 && bbox.height > 0) {
        minX = bbox.x || 0;
        minY = bbox.y || 0;
        origW = bbox.width;
        origH = bbox.height;
      }
    } catch (_) {}
  }

  // 2. Prepare SVG for native vector viewport
  svg.removeAttribute('viewBox');
  svg.removeAttribute('width');
  svg.removeAttribute('height');
  svg.style.maxWidth = 'none';
  svg.style.maxHeight = 'none';
  svg.style.width = '100%';
  svg.style.height = '100%';
  svg.style.display = 'block';
  svg.style.overflow = 'hidden';

  // 3. Wrap diagram elements into a dedicated SVG vector panzoom layer
  let panzoomLayer = svg.querySelector(':scope > g.mermaid-panzoom-layer');
  if (!panzoomLayer) {
    panzoomLayer = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    panzoomLayer.setAttribute('class', 'mermaid-panzoom-layer');

    const children = Array.from(svg.childNodes);
    children.forEach((child) => {
      const tag = child.tagName ? child.tagName.toLowerCase() : '';
      if (tag === 'style' || tag === 'defs' || tag === 'metadata') {
        return;
      }
      panzoomLayer.appendChild(child);
    });
    svg.appendChild(panzoomLayer);
  }

  let scale = 1;
  let tx = 0;
  let ty = 0;
  let isDragging = false;
  let startX = 0;
  let startY = 0;
  const MIN_SCALE = 0.2;
  const MAX_SCALE = 6.0;

  function applyTransform(smooth = false) {
    panzoomLayer.classList.toggle('smooth-transition', smooth);
    panzoomLayer.setAttribute('transform', `matrix(${scale}, 0, 0, ${scale}, ${tx}, ${ty})`);
    if (zoomBadge) {
      zoomBadge.textContent = `${Math.round(scale * 100)}%`;
    }
  }

  function clampPan() {
    const vRect = viewport.getBoundingClientRect();
    if (vRect.width <= 0 || vRect.height <= 0) return;
    const margin = 50;
    const diagramW = origW * scale;
    const diagramH = origH * scale;

    const minTx = margin - (minX * scale + diagramW);
    const maxTx = vRect.width - margin - minX * scale;
    const minTy = margin - (minY * scale + diagramH);
    const maxTy = vRect.height - margin - minY * scale;

    if (minTx <= maxTx) {
      tx = Math.min(Math.max(tx, minTx), maxTx);
    }
    if (minTy <= maxTy) {
      ty = Math.min(Math.max(ty, minTy), maxTy);
    }
  }

  function fitDiagram(smooth = false) {
    const vRect = viewport.getBoundingClientRect();
    if (vRect.width <= 0 || vRect.height <= 0) return;

    const availW = Math.max(vRect.width - 60, 100);
    const availH = Math.max(vRect.height - 60, 100);

    const ratioW = availW / origW;
    const ratioH = availH / origH;
    const fitScale = Math.min(ratioW, ratioH, 1.0);

    scale = Math.min(Math.max(fitScale, MIN_SCALE), 1.0);
    tx = (vRect.width - origW * scale) / 2 - minX * scale;
    ty = (vRect.height - origH * scale) / 2 - minY * scale;

    clampPan();
    applyTransform(smooth);
  }

  // Zoom centered strictly on a focal point (focalX, focalY) relative to viewport
  function zoomTowards(newTargetScale, focalX, focalY, smooth = false) {
    const clampedScale = Math.min(Math.max(newTargetScale, MIN_SCALE), MAX_SCALE);
    if (clampedScale === scale) return;

    // Mathematical focal point preservation:
    // The point under (focalX, focalY) remains stationary
    tx = focalX - (focalX - tx) * (clampedScale / scale);
    ty = focalY - (focalY - ty) * (clampedScale / scale);
    scale = clampedScale;

    clampPan();
    applyTransform(smooth);
  }

  // Mouse wheel zoom centered exactly on cursor position with smooth exponential curve
  viewport.addEventListener('wheel', (e) => {
    e.preventDefault();
    e.stopPropagation();

    const vRect = viewport.getBoundingClientRect();
    const cursorX = e.clientX - vRect.left;
    const cursorY = e.clientY - vRect.top;

    const deltaMultiplier = e.deltaMode === 1 ? 24 : (e.deltaMode === 2 ? 400 : 1);
    const delta = -e.deltaY * deltaMultiplier;
    const zoomFactor = Math.min(Math.max(Math.exp(delta * 0.0015), 0.75), 1.35);

    zoomTowards(scale * zoomFactor, cursorX, cursorY, false);
  }, { passive: false });

  // Pointer-based Panning (Mouse & Single Touch via Pointer Events)
  viewport.addEventListener('pointerdown', (e) => {
    if (e.button !== 0) return;
    isDragging = true;
    startX = e.clientX - tx;
    startY = e.clientY - ty;
    viewport.classList.add('is-dragging');
    panzoomLayer.classList.remove('smooth-transition');
    try { viewport.setPointerCapture(e.pointerId); } catch (_) {}
  });

  viewport.addEventListener('pointermove', (e) => {
    if (!isDragging) return;
    tx = e.clientX - startX;
    ty = e.clientY - startY;
    clampPan();
    applyTransform(false);
  });

  const stopDragging = (e) => {
    if (isDragging) {
      isDragging = false;
      viewport.classList.remove('is-dragging');
      try { viewport.releasePointerCapture(e.pointerId); } catch (_) {}
    }
  };

  viewport.addEventListener('pointerup', stopDragging);
  viewport.addEventListener('pointercancel', stopDragging);

  // Double-click to reset view
  viewport.addEventListener('dblclick', (e) => {
    e.preventDefault();
    fitDiagram(true);
  });

  // Toolbar Buttons
  if (zoomInBtn) {
    zoomInBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      const vRect = viewport.getBoundingClientRect();
      zoomTowards(scale * 1.25, vRect.width / 2, vRect.height / 2, true);
    });
  }

  if (zoomOutBtn) {
    zoomOutBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      const vRect = viewport.getBoundingClientRect();
      zoomTowards(scale * 0.8, vRect.width / 2, vRect.height / 2, true);
    });
  }

  if (resetBtn) {
    resetBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      fitDiagram(true);
    });
  }

  // Fullscreen toggle
  function toggleFullscreen() {
    const isFs = wrapper.classList.toggle('mermaid-fullscreen');
    if (fullscreenBtn) {
      fullscreenBtn.title = isFs ? 'Exit Fullscreen (Esc)' : 'Expand Fullscreen (⛶)';
    }
    setTimeout(() => {
      fitDiagram(false);
    }, 60);
  }

  if (fullscreenBtn) {
    fullscreenBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      toggleFullscreen();
    });
  }

  // Two-finger pinch-to-zoom on touch devices
  let initialPinchDist = 0;
  let initialPinchScale = 1;

  viewport.addEventListener('touchstart', (e) => {
    if (e.touches.length === 2) {
      isDragging = false;
      initialPinchDist = Math.hypot(
        e.touches[0].clientX - e.touches[1].clientX,
        e.touches[0].clientY - e.touches[1].clientY
      );
      initialPinchScale = scale;
    }
  }, { passive: true });

  viewport.addEventListener('touchmove', (e) => {
    if (e.touches.length === 2 && initialPinchDist > 0) {
      const currentDist = Math.hypot(
        e.touches[0].clientX - e.touches[1].clientX,
        e.touches[0].clientY - e.touches[1].clientY
      );
      const vRect = viewport.getBoundingClientRect();
      const midX = (e.touches[0].clientX + e.touches[1].clientX) / 2 - vRect.left;
      const midY = (e.touches[0].clientY + e.touches[1].clientY) / 2 - vRect.top;
      const factor = currentDist / initialPinchDist;
      zoomTowards(initialPinchScale * factor, midX, midY, false);
    }
  }, { passive: true });

  viewport.addEventListener('touchend', (e) => {
    if (e.touches.length < 2) {
      initialPinchDist = 0;
    }
  });

  // Esc key exits fullscreen
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && wrapper.classList.contains('mermaid-fullscreen')) {
      wrapper.classList.remove('mermaid-fullscreen');
      if (fullscreenBtn) fullscreenBtn.title = 'Expand Fullscreen (⛶)';
      setTimeout(() => fitDiagram(false), 60);
    }
  });

  // Dynamic resize handling
  let hasFitted = false;
  const ro = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const cr = entry.contentRect;
      if (cr.width > 0 && cr.height > 0) {
        if (!hasFitted) {
          hasFitted = true;
          fitDiagram(false);
        }
      }
    }
  });
  ro.observe(viewport);

  // Initial fit
  requestAnimationFrame(() => {
    fitDiagram(false);
  });
}
