import { api } from './client'
import type { LoginRequest, LoginResponse, MeResponse } from '@/types/api'

export const authApi = {
  login(data: LoginRequest): Promise<LoginResponse> {
    return api.post('/auth/login', data)
  },

  logout(): Promise<void> {
    return api.post('/auth/logout')
  },

  me(): Promise<MeResponse> {
    return api.get('/auth/me')
  },
}
