import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api'

const driverLabels: Record<string, string> = {
  local: '本地磁盘',
  s3: '通用 S3',
  qiniu: '七牛云 Kodo',
  aliyun: '阿里云 OSS',
  tencent: '腾讯云 COS',
}

const driverIcons: Record<string, string> = {
  local: '💾',
  s3: '☁️',
  qiniu: '📦',
  aliyun: '🔶',
  tencent: '🔷',
}

export const useMetaStore = defineStore('meta', () => {
  const version = ref('')
  const storageDriver = ref('local')
  const loaded = ref(false)

  async function load() {
    try {
      const r = await api.meta()
      version.value = r.version
      storageDriver.value = r.storage_driver
      loaded.value = true
    } catch {}
  }

  function driverLabel(d: string = storageDriver.value) {
    return driverLabels[d] || d
  }

  function driverIcon(d: string = storageDriver.value) {
    return driverIcons[d] || '📁'
  }

  return { version, storageDriver, loaded, load, driverLabel, driverIcon }
})
