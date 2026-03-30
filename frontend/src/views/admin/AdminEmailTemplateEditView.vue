<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { adminApi } from '@/api/admin'
import type { VariableSpec } from '@/types/admin'

// UI Components
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

// Shared Components
import { ErrorBanner } from '@/components/shared'

// Icons
import {
  ArrowLeft,
  Save,
  Eye,
  Code,
  FileText,
  Variable,
  Lock,
  Plus,
  Trash2,
} from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()

const isNew = computed(() => route.params.id === 'new')
const templateId = computed(() => route.params.id as string)

const isLoading = ref(true)
const isSaving = ref(false)
const error = ref('')

// Form data
const form = ref({
  slug: '',
  name: '',
  subject: '',
  body_html: '',
  body_text: '',
  category: 'transactional' as 'transactional' | 'marketing',
  is_active: true,
  variables_schema: {} as Record<string, VariableSpec>,
})

const isSystem = ref(false)
const activeTab = ref('html')

// Preview state
const showPreview = ref(false)
const previewData = ref<Record<string, string>>({})
const previewResult = ref<{
  subject: string
  body_html: string
  body_text: string
} | null>(null)
const isPreviewLoading = ref(false)

// Variables editor
const newVarName = ref('')
const newVarType = ref('string')
const newVarRequired = ref(true)
const newVarDescription = ref('')

onMounted(async () => {
  if (!isNew.value) {
    await loadTemplate()
  } else {
    isLoading.value = false
  }
})

