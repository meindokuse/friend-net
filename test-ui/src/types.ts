export interface TokenState {
  accessToken: string
  refreshToken: string
  accountId: string
}

export interface Session {
  id: string
  ip: string
  user_agent: string
  created_at: string
  is_current?: boolean
}

export interface UserResponse {
  id: string
  username: string
  email?: string
  phone?: string
  display_name: string
  bio?: string
  avatar_url?: string
  email_verified: boolean
  phone_verified: boolean
  is_active: boolean
  privacy: {
    who_can_message: string
    who_can_see_last_seen: string
    who_can_see_profile: string
  }
  language: string
  timezone: string
  created_at: string
  updated_at: string
  last_seen_at?: string
  version: number
}

export interface PublicUserResponse {
  id: string
  username: string
  display_name: string
  bio?: string
  avatar_url?: string
  last_seen_at?: string
}
