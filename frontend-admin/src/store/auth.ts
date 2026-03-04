import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import Cookies from 'js-cookie'
import { Admin, authApi } from '@/lib/api'

interface AuthState {
  admin: Admin | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
  
  login: (username: string, password: string) => Promise<boolean>
  logout: () => void
  clearError: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      admin: null,
      token: Cookies.get('admin_token') || null,
      isAuthenticated: !!Cookies.get('admin_token'),
      isLoading: false,
      error: null,

      login: async (username: string, password: string) => {
        set({ isLoading: true, error: null })
        try {
          const response = await authApi.login({ username, password })
          if (response.code === 0) {
            const { token, user } = response.data
            Cookies.set('admin_token', token, { expires: 7 })
            set({
              admin: user,
              token,
              isAuthenticated: true,
              isLoading: false,
            })
            return true
          } else {
            set({ error: response.message, isLoading: false })
            return false
          }
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : '登录失败'
          set({ error: errorMessage, isLoading: false })
          return false
        }
      },

      logout: () => {
        Cookies.remove('admin_token')
        set({
          admin: null,
          token: null,
          isAuthenticated: false,
        })
      },

      clearError: () => set({ error: null }),
    }),
    {
      name: 'admin-auth-storage',
      partialize: (state) => ({ token: state.token }),
    }
  )
)
