export interface LoginRequest {
  loginFlag: boolean
  loginType: 'local' | 'ldap'
  data: {
    username: string
    password: string
    nonce: string
  }
}

export interface LoginResponse {
  code: string
  msg: string
  data: {
    username: string
    token: string
    permissions: string[]
  }
}

export interface NonceResponse {
  code: string
  data: {
    nonce: string
  }
}

export interface SystemInfo {
  name: string
  softVer: string
}

export interface ApiResponse<T = any> {
  code: string
  msg?: string
  data?: T
}