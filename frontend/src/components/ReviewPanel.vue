<template>
  <div class="review-panel">
    <div class="task-info">
      <div class="info-row">
        <span class="label">Summary</span>
        <span class="value">{{ review.taskResult.summary }}</span>
      </div>
      <div class="info-row">
        <span class="label">Created</span>
        <span class="value">{{ formatTime(review.createdAt) }}</span>
      </div>
      <div class="details" v-if="review.taskResult.details">
        <span class="label">Details</span>
        <pre>{{ review.taskResult.details }}</pre>
      </div>
    </div>

    <div class="review-result" v-if="review.status === 'completed'">
      <div class="result-header">
        <span class="decision-value" :class="review.result?.decision">
          {{ review.result?.decision === 'approve' ? '✓ Approved' :
             review.result?.decision === 'reject' ? '✗ Rejected' : '↻ Needs Revision' }}
        </span>
        <span class="result-meta">{{ review.result?.reviewedBy }} · {{ formatTime(review.result?.reviewedAt) }}</span>
      </div>
      <div class="comments" v-if="review.result?.comments">
        {{ review.result.comments }}
      </div>
    </div>

    <div class="review-form" v-else>
      <div class="form-group">
        <label>Decision</label>
        <div class="decision-btns">
          <button
            v-for="opt in decisions"
            :key="opt.value"
            :class="['decision-btn', opt.value, { selected: form.decision === opt.value }]"
            @click="form.decision = opt.value"
            :disabled="submitting"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
      <div class="form-group">
        <label>Comments</label>
        <textarea
          v-model="form.comments"
          placeholder="Enter comments..."
          rows="4"
          :disabled="submitting"
        ></textarea>
      </div>
      <div class="form-group identity-group">
        <label>Identity</label>
        <div class="identity-row">
          <input
            v-model="form.identity"
            type="text"
            placeholder="Anonymous"
            :disabled="submitting"
          />
          <button
            class="random-btn"
            @click="fillRandom"
            :disabled="submitting"
          >
            ?
          </button>
        </div>
      </div>
      <button
        class="submit-btn"
        :class="{ [form.decision]: true, loading: submitting }"
        @click="submit"
        :disabled="submitting"
      >
        {{ submitting ? 'Submitting...' : 'Submit Review' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue';

const props = defineProps({
  review: { type: Object, required: true },
  submitting: { type: Boolean, default: false }
});

const emit = defineEmits(['submit']);

const decisions = [
  { value: 'approve', label: 'Approve' },
  { value: 'needs_revision', label: 'Revision' },
  { value: 'reject', label: 'Reject' }
];

const modelPool = [
  'gpt-5.4', 'gpt-5.1', 'gpt-4.5',
  'claude-opus-4.6', 'claude-sonnet-4.6', 'claude-4.5',
  'gemini-3-pro', 'gemini-3.1-pro', 'gemini-3-ultra',
  'grok-4.20', 'grok-4.1', 'grok-3',
  'deepseek-v4', 'qwen-4-super', 'mistral-large-4'
];

const form = reactive({
  decision: 'approve',
  comments: '',
  identity: ''
});

function fillRandom() {
  form.identity = getRandomModel();
}

function getRandomModel() {
  return modelPool[Math.floor(Math.random() * modelPool.length)];
}

function formatTime(iso) {
  if (!iso) return '-';
  return new Date(iso).toLocaleString();
}

function submit() {
  const identity = form.identity.trim() || 'anonymous';
  console.log('[ReviewPanel] Emitting:', { decision: form.decision, comments: form.comments, identity });
  emit('submit', { decision: form.decision, comments: form.comments, identity });
}
</script>

<style scoped>
.review-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  color: var(--text, #333);
}

.task-info {
  background: var(--bg-alt, #fafafa);
  padding: 1rem;
  border-radius: 8px;
}

.info-row {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 0.5rem;
  font-size: 0.85rem;
}

.label {
  color: var(--text-muted, #888);
  min-width: 60px;
  font-size: 0.75rem;
}

.value {
  color: var(--text, #333);
}

.details {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--border, #eee);
}

.details .label {
  display: block;
  margin-bottom: 0.25rem;
}

.details pre {
  background: var(--bg, #fff);
  padding: 0.75rem;
  border-radius: 6px;
  font-size: 0.8rem;
  line-height: 1.5;
  color: var(--text, #555);
  white-space: pre-wrap;
  border: 1px solid var(--border, #eee);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.review-result {
  padding: 1rem;
  background: var(--bg-alt, #fafafa);
  border-radius: 8px;
  border: 1px solid var(--border, #eee);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.decision-value {
  font-weight: 600;
  font-size: 0.9rem;
}

.decision-value.approved { color: #27ae60; }
.decision-value.reject { color: #e74c3c; }
.decision-value.needs_revision { color: #f39c12; }

.result-meta {
  font-size: 0.75rem;
  color: var(--text-muted, #999);
}

.comments {
  font-size: 0.85rem;
  color: var(--text, #555);
  line-height: 1.5;
  padding: 0.75rem;
  background: var(--bg, #fff);
  border-radius: 6px;
  border: 1px solid var(--border, #eee);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.review-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-group label {
  font-size: 0.75rem;
  color: var(--text-muted, #888);
  font-weight: 500;
}

.identity-group {
  margin-top: 0.25rem;
}

.identity-row {
  display: flex;
  gap: 0.5rem;
}

.identity-row input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border, #ddd);
  border-radius: 6px;
  background: var(--bg, #fff);
  color: var(--text, #333);
  font-size: 0.85rem;
  font-family: inherit;
}

.identity-row input:focus {
  outline: none;
  border-color: var(--text-muted, #999);
}

.identity-row input::placeholder {
  color: var(--text-muted, #888);
}

.identity-row input:disabled {
  background: var(--bg-alt, #fafafa);
  cursor: not-allowed;
}

.random-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border, #ddd);
  border-radius: 6px;
  background: var(--bg, #fff);
  color: var(--text-muted, #888);
  cursor: pointer;
  font-size: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.random-btn:hover:not(:disabled) {
  background: var(--bg-alt, #fafafa);
}

.decision-btns {
  display: flex;
  gap: 0.5rem;
}

.decision-btn {
  flex: 1;
  padding: 0.6rem;
  border: 1px solid var(--border, #ddd);
  border-radius: 6px;
  background: var(--bg, #fff);
  color: var(--text, #333);
  cursor: pointer;
  font-size: 0.85rem;
  font-weight: 500;
  transition: all 0.15s;
}

.decision-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.decision-btn.approve.selected {
  border-color: #27ae60;
  background: #e8f8f0;
  color: #27ae60;
}

.decision-btn.needs_revision.selected {
  border-color: #f39c12;
  background: #fef9e7;
  color: #f39c12;
}

.decision-btn.reject.selected {
  border-color: #e74c3c;
  background: #fdedec;
  color: #e74c3c;
}

textarea {
  padding: 0.75rem;
  border: 1px solid var(--border, #ddd);
  border-radius: 6px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 0.85rem;
  resize: vertical;
  line-height: 1.5;
  background: var(--bg, #fff);
  color: var(--text, #333);
}

textarea:focus {
  outline: none;
  border-color: var(--text-muted, #999);
}

textarea:disabled {
  background: var(--bg-alt, #fafafa);
  cursor: not-allowed;
}

.submit-btn {
  padding: 0.75rem;
  border: none;
  border-radius: 6px;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  color: #fff;
}

.submit-btn.approve { background: #27ae60; }
.submit-btn.needs_revision { background: #f39c12; }
.submit-btn.reject { background: #e74c3c; }

.submit-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.submit-btn.loading {
  background: #aaa;
}
</style>
