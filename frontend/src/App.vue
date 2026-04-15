<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'
import AppLayout from './components/AppLayout.vue'
import Toaster from './components/Toaster.vue'
import { RouterView } from 'vue-router'

const route = useRoute()
const isPublic = computed(() => !!route.meta.public)
</script>

<template>
  <Toaster />
  <RouterView v-if="isPublic" />
  <AppLayout v-else />
</template>
