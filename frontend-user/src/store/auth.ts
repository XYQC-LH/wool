import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import Cookies from 'js-cookie'
import { User, userApi } from '@/lib/api'

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
  
  // Actions
  login: (username: string, password: string) => Promise<boolean>
  register: (username: string, email: string, password: string) => Promise<boolean>
  logout: () => void
  fetchProfile: () => Promise<void>
  updateProfile: (data: { username?: string; email?: string; avatar_url?: string }) => Promise<void>
  clearError: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: Cookies.get('token') || null,
      isAuthenticated: !!Cookies.get('token'),
      isLoading: false,
      error: null,

      login: async (username: string, password: string) => {
        set({ isLoading: true, error: null })
        try {
          const response = await userApi.login({ username, password })
          if (response.code === 0) {
            const { token, user } = response.data
            Cookies.set('token', token, { expires: 7 })
            set({
              user,
              token,
              isAuthenticated: true,
              isLoading: false,
            })
            return true
          } else {
            set({ error: response.message, isLoading: false, isAuthenticated: false })
            return false
          }
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : '登录失败'
          set({ error: errorMessage, isLoading: false, isAuthenticated: false })
          return false
        }
      },

      register: async (username: string, email: string, password: string) => {
        set({ isLoading: true, error: null })
        try {
          const response = await userApi.register({ username, email, password })
          if (response.code === 0) {
            set({ isLoading: false })
            return true
          } else {
            set({ error: response.message, isLoading: false })
            return false
          }
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : '注册失败'
          set({ error: errorMessage, isLoading: false })
          return false
        }
      },

      logout: () => {
        Cookies.remove('token')
        set({
          user: null,
          token: null,
          isAuthenticated: false,
        })
      },

      fetchProfile: async () => {
        let { token } = get()
        if (!token) {
          token = Cookies.get('token') || null
          if (token) {
            set({ token, isAuthenticated: true })
          }
        }
        if (!token) return

        set({ isLoading: true })
        try {
          const response = await userApi.getProfile()
          if (response.code === 0) {
            set({ user: response.data, isAuthenticated: true, isLoading: false })
          } else {
            set({ isLoading: false })
          }
        } catch {
          Cookies.remove('token')
          set({ user: null, token: null, isAuthenticated: false, isLoading: false })
        }
      },

      updateProfile: async (data: { username?: string; email?: string; avatar_url?: string }) => {
        set({ isLoading: true, error: null })
        try {
          const response = await userApi.updateProfile(data)
          if (response.code === 0) {
            await get().fetchProfile()
          } else {
            set({ error: response.message, isLoading: false })
          }
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : '更新失败'
          set({ error: errorMessage, isLoading: false })
        }
      },

      clearError: () => set({ error: null }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ token: state.token }),
    }
  )
)

declare global {
  interface Window {
    __nexusAuthListenerInstalled?: boolean
  }
}

if (typeof window !== 'undefined' && !window.__nexusAuthListenerInstalled) {
  window.__nexusAuthListenerInstalled = true
  window.addEventListener('auth:logout', () => {
    useAuthStore.getState().logout()
  })
}
