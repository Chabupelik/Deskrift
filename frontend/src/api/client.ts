import axios from 'axios'

const client = axios.create({
  baseURL: '/api/v1',
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export interface LoginInitResponse {
  login_id: string
  expires_in: number
}

export interface LoginCheckResponse {
  status: 'pending' | 'approved' | 'rejected' | 'expired'
  access_token?: string
  token_type?: string
}

export const authApi = {
  requestLogin: () => client.post<LoginInitResponse>('/auth/login'),
  checkLogin: (loginId: string) => client.get<LoginCheckResponse>(`/auth/login/${loginId}`),
}

export default client
