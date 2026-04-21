<script setup lang="ts">
import FreightRequestWizard from '@/components/freight-request/FreightRequestWizard.vue'
import DetailPageHeader from '@/components/shared/DetailPageHeader.vue'
</script>

<template>
  <div>
    <DetailPageHeader :back-to="{ name: 'freight-requests' }" back-label="К списку заявок" />

    <div class="min-h-screen bg-white md:bg-background py-6">
      <div class="max-w-5xl mx-auto px-4">
        <FreightRequestWizard title="Новая заявка на перевозку" />
      </div>
    </div>
  </div>
</template>
