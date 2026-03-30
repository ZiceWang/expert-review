<template>
  <div class="app" :class="{ dark: isDark }">
    <header class="header">
      <h1>Agent Review</h1>
      <div class="header-right">
        <div class="connection-status" :class="connectionStatus">
          <span class="status-dot"></span>
        </div>
        <button @click="refresh" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          {{ loading ? '' : '↻' }}
        </button>
        <button @click="isDark = !isDark">
          {{ isDark ? '☀' : '☾' }}
        </button>
      </div>
    </header>

    <main class="main">
      <div class="review-list">
        <div class="search-box">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search..."
          />
          <button v-if="searchQuery" @click="searchQuery = ''" class="clear-btn">×</button>
        </div>

        <h2>Pending ({{ filteredPending.length }})</h2>
        <div v-if="filteredPending.length === 0 && !loading" class="empty">
          No pending
        </div>
        <div
          v-for="review in filteredPending"
          :key="review.id"
          class="review-item"
          :class="{ selected: selectedReview?.id === review.id, new: isNew(review) }"
          @click="selectReview(review)"
        >
          <div class="review-summary">{{ review.taskResult.summary || 'No summary' }}</div>
          <div class="review-meta">
            <span>{{ formatTime(review.createdAt) }}</span>
            <span v-if="isNew(review)" class="new-badge">NEW</span>
          </div>
        </div>

        <template v-if="filteredCompleted.length > 0">
          <h2 class="history-header">History ({{ filteredCompleted.length }})</h2>
          <div
            v-for="review in filteredCompleted"
            :key="review.id"
            class="review-item completed"
            :class="{ selected: selectedReview?.id === review.id }"
            @click="selectReview(review)"
          >
            <div class="review-summary">{{ review.taskResult.summary || 'No summary' }}</div>
            <div class="review-meta">
              <span>{{ formatTime(review.createdAt) }}</span>
              <span class="decision-badge" :class="review.result?.decision">
                {{ review.result?.decision === 'approve' ? '✓' : review.result?.decision === 'reject' ? '✗' : '↻' }}
              </span>
            </div>
          </div>
        </template>
      </div>

      <div class="review-detail">
        <ReviewPanel
          v-if="selectedReview"
          :review="selectedReview"
          :submitting="submitting"
          @submit="submitReview"
        />
        <div v-else class="no-selection">
          Select a review
        </div>
      </div>
    </main>

    <div class="fab" @click="toggleFab()">
      <div v-if="fabExpanded" class="fab-panel" @click.stop>
        <div v-if="quote" class="quote">
          <p class="quote-text">"{{ quote.q }}"</p>
          <p class="quote-author">— {{ quote.a }}</p>
        </div>
        <div v-else class="quote-loading">Loading...</div>
      </div>
      <svg class="fab-icon" :class="{ rotated: fabExpanded }" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="9"/>
        <path d="M12 8v8M8 12l4 4 4-4"/>
      </svg>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import ReviewPanel from './components/ReviewPanel.vue';

const reviews = ref([]);
const selectedReview = ref(null);
const loading = ref(false);
const submitting = ref(false);
const connected = ref(false);
const newReviewIds = ref(new Set());
const fabExpanded = ref(false);
const quote = ref(null);
let quoteCache = null;

const API = '/api';
const isDark = ref(false);
const searchQuery = ref('');

const pendingReviews = computed(() =>
  reviews.value.filter(r => r.status === 'pending')
);

const completedReviews = computed(() =>
  reviews.value.filter(r => r.status === 'completed')
);

const filteredPending = computed(() => {
  if (!searchQuery.value) return pendingReviews.value;
  const q = searchQuery.value.toLowerCase();
  return pendingReviews.value.filter(r =>
    r.taskResult.summary?.toLowerCase().includes(q) ||
    r.taskResult.details?.toLowerCase().includes(q)
  );
});

const filteredCompleted = computed(() => {
  if (!searchQuery.value) return completedReviews.value;
  const q = searchQuery.value.toLowerCase();
  return completedReviews.value.filter(r =>
    r.taskResult.summary?.toLowerCase().includes(q) ||
    r.taskResult.details?.toLowerCase().includes(q)
  );
});

const connectionStatus = computed(() => connected.value ? 'connected' : 'disconnected');

function isNew(review) {
  return newReviewIds.value.has(review.id);
}

async function fetchReviews() {
  loading.value = true;
  const prevPending = pendingReviews.value.map(r => r.id);
  const prevCompleted = completedReviews.value.map(r => r.id);

  try {
    const res = await fetch(`${API}/reviews`);
    if (res.ok) {
      connected.value = true;
      const data = await res.json();
      console.log('[Frontend] Fetched reviews:', data);

      // Merge: update existing items, add new items
      const existingMap = new Map(reviews.value.map(r => [r.id, r]));
      data.forEach(r => {
        existingMap.set(r.id, r);
        // Mark new pending items
        if (r.status === 'pending' && !prevPending.includes(r.id)) {
          newReviewIds.value.add(r.id);
          setTimeout(() => newReviewIds.value.delete(r.id), 3000);
        }
      });

      // Sort by createdAt descending
      reviews.value = Array.from(existingMap.values())
        .sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    } else {
      connected.value = false;
    }
  } catch (e) {
    connected.value = false;
  } finally {
    loading.value = false;
  }
}

function selectReview(review) {
  selectedReview.value = review;
  newReviewIds.value.delete(review.id);
}

