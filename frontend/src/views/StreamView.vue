<template>
  <div class="stream-view">
    <StreamCanvas />
    <InputCapture />
    <QualityBadge />
  </div>
</template>

<script setup lang="ts">
import StreamCanvas from '../components/StreamCanvas.vue'
import InputCapture from '../components/InputCapture.vue'
import QualityBadge from '../components/QualityBadge.vue'
</script>

<style scoped>
.stream-view {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>

