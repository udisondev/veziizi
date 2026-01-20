/**
 * Mock Handlers for Admin API
 */

import { registerHandler } from './index'
import { mockEmailTemplates } from '@/sandbox/mockData/emailTemplates'
import type { VariableSpec } from '@/types/admin'

export function adminHandlers(): void {
  // ===============================
  // Email Templates CRUD
  // ===============================

  // List templates
  registerHandler('GET', '/admin/email-templates', (_params, _body, query) => {
    const filter = {
      category: query?.get('category') || undefined,
      is_active: query?.has('is_active') ? query.get('is_active') === 'true' : undefined,
      is_system: query?.has('is_system') ? query.get('is_system') === 'true' : undefined,
      search: query?.get('search') || undefined,
      limit: query?.has('limit') ? parseInt(query.get('limit') || '50') : 50,
      offset: query?.has('offset') ? parseInt(query.get('offset') || '0') : 0,
    }

    const result = mockEmailTemplates.list(filter)
    return { data: result }
  })

  // Get single template
  registerHandler('GET', '/admin/email-templates/:id', (params) => {
    const template = mockEmailTemplates.get(params.id || '')
    if (!template) {
      return { status: 404, data: { error: 'Template not found' } }
    }
    return { data: template }
  })

  // Create template
  registerHandler('POST', '/admin/email-templates', (_params, body) => {
    const data = body as {
      slug: string
      name: string
      subject: string
      body_html: string
      body_text: string
      category: 'transactional' | 'marketing'
      variables_schema?: Record<string, VariableSpec>
    }

    // Validate required fields
    if (!data.slug || !data.name || !data.subject || !data.body_html || !data.body_text) {
      return { status: 400, data: { error: 'Missing required fields' } }
    }

    const template = mockEmailTemplates.create(data)
    return { status: 201, data: template }
  })

  // Update template
  registerHandler('PATCH', '/admin/email-templates/:id', (params, body) => {
    const data = body as {
      name?: string
      subject?: string
      body_html?: string
      body_text?: string
      category?: 'transactional' | 'marketing'
      variables_schema?: Record<string, VariableSpec>
      is_active?: boolean
    }

    const template = mockEmailTemplates.update(params.id || '', data)
    if (!template) {
      return { status: 404, data: { error: 'Template not found' } }
    }
    return { data: template }
  })

  // Delete template
  registerHandler('DELETE', '/admin/email-templates/:id', (params) => {
    const deleted = mockEmailTemplates.delete(params.id || '')
    if (!deleted) {
      return { status: 400, data: { error: 'Cannot delete system template or template not found' } }
    }
    return { status: 204 }
  })

  // Preview template
  registerHandler('POST', '/admin/email-templates/preview', (_params, body) => {
    const data = body as {
      subject: string
      body_html: string
      body_text: string
      variables: Record<string, string>
    }

    const preview = mockEmailTemplates.preview(data)
    return { data: preview }
  })
}