async function loadTemplate() {
  isLoading.value = true
  error.value = ''
  try {
    const template = await adminApi.getEmailTemplate(templateId.value)
    form.value = {
      slug: template.slug,
      name: template.name,
      subject: template.subject,
      body_html: template.body_html,
      body_text: template.body_text,
      category: template.category,
      is_active: template.is_active,
      variables_schema: template.variables_schema || {},
    }
    isSystem.value = template.is_system

    // Initialize preview data with empty values
    for (const key of Object.keys(template.variables_schema || {})) {
      previewData.value[key] = ''
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка загрузки'
  } finally {
    isLoading.value = false
  }
}

async function saveTemplate() {
  if (!form.value.name || !form.value.subject || !form.value.body_html || !form.value.body_text) {
    error.value = 'Заполните все обязательные поля'
    return
  }

  isSaving.value = true
  error.value = ''

  try {
    if (isNew.value) {
      if (!form.value.slug) {
        error.value = 'Slug обязателен для нового шаблона'
        isSaving.value = false
        return
      }
      await adminApi.createEmailTemplate({
        slug: form.value.slug,
        name: form.value.name,
        subject: form.value.subject,
        body_html: form.value.body_html,
        body_text: form.value.body_text,
        category: form.value.category,
        variables_schema: form.value.variables_schema,
      })
    } else {
      await adminApi.updateEmailTemplate(templateId.value, {
        name: form.value.name,
        subject: form.value.subject,
        body_html: form.value.body_html,
        body_text: form.value.body_text,
        category: form.value.category,
        variables_schema: form.value.variables_schema,
        is_active: form.value.is_active,
      })
    }
    router.push('/admin/email-templates')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка сохранения'
  } finally {
    isSaving.value = false
  }
}

async function generatePreview() {
  isPreviewLoading.value = true
  try {
    previewResult.value = await adminApi.previewEmailTemplate({
      subject: form.value.subject,
      body_html: form.value.body_html,
      body_text: form.value.body_text,
      variables: previewData.value,
    })
    showPreview.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Ошибка предпросмотра'
  } finally {
    isPreviewLoading.value = false
  }
}

function addVariable() {
  const varName = newVarName.value.trim()
  if (!varName) return

  form.value.variables_schema[varName] = {
    type: newVarType.value,
    required: newVarRequired.value,
    description: newVarDescription.value || undefined,
  }

  // Add to preview data BEFORE reset
  previewData.value[varName] = ''

  // Reset form
  newVarName.value = ''
  newVarType.value = 'string'
  newVarRequired.value = true
  newVarDescription.value = ''
}

function removeVariable(name: string) {
  delete form.value.variables_schema[name]
  delete previewData.value[name]
}

function goBack() {
  router.push('/admin/email-templates')
}

const variablesList = computed(() => Object.entries(form.value.variables_schema))
</script>

<template>
  <div class="min-h-screen bg-slate-900">
    <!-- Header -->
    <header class="bg-slate-800 border-b border-slate-700 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-14">
          <div class="flex items-center gap-4">
            <Button
              variant="ghost"
              size="sm"
              class="text-slate-400 hover:text-white"
              @click="goBack"
            >
              <ArrowLeft class="h-4 w-4 mr-2" />
              Назад
            </Button>
            <h1 class="text-lg font-semibold text-white">
              {{ isNew ? 'Новый шаблон' : 'Редактирование шаблона' }}
            </h1>
            <Lock v-if="isSystem" class="h-4 w-4 text-slate-500" title="Системный шаблон" />
          </div>
          <div class="flex items-center gap-2">
            <Button
              variant="outline"
              class="border-slate-600 text-slate-300 hover:bg-slate-700"
              :disabled="isPreviewLoading"
              @click="generatePreview"
            >
              <Eye class="h-4 w-4 mr-2" />
              Предпросмотр
            </Button>
            <Button
              class="bg-indigo-600 hover:bg-indigo-700"
              :disabled="isSaving"
              @click="saveTemplate"
            >
              <Save class="h-4 w-4 mr-2" />
              {{ isSaving ? 'Сохранение...' : 'Сохранить' }}
            </Button>
          </div>
        </div>
      </div>
    </header>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Error -->
      <ErrorBanner v-if="error" :message="error" class="mb-6" />

      <!-- Loading -->
      <div v-if="isLoading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      </div>

      <div v-else class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Main Editor -->
        <div class="lg:col-span-2 space-y-6">
          <!-- Basic Info -->
          <Card class="bg-slate-800 border-slate-700">
            <CardHeader>
              <CardTitle class="text-white flex items-center gap-2">
                <FileText class="h-5 w-5" />
                Основная информация
              </CardTitle>
            </CardHeader>
            <CardContent class="space-y-4">
              <div class="grid grid-cols-2 gap-4">
                <div class="space-y-2">
                  <Label class="text-slate-300">Название *</Label>
                  <Input
                    v-model="form.name"
                    placeholder="Сброс пароля"
                    class="bg-slate-900 border-slate-600 text-white"
                  />
                </div>
                <div class="space-y-2">
                  <Label class="text-slate-300">Slug {{ isNew ? '*' : '(только чтение)' }}</Label>
                  <Input
                    v-model="form.slug"
                    placeholder="password-reset"
                    :disabled="!isNew"
                    class="bg-slate-900 border-slate-600 text-white disabled:opacity-50"
                  />
                </div>
              </div>

              <div class="grid grid-cols-2 gap-4">
                <div class="space-y-2">
                  <Label class="text-slate-300">Категория</Label>
                  <Select v-model="form.category">
                    <SelectTrigger class="bg-slate-900 border-slate-600 text-white">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent class="bg-slate-800 border-slate-600">
                      <SelectItem value="transactional" class="text-white hover:bg-slate-700">
                        Транзакционный
                      </SelectItem>
                      <SelectItem value="marketing" class="text-white hover:bg-slate-700">
                        Маркетинговый
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div class="space-y-2">
                  <Label class="text-slate-300">Статус</Label>
                  <div class="flex items-center gap-2 h-10">
                    <Switch
                      :checked="form.is_active"
                      @update:checked="form.is_active = $event"
                    />
                    <span class="text-sm text-slate-400">
                      {{ form.is_active ? 'Активен' : 'Неактивен' }}
                    </span>
                  </div>
                </div>
              </div>

              <div class="space-y-2">
                <Label class="text-slate-300">Тема письма *</Label>
                <Input
                  v-model="form.subject"
                  placeholder="Сброс пароля для {{.Name}}"
                  class="bg-slate-900 border-slate-600 text-white font-mono"
                />
                <p class="text-xs text-slate-500">
                  Используйте <code class="bg-slate-700 px-1 rounded" v-pre>{{.VariableName}}</code> для подстановки переменных
                </p>
              </div>
            </CardContent>
          </Card>

          <!-- Content Editor -->
          <Card class="bg-slate-800 border-slate-700">
            <CardHeader>
              <CardTitle class="text-white flex items-center gap-2">
                <Code class="h-5 w-5" />
                Содержимое письма
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs v-model="activeTab" class="w-full">
                <TabsList class="bg-slate-700 mb-4">
                  <TabsTrigger value="html" class="data-[state=active]:bg-slate-600">
                    HTML
                  </TabsTrigger>
                  <TabsTrigger value="text" class="data-[state=active]:bg-slate-600">
                    Plain Text
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="html">
                  <Textarea
                    v-model="form.body_html"
                    placeholder="<!DOCTYPE html>..."
                    class="bg-slate-900 border-slate-600 text-white font-mono text-sm min-h-[400px]"
                  />
                </TabsContent>

                <TabsContent value="text">
                  <Textarea
                    v-model="form.body_text"
                    placeholder="Текстовая версия письма..."
                    class="bg-slate-900 border-slate-600 text-white font-mono text-sm min-h-[400px]"
                  />
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </div>

        <!-- Sidebar -->
        <div class="space-y-6">
          <!-- Variables -->
          <Card class="bg-slate-800 border-slate-700">
            <CardHeader>
              <CardTitle class="text-white flex items-center gap-2">
                <Variable class="h-5 w-5" />
                Переменные
              </CardTitle>
            </CardHeader>
            <CardContent class="space-y-4">
              <!-- Existing variables -->
              <div v-if="variablesList.length > 0" class="space-y-2">
                <div
                  v-for="[name, spec] in variablesList"
                  :key="name"
                  class="flex items-center justify-between p-2 bg-slate-900 rounded-lg"
                >
                  <div>
                    <code class="text-indigo-400 text-sm">&#123;&#123;.{{ name }}&#125;&#125;</code>
                    <div class="text-xs text-slate-500 mt-1">
                      {{ spec.type }}
                      <span v-if="spec.required" class="text-red-400">*</span>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                    @click="removeVariable(name)"
                  >
                    <Trash2 class="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div v-else class="text-sm text-slate-500 text-center py-4">
                Нет переменных
              </div>

              <!-- Add new variable -->
              <div class="border-t border-slate-700 pt-4 space-y-3">
                <div class="text-sm font-medium text-slate-300">Добавить переменную</div>
                <Input
                  v-model="newVarName"
                  placeholder="Имя (например: Name)"
                  class="bg-slate-900 border-slate-600 text-white text-sm"
                />
                <div class="flex gap-2">
                  <Select v-model="newVarType">
                    <SelectTrigger class="bg-slate-900 border-slate-600 text-white text-sm flex-1">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent class="bg-slate-800 border-slate-600">
                      <SelectItem value="string" class="text-white hover:bg-slate-700">string</SelectItem>
                      <SelectItem value="number" class="text-white hover:bg-slate-700">number</SelectItem>
                      <SelectItem value="boolean" class="text-white hover:bg-slate-700">boolean</SelectItem>
                    </SelectContent>
                  </Select>
                  <div class="flex items-center gap-2">
                    <Switch
                      :checked="newVarRequired"
                      @update:checked="newVarRequired = $event"
                    />
                    <span class="text-xs text-slate-400">Обяз.</span>
                  </div>
                </div>
                <Input
                  v-model="newVarDescription"
                  placeholder="Описание (опционально)"
                  class="bg-slate-900 border-slate-600 text-white text-sm"
                />
                <Button
                  variant="outline"
                  size="sm"
                  class="w-full border-slate-600 text-slate-300 hover:bg-slate-700"
                  @click="addVariable"
                >
                  <Plus class="h-4 w-4 mr-2" />
                  Добавить
                </Button>
              </div>
            </CardContent>
          </Card>

          <!-- Preview Data -->
          <Card v-if="variablesList.length > 0" class="bg-slate-800 border-slate-700">
            <CardHeader>
              <CardTitle class="text-white flex items-center gap-2">
                <Eye class="h-5 w-5" />
                Данные для предпросмотра
              </CardTitle>
            </CardHeader>
            <CardContent class="space-y-3">
              <div v-for="[name] in variablesList" :key="name" class="space-y-1">
                <Label class="text-slate-400 text-xs">{{ name }}</Label>
                <Input
                  v-model="previewData[name]"
                  :placeholder="`Значение для ${name}`"
                  class="bg-slate-900 border-slate-600 text-white text-sm"
                />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <!-- Preview Modal -->
      <div
        v-if="showPreview && previewResult"
        class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4"
        @click.self="showPreview = false"
      >
        <Card class="bg-slate-800 border-slate-700 w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
          <CardHeader class="flex flex-row items-center justify-between">
            <CardTitle class="text-white">Предпросмотр письма</CardTitle>
            <Button
              variant="ghost"
              size="sm"
              class="text-slate-400 hover:text-white"
              @click="showPreview = false"
            >
              Закрыть
            </Button>
          </CardHeader>
          <CardContent class="overflow-auto flex-1">
            <div class="space-y-4">
              <div>
                <Label class="text-slate-400 text-xs">Тема</Label>
                <div class="text-white font-medium mt-1">{{ previewResult.subject }}</div>
              </div>
              <Tabs default-value="preview-html">
                <TabsList class="bg-slate-700">
                  <TabsTrigger value="preview-html" class="data-[state=active]:bg-slate-600">
                    HTML
                  </TabsTrigger>
                  <TabsTrigger value="preview-text" class="data-[state=active]:bg-slate-600">
                    Text
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="preview-html" class="mt-4">
                  <div
                    class="bg-white rounded-lg p-4"
                    v-html="previewResult.body_html"
                  />
                </TabsContent>
                <TabsContent value="preview-text" class="mt-4">
                  <pre class="bg-slate-900 text-slate-300 p-4 rounded-lg text-sm whitespace-pre-wrap">{{ previewResult.body_text }}</pre>
                </TabsContent>
              </Tabs>
            </div>
          </CardContent>
        </Card>
      </div>
    </main>
  </div>
</template>
