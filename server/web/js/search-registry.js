/**
 * KB-CLI Web Fuzzy Finder — Search Registry & Smart Query Engine
 * Modular registry supporting structured token queries (#tags, @types, is:status, +area),
 * plugin-contributed search qualifiers/triggers, and live autocomplete suggestion routing.
 */

import { state } from './state.js';

class SearchRegistry {
  constructor() {
    this.qualifiers = new Map();
    this.triggers = new Map();
    this.types = new Map();
    this.statuses = new Map();
    this.pluginRegistrations = new Map(); // pluginId -> { qualifiers: Set, triggers: Set, types: Set }

    this.registerBuiltins();
  }

  registerBuiltins() {
    // 1. Built-in Item Types for '@' trigger
    const builtinTypes = [
      { id: 'note', label: 'note', description: 'Standard note', badgeClass: 'badge-note' },
      { id: 'todo', label: 'todo', description: 'Actionable task / todo', badgeClass: 'badge-todo' },
      { id: 'project', label: 'project', description: 'Project or milestone', badgeClass: 'badge-project' },
      { id: 'idea', label: 'idea', description: 'Concept or spark', badgeClass: 'badge-idea' },
      { id: 'decision', label: 'decision', description: 'Architecture or decision record', badgeClass: 'badge-decision' },
      { id: 'concept', label: 'concept', description: 'Knowledge base concept', badgeClass: 'badge-concept' },
      { id: 'til', label: 'til', description: 'Today I Learned snippet', badgeClass: 'badge-til' },
      { id: 'resource', label: 'resource', description: 'Reference or external bookmark', badgeClass: 'badge-resource' },
      { id: 'log', label: 'log', description: 'Daily log timeline entry', badgeClass: 'badge-log' },
      { id: 'doc', label: 'doc', description: 'System documentation page', badgeClass: 'badge-doc' },
    ];
    for (const t of builtinTypes) {
      this.types.set(t.id, t);
    }

    // 2. Built-in Statuses for 'is:' / 'status:'
    const builtinStatuses = [
      { id: 'active', label: 'active', description: 'Active items' },
      { id: 'in-progress', label: 'in-progress', description: 'Currently working on' },
      { id: 'completed', label: 'completed', description: 'Finished or checked off' },
      { id: 'raw', label: 'raw', description: 'Unrefined draft entry' },
      { id: 'refined', label: 'refined', description: 'Reviewed & structured note' },
      { id: 'archived', label: 'archived', description: 'Archived notes' },
    ];
    for (const s of builtinStatuses) {
      this.statuses.set(s.id, s);
    }

    // 3. Built-in Triggers
    // '#' -> Tags Trigger
    this.registerTrigger({
      trigger: '#',
      category: 'Tags',
      provideSuggestions: (query) => {
        const queryLower = (query || '').toLowerCase();
        // Collect tags from all items with counts
        const tagCounts = new Map();
        for (const item of state.allItems || []) {
          if (Array.isArray(item.tags)) {
            for (const tag of item.tags) {
              const count = tagCounts.get(tag) || 0;
              tagCounts.set(tag, count + 1);
            }
          }
        }

        const results = [];
        for (const [tag, count] of tagCounts.entries()) {
          if (!queryLower || tag.toLowerCase().includes(queryLower)) {
            results.push({
              text: `#${tag}`,
              label: `#${tag}`,
              count: `${count} note${count === 1 ? '' : 's'}`,
              category: 'Tags',
              priority: tag.toLowerCase().startsWith(queryLower) ? 10 : 1,
            });
          }
        }

        results.sort((a, b) => b.priority - a.priority || a.label.localeCompare(b.label));
        return results.slice(0, 8);
      },
    });

    // '@' -> Types Trigger
    this.registerTrigger({
      trigger: '@',
      category: 'Types',
      provideSuggestions: (query) => {
        const queryLower = (query || '').toLowerCase();
        const results = [];
        for (const t of this.types.values()) {
          if (!queryLower || t.label.toLowerCase().includes(queryLower) || t.id.includes(queryLower)) {
            results.push({
              text: `@${t.id}`,
              label: `@${t.label}`,
              description: t.description,
              badge: t.badgeClass || 'badge-type',
              category: 'Types',
              priority: t.id.startsWith(queryLower) ? 10 : 1,
            });
          }
        }
        results.sort((a, b) => b.priority - a.priority || a.label.localeCompare(b.label));
        return results;
      },
    });

    // '+' -> Areas Trigger
    this.registerTrigger({
      trigger: '+',
      category: 'Areas',
      provideSuggestions: (query) => {
        const queryLower = (query || '').toLowerCase();
        const areaCounts = new Map();
        for (const item of state.allItems || []) {
          if (item.area) {
            const count = areaCounts.get(item.area) || 0;
            areaCounts.set(item.area, count + 1);
          }
        }

        const results = [];
        for (const [area, count] of areaCounts.entries()) {
          if (!queryLower || area.toLowerCase().includes(queryLower)) {
            results.push({
              text: `+${area}`,
              label: `+${area}`,
              count: `${count} item${count === 1 ? '' : 's'}`,
              category: 'Areas',
              priority: area.toLowerCase().startsWith(queryLower) ? 10 : 1,
            });
          }
        }
        results.sort((a, b) => b.priority - a.priority || a.label.localeCompare(b.label));
        return results.slice(0, 6);
      },
    });

    // 'is:' / 'status:' Qualifier
    this.registerQualifier({
      name: 'is',
      prefix: 'is:',
      description: 'Filter notes by status (active, completed, in-progress, archived, etc.)',
      provideSuggestions: (query) => {
        const queryLower = (query || '').toLowerCase();
        const results = [];
        for (const s of this.statuses.values()) {
          if (!queryLower || s.id.includes(queryLower)) {
            results.push({
              text: `is:${s.id}`,
              label: `is:${s.id}`,
              description: s.description,
              category: 'Status',
            });
          }
        }
        return results;
      },
      filterItem: (item, val) => {
        const status = (item.status || '').toLowerCase();
        if (val === 'archived') return !!item.deleted_at;
        return status === val.toLowerCase();
      },
    });
  }

