<script setup lang="ts">
const output = ref('')
const isStreaming = ref(false)

async function stream() {
  output.value = ''
  const response = await fetch('/api/streaming')
  const reader = response.body?.getReader()
  const decoder = new TextDecoder()

  while (reader) {
    const { value, done } = await reader.read()

    if(done) break;

    if (value) {
      output.value += decoder.decode(value, { stream: true })
    }
  }
}
</script>

<template>
  <div>
    <pre>{{ output }}</pre>
    <button :disabled="isStreaming" @click="stream">Call API</button>
  </div>
</template>

