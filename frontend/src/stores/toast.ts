import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastKind = 'success' | 'error' | 'info' | 'warning'

interface Toast {
  id: number
  kind: ToastKind
  message: string
}

let seq = 0

export const useToastStore = defineStore('toast', () => {
  const items = ref<Toast[]>([])

  function push(message: string, kind: ToastKind = 'info', duration = 3500) {
    const id = ++seq
    items.value.push({ id, kind, message })
    if (duration > 0) {
      setTimeout(() => remove(id), duration)
    }
    return id
  }

  function remove(id: number) {
    items.value = items.value.filter(t => t.id !== id)
  }

  return {
    items,
    push,
    remove,
    success: (m: string, d?: number) => push(m, 'success', d),
    error:   (m: string, d?: number) => push(m, 'error', d),
    info:    (m: string, d?: number) => push(m, 'info', d),
    warning: (m: string, d?: number) => push(m, 'warning', d),
  }
})
