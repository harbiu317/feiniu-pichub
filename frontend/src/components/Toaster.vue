<script setup lang="ts">
import { useToastStore } from '@/stores/toast'
const store = useToastStore()
</script>

<template>
  <div class="toaster">
    <transition-group name="toast">
      <div
        v-for="t in store.items"
        :key="t.id"
        class="toast"
        :class="['toast-' + t.kind]"
        @click="store.remove(t.id)"
      >
        <span class="toast-icon">
          {{ t.kind === 'success' ? '✓' : t.kind === 'error' ? '✗' : t.kind === 'warning' ? '!' : 'i' }}
        </span>
        <span class="toast-msg">{{ t.message }}</span>
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toaster {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
  max-width: 420px;
}
.toast {
  pointer-events: auto;
  background: var(--bg);
  border: 1px solid var(--border-2);
  border-radius: var(--radius);
  padding: 10px 14px;
  font-size: 13px;
  color: var(--fg);
  box-shadow: var(--shadow-lg);
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  min-width: 260px;
}
.toast-icon {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 12px;
  flex-shrink: 0;
  color: #fff;
}
.toast-success .toast-icon { background: #0070F3; }
.toast-error   .toast-icon { background: var(--danger); }
.toast-warning .toast-icon { background: var(--warning); }
.toast-info    .toast-icon { background: var(--fg); }

.toast-success { border-left: 3px solid #0070F3; }
.toast-error   { border-left: 3px solid var(--danger); }
.toast-warning { border-left: 3px solid var(--warning); }
.toast-info    { border-left: 3px solid var(--fg); }

.toast-msg { flex: 1; word-break: break-all; }

.toast-enter-active,
.toast-leave-active { transition: all 0.25s ease; }
.toast-enter-from { opacity: 0; transform: translateX(30px); }
.toast-leave-to   { opacity: 0; transform: translateX(30px); }
</style>
