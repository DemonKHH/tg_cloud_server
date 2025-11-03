"use client"

import { createContext, useContext, useEffect, useState, ReactNode } from "react"
import { authAPI } from "@/lib/api"

export interface UserProfile {
  id: number
  username: string
  email: string
  role: string
  is_active: boolean
  last_login_at?: string
  created_at: string
}

interface UserContextType {
  user: UserProfile | null
  loading: boolean
  error: Error | null
  refresh: () => Promise<void>
  logout: () => Promise<void>
}

const UserContext = createContext<UserContextType | undefined>(undefined)

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const fetchProfile = async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await authAPI.getProfile()
      if (response.data && typeof response.data === 'object') {
        setUser(response.data as UserProfile)
      }
    } catch (err) {
      console.error("Failed to fetch user profile:", err)
      setError(err instanceof Error ? err : new Error("获取用户信息失败"))
      setUser(null)
    } finally {
      setLoading(false)
    }
  }

  const logout = async () => {
    try {
      await authAPI.logout()
    } catch (err) {
      console.error("Logout failed:", err)
    } finally {
      localStorage.removeItem("token")
      setUser(null)
      window.location.href = "/login"
    }
  }

  useEffect(() => {
    // 检查是否有 token，如果有则获取用户信息
    const token = typeof window !== 'undefined' ? localStorage.getItem("token") : null
    if (token) {
      fetchProfile()
    } else {
      setLoading(false)
    }
  }, [])

  return (
    <UserContext.Provider value={{ user, loading, error, refresh: fetchProfile, logout }}>
      {children}
    </UserContext.Provider>
  )
}

export function useUser() {
  const context = useContext(UserContext)
  if (context === undefined) {
    throw new Error("useUser must be used within a UserProvider")
  }
  return context
}