  // ──────────────────────────────────────────────────────────────────────────
  // Plugin Registration APIs
  // ──────────────────────────────────────────────────────────────────────────
  registerQualifier(def, pluginId = null) {
    if (!def || !def.prefix) return;
    const prefix = def.prefix.toLowerCase();
    this.qualifiers.set(prefix, { ...def, pluginId });

    if (pluginId) {
      if (!this.pluginRegistrations.has(pluginId)) {
        this.pluginRegistrations.set(pluginId, { qualifiers: new Set(), triggers: new Set(), types: new Set() });
      }
      this.pluginRegistrations.get(pluginId).qualifiers.add(prefix);
    }
  }

  registerType(def, pluginId = null) {
    if (!def || !def.id) return;
    this.types.set(def.id.toLowerCase(), { ...def, pluginId });

    if (pluginId) {
      if (!this.pluginRegistrations.has(pluginId)) {
        this.pluginRegistrations.set(pluginId, { qualifiers: new Set(), triggers: new Set(), types: new Set() });
      }
      this.pluginRegistrations.get(pluginId).types.add(def.id.toLowerCase());
    }
  }

  registerTrigger(def, pluginId = null) {
    if (!def || !def.trigger) return;
    this.triggers.set(def.trigger, { ...def, pluginId });

    if (pluginId) {
      if (!this.pluginRegistrations.has(pluginId)) {
        this.pluginRegistrations.set(pluginId, { qualifiers: new Set(), triggers: new Set(), types: new Set() });
      }
      this.pluginRegistrations.get(pluginId).triggers.add(def.trigger);
    }
  }

  unregisterPlugin(pluginId) {
    if (!this.pluginRegistrations.has(pluginId)) return;
    const reg = this.pluginRegistrations.get(pluginId);
    for (const prefix of reg.qualifiers) this.qualifiers.delete(prefix);
    for (const t of reg.triggers) this.triggers.delete(t);
    for (const id of reg.types) this.types.delete(id);
    this.pluginRegistrations.delete(pluginId);
  }

