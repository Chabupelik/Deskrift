import axios from 'axios'

const client = axios.create({
  baseURL: '/api/v1',
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export const authApi = {
  telegram: (initData: string) =>
    client.post<{ access_token: string }>('/auth/telegram', { init_data: initData }),
}

export default client

