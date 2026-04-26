<script setup lang="ts">
import logoUrl from '@/assets/logo.svg'
</script>

<template>
  <header class="bg-slate-900 border-b border-slate-800">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-center h-14">
        <img :src="logoUrl" alt="ВезиИзи" class="h-9 w-auto" />
      </div>
    </div>
  </header>
</template>
