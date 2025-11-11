// OpenTelemetry Target Allocator UI - Interactive Features

(function() {
  'use strict';

  // State Management
  const state = {
    autoRefresh: false,
    refreshInterval: 30000, // 30 seconds default
    refreshTimer: null,
    sortColumn: null,
    sortDirection: 'asc',
    searchQuery: '',
    currentPage: 1,
    itemsPerPage: 100
  };

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  function init() {
    console.log('Target Allocator UI initialized');

    // Initialize all features
    initSearch();
    initTableSorting();
    initAutoRefresh();
    initBackToTop();
    initPagination();
    loadStateFromURL();

    // Add keyboard shortcuts
    initKeyboardShortcuts();
  }

  // ==================== Search & Filter ====================

  function initSearch() {
    const searchInput = document.getElementById('search-input');
    if (!searchInput) return;

    // Debounce search input
    let searchTimeout;
    searchInput.addEventListener('input', function(e) {
      clearTimeout(searchTimeout);
      searchTimeout = setTimeout(() => {
        state.searchQuery = e.target.value.toLowerCase();
        filterTable();
        saveStateToURL();
      }, 300);
    });

    // Focus search with '/' key
    document.addEventListener('keydown', function(e) {
      if (e.key === '/' && !isInputFocused()) {
        e.preventDefault();
        searchInput.focus();
      }
    });
  }

  function filterTable() {
    const tables = document.querySelectorAll('table tbody');
    const query = state.searchQuery;

    tables.forEach(table => {
      const rows = table.querySelectorAll('tr');
      let visibleCount = 0;

      rows.forEach(row => {
        const text = row.textContent.toLowerCase();
        const matches = !query || text.includes(query);

        row.style.display = matches ? '' : 'none';
        if (matches) visibleCount++;
      });

      // Update empty state
      updateEmptyState(table, visibleCount);
    });

    updateResultsCount();
  }

  function updateEmptyState(tbody, visibleCount) {
    const existingEmpty = tbody.parentElement.querySelector('.empty-state-row');

    if (visibleCount === 0 && state.searchQuery) {
      if (!existingEmpty) {
        const colCount = tbody.querySelector('tr')?.children.length || 1;
        const emptyRow = document.createElement('tr');
        emptyRow.className = 'empty-state-row';
        emptyRow.innerHTML = `
          <td colspan="${colCount}" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
            <div class="empty-state-icon">🔍</div>
            <div class="empty-state-message">No results found for "${state.searchQuery}"</div>
            <div style="margin-top: 0.5rem; font-size: var(--font-size-sm);">
              Try a different search term
            </div>
          </td>
        `;
        tbody.appendChild(emptyRow);
      }
    } else if (existingEmpty) {
      existingEmpty.remove();
    }
  }

  function updateResultsCount() {
    const resultCount = document.getElementById('results-count');
    if (!resultCount) return;

    const tables = document.querySelectorAll('table tbody');
    let total = 0;
    let visible = 0;

    tables.forEach(table => {
      const rows = table.querySelectorAll('tr:not(.empty-state-row)');
      rows.forEach(row => {
        total++;
        if (row.style.display !== 'none') visible++;
      });
    });

    if (state.searchQuery) {
      resultCount.textContent = `Showing ${visible} of ${total} results`;
      resultCount.style.display = 'block';
    } else {
      resultCount.style.display = 'none';
    }
  }

  // ==================== Table Sorting ====================

  function initTableSorting() {
    const tables = document.querySelectorAll('table');

    tables.forEach(table => {
      const headers = table.querySelectorAll('th');

      headers.forEach((header, index) => {
        // Skip if header has no text or is explicitly marked as non-sortable
        if (!header.textContent.trim() || header.classList.contains('no-sort')) {
          return;
        }

        header.classList.add('sortable');
        header.style.cursor = 'pointer';

        header.addEventListener('click', () => {
          sortTable(table, index, header);
        });
      });
    });
  }

  function sortTable(table, columnIndex, header) {
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr')).filter(row =>
      !row.classList.contains('empty-state-row')
    );

    // Determine sort direction
    const currentSort = header.classList.contains('sorted-asc') ? 'asc' :
                       header.classList.contains('sorted-desc') ? 'desc' : null;

    let newDirection;
    if (currentSort === null) {
      newDirection = 'asc';
    } else if (currentSort === 'asc') {
      newDirection = 'desc';
    } else {
      newDirection = 'asc';
    }

    // Clear all sort indicators
    table.querySelectorAll('th').forEach(th => {
      th.classList.remove('sorted-asc', 'sorted-desc');
    });

    // Add sort indicator to current column
    header.classList.add(`sorted-${newDirection}`);

    // Sort rows
    rows.sort((a, b) => {
      const aCell = a.children[columnIndex];
      const bCell = b.children[columnIndex];

      if (!aCell || !bCell) return 0;

      // Get text content, preferring data attributes if available
      let aValue = aCell.dataset.sortValue || aCell.textContent.trim();
      let bValue = bCell.dataset.sortValue || bCell.textContent.trim();

      // Try to parse as numbers
      const aNum = parseFloat(aValue);
      const bNum = parseFloat(bValue);

      let comparison;
      if (!isNaN(aNum) && !isNaN(bNum)) {
        comparison = aNum - bNum;
      } else {
        comparison = aValue.localeCompare(bValue, undefined, { numeric: true });
      }

      return newDirection === 'asc' ? comparison : -comparison;
    });

    // Reorder DOM
    rows.forEach(row => tbody.appendChild(row));

    // Update state
    state.sortColumn = columnIndex;
    state.sortDirection = newDirection;
    saveStateToURL();
  }

  // ==================== Auto Refresh ====================

  function initAutoRefresh() {
    const toggle = document.getElementById('auto-refresh-toggle');
    if (!toggle) return;

    toggle.addEventListener('click', () => {
      state.autoRefresh = !state.autoRefresh;

      if (state.autoRefresh) {
        toggle.classList.add('active');
        toggle.innerHTML = `
          <span class="refresh-indicator"></span>
          Auto-refresh ON
        `;
        startAutoRefresh();
      } else {
        toggle.classList.remove('active');
        toggle.innerHTML = `
          <span class="refresh-indicator"></span>
          Auto-refresh OFF
        `;
        stopAutoRefresh();
      }

      saveStateToURL();
    });

    // Manual refresh button
    const manualRefresh = document.getElementById('manual-refresh');
    if (manualRefresh) {
      manualRefresh.addEventListener('click', () => {
        refreshPage();
      });
    }
  }

  function startAutoRefresh() {
    stopAutoRefresh(); // Clear any existing timer

    state.refreshTimer = setInterval(() => {
      refreshPage();
    }, state.refreshInterval);
  }

  function stopAutoRefresh() {
    if (state.refreshTimer) {
      clearInterval(state.refreshTimer);
      state.refreshTimer = null;
    }
  }

  function refreshPage() {
    // Show loading indicator
    showRefreshIndicator();

    // Preserve scroll position
    const scrollPos = window.scrollY;

    // Reload page with current hash
    window.location.reload();

    // Note: scroll position restoration happens in loadStateFromURL after page loads
  }

  function showRefreshIndicator() {
    const indicator = document.createElement('div');
    indicator.id = 'refresh-indicator';
    indicator.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: var(--primary-color);
      color: white;
      padding: 12px 20px;
      border-radius: var(--radius-md);
      box-shadow: var(--shadow-lg);
      z-index: 10000;
      display: flex;
      align-items: center;
      gap: 10px;
      animation: slideIn 0.3s ease-out;
    `;

    indicator.innerHTML = `
      <div class="spinner"></div>
      <span>Refreshing...</span>
    `;

    document.body.appendChild(indicator);
  }

  // ==================== Back to Top ====================

  function initBackToTop() {
    const button = document.createElement('button');
    button.className = 'back-to-top';
    button.innerHTML = '↑';
    button.setAttribute('aria-label', 'Back to top');
    button.title = 'Back to top';
    document.body.appendChild(button);

    // Show/hide based on scroll position
    window.addEventListener('scroll', () => {
      if (window.scrollY > 300) {
        button.classList.add('visible');
      } else {
        button.classList.remove('visible');
      }
    });

    // Scroll to top on click
    button.addEventListener('click', () => {
      window.scrollTo({
        top: 0,
        behavior: 'smooth'
      });
    });
  }

  // ==================== Pagination ====================

  function initPagination() {
    const paginationContainer = document.getElementById('pagination');
    if (!paginationContainer) return;

    // This is a placeholder for pagination functionality
    // Can be expanded based on specific needs
    updatePagination();
  }

  function updatePagination() {
    // Placeholder for pagination logic
    // Would need to be implemented based on backend support
  }

  // ==================== State Persistence ====================

  function saveStateToURL() {
    const params = new URLSearchParams();

    if (state.searchQuery) params.set('q', state.searchQuery);
    if (state.autoRefresh) params.set('refresh', '1');
    if (state.sortColumn !== null) {
      params.set('sort', state.sortColumn);
      params.set('dir', state.sortDirection);
    }

    const newHash = params.toString();
    if (newHash) {
      window.location.hash = newHash;
    } else {
      history.replaceState(null, '', window.location.pathname + window.location.search);
    }
  }

  function loadStateFromURL() {
    const params = new URLSearchParams(window.location.hash.substring(1));

    // Restore search
    const query = params.get('q');
    if (query) {
      const searchInput = document.getElementById('search-input');
      if (searchInput) {
        searchInput.value = query;
        state.searchQuery = query.toLowerCase();
        filterTable();
      }
    }

    // Restore auto-refresh
    if (params.get('refresh') === '1') {
      const toggle = document.getElementById('auto-refresh-toggle');
      if (toggle) {
        toggle.click(); // Trigger the toggle
      }
    }

    // Restore sort
    const sortCol = params.get('sort');
    const sortDir = params.get('dir');
    if (sortCol !== null && sortDir) {
      const table = document.querySelector('table');
      if (table) {
        const header = table.querySelectorAll('th')[parseInt(sortCol)];
        if (header) {
          state.sortColumn = parseInt(sortCol);
          state.sortDirection = sortDir;

          // Apply sort
          setTimeout(() => {
            if (sortDir === 'desc') {
              // Click twice to get desc
              header.click();
              setTimeout(() => header.click(), 10);
            } else {
              header.click();
            }
          }, 100);
        }
      }
    }
  }

  // ==================== Keyboard Shortcuts ====================

  function initKeyboardShortcuts() {
    document.addEventListener('keydown', function(e) {
      // Skip if user is typing in an input
      if (isInputFocused()) return;

      switch(e.key) {
        case 'r':
          if (e.ctrlKey || e.metaKey) return; // Don't override browser refresh
          e.preventDefault();
          refreshPage();
          break;

        case 'a':
          e.preventDefault();
          const toggle = document.getElementById('auto-refresh-toggle');
          if (toggle) toggle.click();
          break;

        case 'h':
          e.preventDefault();
          window.location.href = '/';
          break;

        case '?':
          e.preventDefault();
          showKeyboardShortcuts();
          break;
      }
    });
  }

  function showKeyboardShortcuts() {
    const existing = document.getElementById('shortcuts-modal');
    if (existing) {
      existing.remove();
      return;
    }

    const modal = document.createElement('div');
    modal.id = 'shortcuts-modal';
    modal.style.cssText = `
      position: fixed;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      background: white;
      padding: 2rem;
      border-radius: var(--radius-lg);
      box-shadow: var(--shadow-lg);
      z-index: 10001;
      max-width: 400px;
      width: 90%;
    `;

    modal.innerHTML = `
      <h3 style="margin-bottom: 1rem;">Keyboard Shortcuts</h3>
      <div style="display: grid; gap: 0.5rem;">
        <div style="display: flex; justify-content: space-between;">
          <kbd style="background: var(--bg-secondary); padding: 0.25rem 0.5rem; border-radius: 4px;">/</kbd>
          <span>Focus search</span>
        </div>
        <div style="display: flex; justify-content: space-between;">
          <kbd style="background: var(--bg-secondary); padding: 0.25rem 0.5rem; border-radius: 4px;">r</kbd>
          <span>Refresh page</span>
        </div>
        <div style="display: flex; justify-content: space-between;">
          <kbd style="background: var(--bg-secondary); padding: 0.25rem 0.5rem; border-radius: 4px;">a</kbd>
          <span>Toggle auto-refresh</span>
        </div>
        <div style="display: flex; justify-content: space-between;">
          <kbd style="background: var(--bg-secondary); padding: 0.25rem 0.5rem; border-radius: 4px;">h</kbd>
          <span>Go to home</span>
        </div>
        <div style="display: flex; justify-content: space-between;">
          <kbd style="background: var(--bg-secondary); padding: 0.25rem 0.5rem; border-radius: 4px;">?</kbd>
          <span>Show shortcuts</span>
        </div>
      </div>
      <button onclick="document.getElementById('shortcuts-modal').remove(); document.getElementById('shortcuts-overlay').remove();"
              style="margin-top: 1.5rem; width: 100%; padding: 0.5rem; background: var(--primary-color); color: white; border: none; border-radius: var(--radius-sm); cursor: pointer;">
        Close
      </button>
    `;

    const overlay = document.createElement('div');
    overlay.id = 'shortcuts-overlay';
    overlay.style.cssText = `
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.5);
      z-index: 10000;
    `;
    overlay.addEventListener('click', () => {
      modal.remove();
      overlay.remove();
    });

    document.body.appendChild(overlay);
    document.body.appendChild(modal);
  }

  // ==================== Utility Functions ====================

  function isInputFocused() {
    const activeElement = document.activeElement;
    return activeElement && (
      activeElement.tagName === 'INPUT' ||
      activeElement.tagName === 'TEXTAREA' ||
      activeElement.isContentEditable
    );
  }

  // ==================== Simple Charts (Canvas-based) ====================

  window.TAChart = {
    drawBarChart: function(canvasId, data, options = {}) {
      const canvas = document.getElementById(canvasId);
      if (!canvas) return;

      const ctx = canvas.getContext('2d');
      const width = canvas.width;
      const height = canvas.height;

      // Clear canvas
      ctx.clearRect(0, 0, width, height);

      if (!data || data.length === 0) {
        ctx.fillStyle = '#6C757D';
        ctx.font = '14px var(--font-family)';
        ctx.textAlign = 'center';
        ctx.fillText('No data available', width / 2, height / 2);
        return;
      }

      const maxValue = Math.max(...data.map(d => d.value));
      const barWidth = (width - 40) / data.length;
      const chartHeight = height - 60;

      // Draw bars
      data.forEach((item, index) => {
        const barHeight = (item.value / maxValue) * chartHeight;
        const x = 20 + index * barWidth + barWidth * 0.1;
        const y = height - 40 - barHeight;
        const barActualWidth = barWidth * 0.8;

        // Draw bar
        ctx.fillStyle = options.color || '#425CC7';
        ctx.fillRect(x, y, barActualWidth, barHeight);

        // Draw value on top
        ctx.fillStyle = '#212529';
        ctx.font = '12px var(--font-family)';
        ctx.textAlign = 'center';
        ctx.fillText(item.value, x + barActualWidth / 2, y - 5);

        // Draw label
        ctx.fillStyle = '#6C757D';
        ctx.font = '11px var(--font-family)';
        ctx.save();
        ctx.translate(x + barActualWidth / 2, height - 10);
        ctx.rotate(-Math.PI / 4);
        ctx.textAlign = 'right';
        ctx.fillText(item.label, 0, 0);
        ctx.restore();
      });
    },

    drawPieChart: function(canvasId, data, options = {}) {
      const canvas = document.getElementById(canvasId);
      if (!canvas) return;

      const ctx = canvas.getContext('2d');
      const width = canvas.width;
      const height = canvas.height;

      ctx.clearRect(0, 0, width, height);

      if (!data || data.length === 0) {
        ctx.fillStyle = '#6C757D';
        ctx.font = '14px var(--font-family)';
        ctx.textAlign = 'center';
        ctx.fillText('No data available', width / 2, height / 2);
        return;
      }

      const total = data.reduce((sum, item) => sum + item.value, 0);
      const centerX = width / 2;
      const centerY = height / 2 - 20;
      const radius = Math.min(width, height) / 3;

      const colors = options.colors || [
        '#425CC7', '#F5A800', '#34A853', '#EA4335', '#FBBC04',
        '#4285F4', '#9C27B0', '#00BCD4', '#FF5722', '#607D8B'
      ];

      let currentAngle = -Math.PI / 2;

      data.forEach((item, index) => {
        const sliceAngle = (item.value / total) * 2 * Math.PI;

        // Draw slice
        ctx.beginPath();
        ctx.moveTo(centerX, centerY);
        ctx.arc(centerX, centerY, radius, currentAngle, currentAngle + sliceAngle);
        ctx.closePath();
        ctx.fillStyle = colors[index % colors.length];
        ctx.fill();

        currentAngle += sliceAngle;
      });

      // Draw legend
      const legendX = 20;
      let legendY = height - 60;

      data.forEach((item, index) => {
        ctx.fillStyle = colors[index % colors.length];
        ctx.fillRect(legendX, legendY, 12, 12);

        ctx.fillStyle = '#212529';
        ctx.font = '11px var(--font-family)';
        ctx.textAlign = 'left';
        const percentage = ((item.value / total) * 100).toFixed(1);
        ctx.fillText(`${item.label}: ${item.value} (${percentage}%)`, legendX + 18, legendY + 10);

        legendY += 18;
      });
    }
  };

  // Expose API for external use
  window.TargetAllocatorUI = {
    refresh: refreshPage,
    search: function(query) {
      const input = document.getElementById('search-input');
      if (input) {
        input.value = query;
        input.dispatchEvent(new Event('input'));
      }
    },
    toggleAutoRefresh: function() {
      const toggle = document.getElementById('auto-refresh-toggle');
      if (toggle) toggle.click();
    },
    getState: function() {
      return { ...state };
    }
  };

})();
