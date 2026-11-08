<template>
  <div class="stream-view">
    <StreamCanvas />
  </div>
</template>

<script setup lang="ts">
import StreamCanvas from '../components/StreamCanvas.vue'
</script>

<style scoped>
.stream-view {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: #000;
}
</style>
