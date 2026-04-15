<script setup lang="ts">
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { computed, onMounted, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { useMetaStore } from '@/stores/meta'
import { api } from '@/api'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const metaStore = useMetaStore()

const stats = ref<any>({})

onMounted(async () => {
  await userStore.fetchMe()
  await metaStore.load()
  try {
    stats.value = await api.stats()
  } catch {}
})

const pageTitle = computed(() => (route.meta.title as string) || 'PicHub')

function logout() {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="logo">
        <div class="logo-dot"></div>
        <span>PicHub</span>
      </div>
      <div class="nav-section">Workspace</div>
      <RouterLink to="/upload" class="nav-item" active-class="active">
        <svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12"/></svg>
        上传
      </RouterLink>
      <RouterLink to="/gallery" class="nav-item" active-class="active">
        <svg viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>
        图库
        <span class="badge" v-if="stats.total">{{ stats.total }}</span>
      </RouterLink>
      <RouterLink to="/albums" class="nav-item" active-class="active" v-if="userStore.isLoggedIn">
        <svg viewBox="0 0 24 24"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        相册
        <span class="badge" v-if="stats.album_count">{{ stats.album_count }}</span>
      </RouterLink>
      <RouterLink to="/tokens" class="nav-item" active-class="active" v-if="userStore.isLoggedIn">
        <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4"/></svg>
        API 令牌
      </RouterLink>
      <template v-if="userStore.isAdmin">
        <div class="nav-section">Admin</div>
        <RouterLink to="/admin" class="nav-item" active-class="active">
          <svg viewBox="0 0 24 24"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          管理后台
        </RouterLink>
      </template>

      <div class="sidebar-footer">
        <template v-if="userStore.isLoggedIn">
          <div class="user-chip">
            <div class="avatar">{{ userStore.user?.username?.[0]?.toUpperCase() || 'U' }}</div>
            <div class="user-meta">
              <div class="name">{{ userStore.user?.username }}</div>
              <div class="role">{{ userStore.user?.role }}</div>
            </div>
            <button class="icon-btn" @click="logout" title="退出">
              <svg viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/></svg>
            </button>
          </div>
        </template>
        <template v-else>
          <RouterLink to="/login" class="btn btn-primary btn-block">登录</RouterLink>
        </template>
        <div class="build-info" v-if="metaStore.loaded">
          <span class="storage-chip" :title="'当前存储: ' + metaStore.driverLabel()">
            {{ metaStore.driverIcon() }} {{ metaStore.driverLabel() }}
          </span>
          <span class="version">v{{ metaStore.version }}</span>
        </div>
      </div>
    </aside>

    <main class="main">
      <div class="topbar">
        <span class="breadcrumb">Workspace</span>
        <span class="breadcrumb">/</span>
        <h1>{{ pageTitle }}</h1>
      </div>
      <div class="content">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.app { display: grid; grid-template-columns: 220px 1fr; min-height: 100vh; }
.sidebar {
  background: var(--bg-2);
  border-right: 1px solid var(--border);
  padding: 16px 12px;
  display: flex; flex-direction: column; gap: 4px;
}
.logo {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px; margin-bottom: 16px;
  font-weight: 600; font-size: 14px;
}
.logo-dot { width: 18px; height: 18px; background: #000; clip-path: polygon(50% 0, 100% 100%, 0 100%); }
.nav-section { font-size: 11px; color: var(--fg-3); text-transform: uppercase; letter-spacing: 0.05em; padding: 12px 10px 4px; }
.nav-item {
  display: flex; align-items: center; gap: 10px;
  padding: 6px 10px; border-radius: var(--radius);
  color: var(--fg-2); font-size: 13px; cursor: pointer;
  text-decoration: none;
}
.nav-item:hover { background: var(--bg-3); color: var(--fg); }
.nav-item.active { background: var(--bg-3); color: var(--fg); }
.nav-item svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.5; flex-shrink: 0; }
.badge { margin-left: auto; background: var(--border-2); color: var(--fg-2); font-size: 11px; padding: 1px 6px; border-radius: 4px; }

.sidebar-footer { margin-top: auto; padding-top: 16px; border-top: 1px solid var(--border); }
.user-chip {
  display: flex; align-items: center; gap: 10px;
  padding: 8px;
  border-radius: var(--radius);
}
.user-chip:hover { background: var(--bg-3); }
.avatar {
  width: 28px; height: 28px; border-radius: 50%;
  background: var(--accent);
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 600; color: white;
}
.user-meta { flex: 1; min-width: 0; }
.user-meta .name { font-size: 13px; color: var(--fg); }
.user-meta .role { font-size: 11px; color: var(--fg-3); }
.icon-btn { background: none; border: none; color: var(--fg-3); padding: 4px; border-radius: 4px; }
.icon-btn:hover { background: var(--border); color: var(--fg); }
.icon-btn svg { width: 14px; height: 14px; stroke: currentColor; fill: none; stroke-width: 1.5; }

.build-info {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: var(--fg-3);
  gap: 6px;
}
.storage-chip {
  background: var(--bg-3);
  padding: 2px 8px;
  border-radius: 999px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.version { font-family: var(--font-mono); flex-shrink: 0; }

.main { overflow: auto; }
.topbar {
  height: 48px; border-bottom: 1px solid var(--border);
  display: flex; align-items: center;
  padding: 0 24px; gap: 12px;
  position: sticky; top: 0; background: var(--bg); z-index: 10;
}
.breadcrumb { color: var(--fg-3); font-size: 13px; }
.topbar h1 { font-size: 14px; font-weight: 500; margin: 0; }
.content { padding: 24px; max-width: 1400px; margin: 0 auto; width: 100%; }

.btn {
  background: var(--bg-3);
  border: 1px solid var(--border-2);
  color: var(--fg);
  padding: 6px 12px; border-radius: var(--radius);
  font-size: 13px; font-weight: 500;
  cursor: pointer; transition: all var(--dur) var(--ease);
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  text-decoration: none;
}
.btn:hover { background: var(--border); border-color: var(--fg-3); }
.btn-primary { background: var(--accent); border-color: var(--accent); color: white; }
.btn-primary:hover { background: var(--accent-hover); border-color: var(--accent-hover); }
.btn-block { width: 100%; }
</style>