  // ──────────────────────────────────────────────────────────────────────────
  // Query Parsing & Filtering
  // ──────────────────────────────────────────────────────────────────────────
  /**
   * Parses raw user search input into structured filter tokens and clean fuzzy text.
   * e.g. '#backend #golang @todo is:active database migration'
   */
  parseQuery(queryString) {
    const raw = (queryString || '').trim();
    if (!raw) {
      return {
        raw: '',
        tags: [],
        excludedTags: [],
        types: [],
        excludedTypes: [],
        statuses: [],
        areas: [],
        customQualifiers: {},
        cleanText: '',
      };
    }

    const tokens = raw.split(/\s+/);
    const tags = [];
    const excludedTags = [];
    const types = [];
    const excludedTypes = [];
    const statuses = [];
    const areas = [];
    const customQualifiers = {};
    const textWords = [];

    for (const token of tokens) {
      if (!token) continue;

      // 1. Tag negation: -#tag
      if (token.startsWith('-#') && token.length > 2) {
        excludedTags.push(token.slice(2).toLowerCase());
        continue;
      }

      // 2. Tags: #tag
      if (token.startsWith('#') && token.length > 1) {
        tags.push(token.slice(1).toLowerCase());
        continue;
      }

      // 3. Type negation: -@type
      if (token.startsWith('-@') && token.length > 2) {
        excludedTypes.push(token.slice(2).toLowerCase());
        continue;
      }

      // 4. Types: @type or type:note
      if (token.startsWith('@') && token.length > 1) {
        types.push(token.slice(1).toLowerCase());
        continue;
      }
      if (token.toLowerCase().startsWith('type:') && token.length > 5) {
        types.push(token.slice(5).toLowerCase());
        continue;
      }

      // 5. Status: is:status or status:active
      if (token.toLowerCase().startsWith('is:') && token.length > 3) {
        statuses.push(token.slice(3).toLowerCase());
        continue;
      }
      if (token.toLowerCase().startsWith('status:') && token.length > 7) {
        statuses.push(token.slice(7).toLowerCase());
        continue;
      }

      // 6. Areas: +work or area:personal
      if (token.startsWith('+') && token.length > 1) {
        areas.push(token.slice(1).toLowerCase());
        continue;
      }
      if (token.toLowerCase().startsWith('area:') && token.length > 5) {
        areas.push(token.slice(5).toLowerCase());
        continue;
      }

      // 7. Check registered custom qualifiers
      let matchedCustom = false;
      for (const [prefix, qDef] of this.qualifiers.entries()) {
        if (token.toLowerCase().startsWith(prefix) && token.length > prefix.length) {
          const val = token.slice(prefix.length);
          if (!customQualifiers[qDef.name]) customQualifiers[qDef.name] = [];
          customQualifiers[qDef.name].push(val);
          matchedCustom = true;
          break;
        }
      }
      if (matchedCustom) continue;

      // 8. Otherwise treat as freeform search term
      textWords.push(token);
    }

    return {
      raw,
      tags,
      excludedTags,
      types,
      excludedTypes,
      statuses,
      areas,
      customQualifiers,
      cleanText: textWords.join(' ').trim(),
    };
  }

  /**
   * Tests whether an item satisfies all structured criteria in the parsed query.
   */
  filterItem(item, parsedQuery) {
    if (!item || !parsedQuery) return true;

    // 1. Tags matching: Item must include ALL specified tags (AND logic)
    if (parsedQuery.tags.length > 0) {
      if (!Array.isArray(item.tags) || item.tags.length === 0) return false;
      const itemTagsLower = item.tags.map((t) => t.toLowerCase());
      for (const reqTag of parsedQuery.tags) {
        if (!itemTagsLower.includes(reqTag)) return false;
      }
    }

    // 2. Tag negation: Item must NOT include any excluded tags
    if (parsedQuery.excludedTags.length > 0 && Array.isArray(item.tags)) {
      const itemTagsLower = item.tags.map((t) => t.toLowerCase());
      for (const exTag of parsedQuery.excludedTags) {
        if (itemTagsLower.includes(exTag)) return false;
      }
    }

    // 3. Type matching: Item must match at least one of the specified types
    if (parsedQuery.types.length > 0) {
      const itemType = (item.is_log ? 'log' : (item.is_doc ? 'doc' : (item.type || 'note'))).toLowerCase();
      if (!parsedQuery.types.includes(itemType)) return false;
    }

    // 4. Type exclusion: Item must not match any excluded types
    if (parsedQuery.excludedTypes.length > 0) {
      const itemType = (item.is_log ? 'log' : (item.is_doc ? 'doc' : (item.type || 'note'))).toLowerCase();
      if (parsedQuery.excludedTypes.includes(itemType)) return false;
    }

    // 5. Status matching: Item status must match one of specified statuses
    if (parsedQuery.statuses.length > 0) {
      const itemStatus = (item.status || '').toLowerCase();
      const isArchived = Boolean(item.deleted_at);

      const statusMatch = parsedQuery.statuses.some((reqStatus) => {
        if (reqStatus === 'archived') return isArchived;
        if (isArchived) return false;
        return itemStatus === reqStatus;
      });

      if (!statusMatch) return false;
    }

    // 6. Area matching
    if (parsedQuery.areas.length > 0) {
      const itemArea = (item.area || '').toLowerCase();
      if (!parsedQuery.areas.includes(itemArea)) return false;
    }

    // 7. Custom plugin qualifier matchers
    for (const [name, values] of Object.entries(parsedQuery.customQualifiers)) {
      for (const [prefix, qDef] of this.qualifiers.entries()) {
        if (qDef.name === name && typeof qDef.filterItem === 'function') {
          for (const val of values) {
            if (!qDef.filterItem(item, val)) return false;
          }
        }
      }
    }

    return true;
  }