async function submitReview({ decision, comments, identity }) {
  if (!selectedReview.value) return;

  console.log('[Frontend] Submitting:', { decision, comments, identity });

  submitting.value = true;
  try {
    const res = await fetch(`${API}/reviews/${selectedReview.value.id}/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ decision, comments, reviewedBy: identity })
    });
    const data = await res.json();
    console.log('[Frontend] Submit response:', data);
    selectedReview.value = null;
    await fetchReviews();
  } catch (e) {
    console.error('Failed to submit review:', e);
  } finally {
    submitting.value = false;
  }
}

async function prefetchQuote() {
  try {
    const res = await fetch('/quotes.json');
    const quotes = await res.json();
    const q = quotes[Math.floor(Math.random() * quotes.length)];
    quoteCache = { q: q.quote, a: q.author };
  } catch (e) {
    console.error('Failed to prefetch quote:', e);
  }
}

function toggleFab() {
  fabExpanded.value = !fabExpanded.value;
  if (fabExpanded.value) {
    quote.value = quoteCache;
    prefetchQuote();
  }
}

function formatTime(iso) {
  const date = new Date(iso);
  const now = new Date();
  const diff = Math.floor((now - date) / 1000);

  if (diff < 60) return 'Just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
  return date.toLocaleDateString();
}

let pollInterval;

onMounted(() => {
  fetchReviews();
  prefetchQuote();
  pollInterval = setInterval(fetchReviews, 5000);
});

onUnmounted(() => {
  clearInterval(pollInterval);
});
</script>

<style scoped>
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #fafafa;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  color: #333;
  --bg: #fff;
  --bg-alt: #fafafa;
  --text: #333;
  --text-muted: #888;
  --border: #eee;
}

.app.dark {
  background: #1a1a1a;
  color: #ddd;
  --bg: #252525;
  --bg-alt: #1a1a1a;
  --text: #ddd;
  --text-muted: #777;
  --border: #333;
}

.header {
  background: var(--bg);
  padding: 0.75rem 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid var(--border);
}

.header h1 {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.connection-status {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.connection-status.connected {
  background: #27ae60;
}

.connection-status.disconnected {
  background: #e74c3c;
}

.header button {
  background: none;
  border: 1px solid var(--border);
  color: var(--text);
  width: 28px;
  height: 28px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.header button:hover {
  background: var(--bg-alt);
}

.spinner {
  width: 12px;
  height: 12px;
  border: 2px solid var(--border);
  border-top-color: var(--text-muted);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.main {
  flex: 1;
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 1px;
  background: var(--border);
  overflow: hidden;
}

.review-list {
  background: var(--bg);
  overflow-y: auto;
  padding: 1rem;
}

.search-box {
  position: relative;
  margin-bottom: 1rem;
}

.search-box input {
  width: 100%;
  padding: 0.5rem 2rem 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
  color: var(--text);
  font-size: 0.85rem;
  font-family: inherit;
}

.search-box input:focus {
  outline: none;
  border-color: var(--text-muted);
}

.search-box input::placeholder {
  color: var(--text-muted);
}

.clear-btn {
  position: absolute;
  right: 0.5rem;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 1rem;
  padding: 0.25rem;
}

.review-list h2 {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  margin-bottom: 0.75rem;
}

.history-header {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border);
}

.empty {
  color: var(--text-muted);
  font-size: 0.85rem;
  padding: 0.5rem 0;
}

.review-item {
  padding: 0.75rem;
  border-radius: 6px;
  margin-bottom: 0.5rem;
  cursor: pointer;
  border: 1px solid transparent;
}

.review-item:hover {
  background: var(--bg-alt);
}

.review-item.selected {
  background: var(--bg-alt);
  border-color: var(--border);
}

.review-item.completed {
  opacity: 0.6;
}

.review-summary {
  font-size: 0.85rem;
  color: var(--text);
  margin-bottom: 0.25rem;
  line-height: 1.4;
}

.review-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.7rem;
  color: var(--text-muted);
}

.new-badge {
  background: #e74c3c;
  color: #fff;
  padding: 0.1rem 0.35rem;
  border-radius: 3px;
  font-size: 0.65rem;
  font-weight: 500;
}

.decision-badge {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.65rem;
}

.decision-badge.approved {
  background: #d4edda;
  color: #27ae60;
}

.decision-badge.reject {
  background: #f8d7da;
  color: #e74c3c;
}

.decision-badge.needs_revision {
  background: #fff3cd;
  color: #f39c12;
}

.review-detail {
  background: var(--bg);
  overflow-y: auto;
  padding: 1rem;
}

.no-selection {
  color: var(--text-muted);
  font-size: 0.85rem;
  padding: 2rem;
  text-align: center;
}

.fab {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--text, #333);
  color: var(--bg, #fff);
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  z-index: 1000;
}

.fab:hover {
  opacity: 0.9;
}

.fab-icon {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  transition: transform 0.3s;
}

.fab-icon.rotated {
  transform: translate(-50%, -50%) rotate(45deg);
}

.fab-panel {
  position: absolute;
  bottom: 60px;
  right: 0;
  width: 280px;
  padding: 1.25rem;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.quote {
  font-family: 'Georgia', 'Times New Roman', serif;
  color: var(--text);
}

.quote-text {
  font-size: 0.95rem;
  font-style: italic;
  line-height: 1.6;
  margin-bottom: 0.75rem;
  color: var(--text);
}

.quote-author {
  font-size: 0.8rem;
  text-align: right;
  color: var(--text-muted);
}

.quote-loading {
  font-size: 0.85rem;
  color: var(--text-muted);
  text-align: center;
  padding: 2rem 0;
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}
</style>
