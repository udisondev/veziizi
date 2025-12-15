import { api } from './client'
import type { RegisterRequest, RegisterResponse } from '@/types/registration'

export const organizationsApi = {
  register(data: RegisterRequest): Promise<RegisterResponse> {
    return api.post('/organizations', data)
  },
}
