<script setup lang="ts">
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50">
    <div class="text-center">
      <div class="text-6xl mb-4">🚫</div>
      <h1 class="text-2xl font-bold text-gray-900 mb-2">Доступ запрещён</h1>
      <p class="text-gray-600 mb-6">У вас нет прав для просмотра этой страницы</p>
      <router-link to="/" class="text-blue-600 hover:text-blue-500">
        На главную
      </router-link>
    </div>
  </div>
</template>
