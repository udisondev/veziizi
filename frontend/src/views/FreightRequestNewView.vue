<script setup lang="ts">
import FreightRequestWizard from '@/components/freight-request/FreightRequestWizard.vue'
import DetailPageHeader from '@/components/shared/DetailPageHeader.vue'
</script>

<template>
  <div>
    <DetailPageHeader :back-to="{ name: 'freight-requests' }" back-label="К списку заявок" />

    <div class="min-h-screen bg-gray-100 py-8 px-4">
      <div class="max-w-4xl mx-auto">
        <h1 class="text-2xl font-bold text-gray-900 mb-6">Новая заявка на перевозку</h1>
        <FreightRequestWizard />
      </div>
    </div>
  </div>
</template>
