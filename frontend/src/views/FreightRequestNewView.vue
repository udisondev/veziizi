<script setup lang="ts">
import FreightRequestWizard from '@/components/freight-request/FreightRequestWizard.vue'
</script>

<template>
  <div class="min-h-screen bg-gray-100 py-8 px-4">
    <div class="max-w-4xl mx-auto">
      <div class="mb-6">
        <router-link
          to="/"
          class="text-blue-600 hover:text-blue-700 text-sm flex items-center gap-1"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
          </svg>
          К списку заявок
        </router-link>
        <h1 class="text-2xl font-bold text-gray-900 mt-2">Новая заявка на перевозку</h1>
      </div>

      <FreightRequestWizard />
    </div>
  </div>
</template>
