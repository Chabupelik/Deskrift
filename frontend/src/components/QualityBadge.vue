<template>
  <div class="quality-badge">{{ stream.quality.toUpperCase() }}</div>
</template>

<script setup lang="ts">
import { useStreamStore } from '../stores/stream'
const stream = useStreamStore()
</script>

<style scoped>
.quality-badge {
  position: fixed;
  top: 8px;
  right: 8px;
  background: rgba(0,0,0,0.5);
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  pointer-events: none;
}
</style>

