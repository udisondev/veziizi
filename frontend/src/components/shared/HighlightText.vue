<script setup lang="ts">
interface Segment {
  text: string
  match: boolean
}

const props = defineProps<{
  text: string
  query: string
}>()

function getSegments(text: string, query: string): Segment[] {
  const trimmed = query.trim()
  if (!trimmed) return [{ text, match: false }]

  const escaped = trimmed.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const regex = new RegExp(escaped, 'gi')

  const segments: Segment[] = []
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ text: text.slice(lastIndex, match.index), match: false })
    }
    segments.push({ text: match[0], match: true })
    lastIndex = regex.lastIndex
  }

  if (lastIndex < text.length) {
    segments.push({ text: text.slice(lastIndex), match: false })
  }

  return segments.length ? segments : [{ text, match: false }]
}
</script>

<template>
  <span>
    <template v-for="(seg, i) in getSegments(text, query)" :key="i">
      <mark v-if="seg.match" class="bg-green-100 text-green-800 rounded-sm px-0 not-italic font-[inherit]">{{ seg.text }}</mark>
      <span v-else>{{ seg.text }}</span>
    </template>
  </span>
</template>