  // ──────────────────────────────────────────────────────────────────────────
  // Autocomplete Suggestions API
  // ──────────────────────────────────────────────────────────────────────────
  /**
   * Given the active word under the caret, returns matching suggestions.
   */
  async getSuggestionsForWord(word) {
    if (!word) return [];

    // 1. Single-character triggers (#tag, @type, +area)
    const firstChar = word[0];
    if (this.triggers.has(firstChar)) {
      const triggerDef = this.triggers.get(firstChar);
      const query = word.slice(1);
      return triggerDef.provideSuggestions(query);
    }

    // 2. Prefixes (e.g. is:active, status:completed, or custom repo:)
    for (const [prefix, qDef] of this.qualifiers.entries()) {
      if (word.toLowerCase().startsWith(prefix)) {
        const query = word.slice(prefix.length);
        if (typeof qDef.provideSuggestions === 'function') {
          return qDef.provideSuggestions(query);
        }
      }
    }

    // 3. If word is partial 'is' or 'stat' -> suggest the qualifier prefix
    if ('is:'.startsWith(word.toLowerCase())) {
      return [{ text: 'is:', label: 'is:...', description: 'Filter by status (active, completed...)' }];
    }

    return [];
  }

  /**
   * Returns empty-state recommendations when clicking into an empty search bar.
   */
  getQuickPicks() {
    return [
      { text: '@todo', label: '@todo', description: 'Actionable tasks & todos', badge: 'badge-todo' },
      { text: 'is:active', label: 'is:active', description: 'Currently active items', category: 'Status' },
      { text: '@project', label: '@project', description: 'Active project notes', badge: 'badge-project' },
      { text: '@idea', label: '@idea', description: 'Sparks & concepts', badge: 'badge-idea' },
    ];
  }

  /**
   * Returns formatted syntax documentation items for the cheatsheet overlay.
   */
  getSyntaxDocumentation() {
    const list = [
      { prefix: '#tag', example: '#backend #golang', description: 'Compound tag filtering (matches all tags)' },
      { prefix: '-#tag', example: '#api -#deprecated', description: 'Exclude items with specified tag' },
      { prefix: '@type', example: '@todo, @project, @idea', description: 'Filter by note/entry type' },
      { prefix: 'is:status', example: 'is:active, is:completed', description: 'Filter by status' },
      { prefix: '+area', example: '+work, +personal', description: 'Filter by organizational area' },
      { prefix: 'word', example: 'auth migration', description: 'Fuzzy search title and markdown body' },
    ];

    // Add any plugin-contributed qualifiers
    for (const qDef of this.qualifiers.values()) {
      if (qDef.pluginId) {
        list.push({
          prefix: qDef.prefix,
          example: `${qDef.prefix}value`,
          description: qDef.description || 'Plugin filter',
          pluginId: qDef.pluginId,
        });
      }
    }

    return list;
  }
}

export const searchRegistry = new SearchRegistry();

