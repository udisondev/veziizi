<script setup lang="ts">
import { AppLink } from '@/components/ui/app-link'
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background">
    <div class="text-center">
      <div class="text-6xl mb-4">🚫</div>
      <h1 class="text-2xl font-bold text-foreground mb-2">Доступ запрещён</h1>
      <p class="text-muted-foreground mb-6">У вас нет прав для просмотра этой страницы</p>
      <AppLink to="/">На главную</AppLink>
    </div>
  </div>
</template>
