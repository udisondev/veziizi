import { api } from './client'
import type { LoginRequest, LoginResponse, MeResponse } from '@/types/api'

export interface ForgotPasswordRequest {
  email: string
}

export interface ResetPasswordRequest {
  token: string
  password: string
}

export const authApi = {
  login(data: LoginRequest, fingerprint?: string): Promise<LoginResponse> {
    const headers: Record<string, string> = {}
    if (fingerprint) {
      headers['X-Fingerprint'] = fingerprint
    }
    return api.post('/auth/login', data, { headers })
  },

  logout(): Promise<void> {
    return api.post('/auth/logout')
  },

  me(): Promise<MeResponse> {
    return api.get('/auth/me')
  },

  forgotPassword(data: ForgotPasswordRequest): Promise<void> {
    return api.post('/auth/forgot-password', data)
  },

  resetPassword(data: ResetPasswordRequest): Promise<void> {
    return api.post('/auth/reset-password', data)
  },
}
