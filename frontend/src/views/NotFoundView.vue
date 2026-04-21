<script setup lang="ts">
import { AppLink } from '@/components/ui/app-link'
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background">
    <div class="text-center">
      <div class="text-6xl mb-4">404</div>
      <h1 class="text-2xl font-bold text-foreground mb-2">Страница не найдена</h1>
      <p class="text-muted-foreground mb-6">Запрашиваемая страница не существует</p>
      <AppLink to="/">На главную</AppLink>
    </div>
  </div>
</template>
