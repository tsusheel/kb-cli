/**
 * KB-CLI Web Fuzzy Finder — Markdown Engine & Syntactical Enhancer
 * Handles full GFM-compatible parsing (headers, blockquotes, tables, task lists,
 * nested code blocks), Highlight.js syntax highlighting, and Mermaid diagrams.
 */

import { escapeHtml } from './utils.js';
import { setupMermaidPanZoom } from './mermaid.js';

/**
 * Initializes external library configurations (e.g. Mermaid theme settings).
 */
export function initMarkdownLibraries() {
  if (window.mermaid) {
    try {
      window.mermaid.initialize({
        startOnLoad: false,
        theme: 'dark',
        securityLevel: 'loose',
        fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
        themeVariables: {
          darkMode: true,
          background: '#0b0e14',
          primaryColor: '#1f6feb',
          primaryTextColor: '#c9d1d9',
          primaryBorderColor: '#30363d',
          lineColor: '#8b949e',
          secondaryColor: '#161b22',
          tertiaryColor: '#21262d',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif'
        }
      });
    } catch (e) {
      console.warn('Failed to initialize mermaid:', e);
    }
  }
}

/**
 * Parses markdown text into sanitized, rich HTML.
 */
export function renderMarkdown(text) {
  if (!text || !text.trim()) {
    return '<p><em>(no content)</em></p>';
  }

  // Step 1: Normalize line endings
  text = text.replace(/\r\n/g, '\n').replace(/\r/g, '\n');

  // Step 2: Extract code blocks (supporting optional indentation e.g. inside lists or blockquotes)
  const codeBlocks = [];
  text = text.replace(/(?:^|\n)([ \t]*)```([a-zA-Z0-9_+-]*)[ \t]*\n([\s\S]*?)\n[ \t]*```[ \t]*(?=\n|$)/g, (match, indent, lang, code) => {
    let cleanCode = code;
    if (indent) {
      const indentRe = new RegExp(`^${indent}`, 'gm');
      cleanCode = cleanCode.replace(indentRe, '');
    }
    const id = codeBlocks.length;
    codeBlocks.push({ lang: lang.trim(), rawCode: cleanCode, escapedCode: escapeHtml(cleanCode) });
    return `\n\n%%BLOCK_CODE_${id}%%\n\n`;
  });

  // Handle code blocks at EOF without trailing newline
  text = text.replace(/(?:^|\n)([ \t]*)```([a-zA-Z0-9_+-]*)[ \t]*\n([\s\S]*?)\n[ \t]*```[ \t]*$/g, (match, indent, lang, code) => {
    let cleanCode = code;
    if (indent) {
      const indentRe = new RegExp(`^${indent}`, 'gm');
      cleanCode = cleanCode.replace(indentRe, '');
    }
    const id = codeBlocks.length;
    codeBlocks.push({ lang: lang.trim(), rawCode: cleanCode, escapedCode: escapeHtml(cleanCode) });
    return `\n\n%%BLOCK_CODE_${id}%%\n\n`;
  });

  // Step 3: Extract inline code
  const inlineCodes = [];
  text = text.replace(/`([^`\n]+)`/g, (match, code) => {
    const id = inlineCodes.length;
    inlineCodes.push(escapeHtml(code));
    return `%%INLINE_CODE_${id}%%`;
  });

  // Inline formatting processor
  function parseInline(str) {
    if (!str) return '';

    let s = escapeHtml(str);

    // Links: [text](url)
    s = s.replace(/\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>');
    s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');

    // Auto links: https://...
    s = s.replace(/(^|[\s(])(https?:\/\/[^\s)<]+)/g, '$1<a href="$2" target="_blank" rel="noopener noreferrer">$2</a>');

    // Bold: **text** or __text__
    s = s.replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>');
    s = s.replace(/__([^_\n]+)__/g, '<strong>$1</strong>');

    // Italic: *text* or _text_
    s = s.replace(/\*([^*\n]+)\*/g, '<em>$1</em>');
    s = s.replace(/(^|[^\w])_([^_\n]+)_(?=[^\w]|$)/g, '$1<em>$2</em>');

    // Strikethrough: ~~text~~
    s = s.replace(/~~([^~\n]+)~~/g, '<del>$1</del>');

    return s;
  }

  // Block-level processor
  function parseBlocks(src) {
    const lines = src.split('\n');
    const out = [];
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];

      // 1. Code block token placeholder
      const codeBlockMatch = line.trim().match(/^%%BLOCK_CODE_(\d+)%%$/);
      if (codeBlockMatch) {
        const id = parseInt(codeBlockMatch[1], 10);
        const block = codeBlocks[id];
        if (block) {
          const rawLang = (block.lang || '').toLowerCase();
          if (rawLang === 'mermaid') {
            const safeRaw = escapeHtml(block.rawCode.trim());
            out.push(`
              <div class="mermaid-block-wrapper">
                <div class="mermaid-block-header">
                  <div class="mermaid-header-left">
                    <span class="mermaid-tag">
                      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg>
                      <span class="mermaid-tag-prefix">Mermaid </span><span>Diagram</span>
                    </span>
                    <span class="zoom-level-badge">100%</span>
                  </div>
                  <div class="mermaid-header-actions">
                    <button type="button" class="diagram-btn zoom-in-btn" title="Zoom In (+)" aria-label="Zoom In">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                    </button>
                    <button type="button" class="diagram-btn zoom-out-btn" title="Zoom Out (-)" aria-label="Zoom Out">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                    </button>
                    <button type="button" class="diagram-btn zoom-reset-btn" title="Reset View (Double-click)" aria-label="Reset View">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path><path d="M3 3v5h5"></path></svg>
                    </button>
                    <button type="button" class="diagram-btn fullscreen-btn" title="Expand Fullscreen (⛶)" aria-label="Fullscreen">
                      <svg class="fs-expand-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/></svg>
                      <svg class="fs-compress-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="display:none;"><path d="M4 14h6v6M20 10h-6V4M14 10l7-7M10 14l-7 7"/></svg>
                    </button>
                    <button type="button" class="code-copy-btn" title="Copy Diagram Source" aria-label="Copy Diagram Source">
                      <svg class="copy-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                      <svg class="check-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="display:none;"><polyline points="20 6 9 17 4 12"></polyline></svg>
                      <span class="copy-text">Copy</span>
                    </button>
                  </div>
                </div>
                <div class="mermaid-viewport" data-raw-code="${safeRaw}">
                  <div class="mermaid-diagram" data-raw-code="${safeRaw}">${safeRaw}</div>
                </div>
              </div>
            `);
          } else {
            const isBoxDrawing = /[┌┐└┘├┤┬┴┼─│═║╔╗╚╝╠╣╦╩╬━┃┏┓┗┛┣┫┳┻╋]/.test(block.rawCode) ||
              ['box', 'ui', 'card', 'ascii', 'diagram'].includes((block.lang || '').toLowerCase());
            const langDisplay = isBoxDrawing
              ? (block.lang && block.lang !== 'text' ? block.lang : 'ui layout')
              : (block.lang || 'code');
            const langClass = block.lang ? ` class="hljs language-${escapeHtml(block.lang)}"` : ' class="hljs"';
            const wrapperClass = isBoxDrawing
              ? 'code-block-wrapper box-drawing-wrapper'
              : 'code-block-wrapper';

            out.push(`
              <div class="${wrapperClass}">
                <div class="code-block-header">
                  <span class="code-lang-tag">${escapeHtml(langDisplay)}</span>
                  <button type="button" class="code-copy-btn" title="Copy" aria-label="Copy">
                    <svg class="copy-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                    <svg class="check-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none;"><polyline points="20 6 9 17 4 12"></polyline></svg>
                    <span class="copy-text">Copy</span>
                  </button>
                </div>
                <pre><code${langClass}>${block.escapedCode}</code></pre>
              </div>
            `);
          }
        }
        i++;
        continue;
      }

      // 2. Blank line
      if (line.trim() === '') {
        i++;
        continue;
      }

      // 3. Headers (# to ######)
      const headerMatch = line.match(/^(#{1,6})[ \t]+(.*)$/);
      if (headerMatch) {
        const level = headerMatch[1].length;
        const content = parseInline(headerMatch[2].trim());
        out.push(`<h${level}>${content}</h${level}>`);
        i++;
        continue;
      }

      // 4. Horizontal rule
      if (/^[ \t]*(?:[-*_][ \t]*){3,}$/.test(line)) {
        out.push('<hr>');
        i++;
        continue;
      }

      // 5. Blockquote (groups contiguous > lines)
      if (/^[ \t]*>/.test(line)) {
        const quoteLines = [];
        while (i < lines.length && /^[ \t]*>/.test(lines[i])) {
          quoteLines.push(lines[i].replace(/^[ \t]*>[ \t]?/, ''));
          i++;
        }
        const innerHtml = parseBlocks(quoteLines.join('\n'));
        out.push(`<blockquote>${innerHtml}</blockquote>`);
        continue;
      }

      // 6. Table (lines starting and ending with |)
      if (/^[ \t]*\|.+?\|[ \t]*$/.test(line)) {
        const tableLines = [];
        while (i < lines.length && /^[ \t]*\|.+?\|[ \t]*$/.test(lines[i])) {
          tableLines.push(lines[i].trim());
          i++;
        }
        if (tableLines.length >= 1) {
          const parseRow = (l) => l.slice(1, -1).split('|').map((c) => parseInline(c.trim()));
          let startIdx = 0;
          let hasHeaderDivider = false;
          if (tableLines.length >= 2 && /^\|(?:\s*:?-+:?\s*\|)+$/.test(tableLines[1])) {
            hasHeaderDivider = true;
          }

          let tHtml = '<table>';
          if (hasHeaderDivider) {
            const headers = parseRow(tableLines[0]);
            tHtml += '<thead><tr>' + headers.map((h) => `<th>${h}</th>`).join('') + '</tr></thead><tbody>';
            startIdx = 2;
          } else {
            tHtml += '<tbody>';
            startIdx = 0;
          }

          for (let r = startIdx; r < tableLines.length; r++) {
            if (/^\|(?:\s*:?-+:?\s*\|)+$/.test(tableLines[r])) continue;
            const cells = parseRow(tableLines[r]);
            tHtml += '<tr>' + cells.map((c) => `<td>${c}</td>`).join('') + '</tr>';
          }
          tHtml += '</tbody></table>';
          out.push(tHtml);
          continue;
        }
      }

      // 7. Task list or Unordered list
      if (/^[ \t]*[-*][ \t]/.test(line)) {
        const listItems = [];
        let isTaskList = false;
        while (i < lines.length && /^[ \t]*[-*][ \t]/.test(lines[i])) {
          const curLine = lines[i];
          const taskMatch = curLine.match(/^[ \t]*[-*][ \t]\[([ xX])\][ \t]+(.*)$/);
          if (taskMatch) {
            isTaskList = true;
            const checked = taskMatch[1].toLowerCase() === 'x';
            const content = parseInline(taskMatch[2].trim());
            listItems.push(
              `<li class="task-item"><input type="checkbox" ${checked ? 'checked' : ''} disabled> <span>${content}</span></li>`
            );
          } else {
            const content = parseInline(curLine.replace(/^[ \t]*[-*][ \t]+/, '').trim());
            listItems.push(`<li>${content}</li>`);
          }
          i++;
        }
        const listClass = isTaskList ? ' class="task-list"' : '';
        out.push(`<ul${listClass}>${listItems.join('')}</ul>`);
        continue;
      }

      // 8. Ordered list
      if (/^[ \t]*\d+\.[ \t]/.test(line)) {
        const listItems = [];
        while (i < lines.length && /^[ \t]*\d+\.[ \t]/.test(lines[i])) {
          const content = parseInline(lines[i].replace(/^[ \t]*\d+\.[ \t]+/, '').trim());
          listItems.push(`<li>${content}</li>`);
          i++;
        }
        out.push(`<ol>${listItems.join('')}</ol>`);
        continue;
      }

      // 9. Regular paragraph (group contiguous non-block lines)
      const paraLines = [];
      while (
        i < lines.length &&
        lines[i].trim() !== '' &&
        !/^(#{1,6})[ \t]+/.test(lines[i]) &&
        !/^[ \t]*>/.test(lines[i]) &&
        !/^[ \t]*\|.+?\|[ \t]*$/.test(lines[i]) &&
        !/^[ \t]*[-*][ \t]/.test(lines[i]) &&
        !/^[ \t]*\d+\.[ \t]/.test(lines[i]) &&
        !/^[ \t]*(?:[-*_][ \t]*){3,}$/.test(lines[i]) &&
        !/^%%BLOCK_CODE_/.test(lines[i].trim())
      ) {
        paraLines.push(parseInline(lines[i].trim()));
        i++;
      }
      if (paraLines.length > 0) {
        out.push(`<p>${paraLines.join('<br>')}</p>`);
      }
    }

    return out.join('\n');
  }

  // Step 4: Parse blocks
  let html = parseBlocks(text);

  // Step 5: Restore inline code
  html = html.replace(/%%INLINE_CODE_(\d+)%%/g, (match, id) => {
    const code = inlineCodes[parseInt(id, 10)] || '';
    return `<code>${code}</code>`;
  });

  return html;
}

/**
 * Applies syntax highlighting (Highlight.js) and diagram rendering (Mermaid.js)
 * to any newly injected markdown container.
 */
export async function applyPostMarkdownEnhancements(container) {
  if (!container) return;

  // 1. Syntax highlighting with Highlight.js
  if (window.hljs) {
    container.querySelectorAll('.code-block-wrapper pre code').forEach((block) => {
      const wrapper = block.closest('.code-block-wrapper');
      const langTag = wrapper ? wrapper.querySelector('.code-lang-tag') : null;
      const classes = Array.from(block.classList);
      const hasSpecificLang = classes.some((c) => c.startsWith('language-'));

      if (hasSpecificLang) {
        try {
          window.hljs.highlightElement(block);
        } catch (e) {
          console.debug('Highlight error:', e);
        }
      } else {
        // Auto-detect language
        const codeText = block.textContent;
        if (codeText && codeText.trim().length > 0) {
          try {
            const res = window.hljs.highlightAuto(codeText);
            if (res && res.value) {
              block.innerHTML = res.value;
              block.classList.add('hljs');
              if (res.language && langTag && (langTag.textContent === 'code' || !langTag.textContent)) {
                langTag.textContent = res.language;
              }
            }
          } catch (e) {
            try {
              window.hljs.highlightElement(block);
            } catch (e2) {}
          }
        }
      }
    });
  }

  // 2. Mermaid diagram rendering
  if (window.mermaid) {
    const diagrams = container.querySelectorAll('.mermaid-diagram:not([data-processed="true"])');
    for (let idx = 0; idx < diagrams.length; idx++) {
      const diag = diagrams[idx];
      const rawCode = diag.getAttribute('data-raw-code') || diag.textContent;
      const uniqueId = `mermaid-svg-${Date.now()}-${idx}-${Math.floor(Math.random() * 10000)}`;

      try {
        const { svg } = await window.mermaid.render(uniqueId, rawCode.trim());
        diag.innerHTML = svg;
        diag.setAttribute('data-processed', 'true');
        const wrapper = diag.closest('.mermaid-block-wrapper');
        if (wrapper) {
          setupMermaidPanZoom(wrapper);
        }
      } catch (err) {
        console.warn('Mermaid rendering failed for snippet:', err);
        diag.setAttribute('data-processed', 'error');
        diag.innerHTML = `
          <div class="mermaid-error">
            <div class="mermaid-error-title">⚠️ Diagram Render Error</div>
            <pre class="mermaid-error-raw"><code>${escapeHtml(rawCode.trim())}</code></pre>
          </div>
        `;
        const orphaned = document.getElementById(uniqueId) || document.getElementById(`d${uniqueId}`);
        if (orphaned) orphaned.remove();
      }
    }
  }
}
