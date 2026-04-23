import { defineStore } from 'pinia'

export const userStore = defineStore('user', {
  state: () => ({
    userInfo: null,
    permissions: [],
    token: null,
    isLoggedIn: false
  }),

  getters: {
    // 获取用户信息
    getUserInfo: (state) => state.userInfo,
    
    // 获取权限列表
    getPermissions: (state) => state.permissions,
    
    // 获取 token
    getToken: (state) => state.token,
    
    // 检查是否已登录
    checkLogin: (state) => state.isLoggedIn && !!state.token
  },

  actions: {
    // 设置登录信息
    setLoginInfo(userInfo, permissions, token) {
      this.userInfo = userInfo
      this.permissions = permissions || []
      this.token = token
      this.isLoggedIn = true
      
      // 保存 token 到 localStorage (API 请求需要)
      localStorage.setItem('token', token)
      
      // 保存完整登录信息
      const loginData = {
        userInfo,
        permissions,
        token,
        loginTime: Date.now()
      }
      localStorage.setItem('loginInfo', JSON.stringify(loginData))
    },

    // 清除登录信息
    logout() {
      this.userInfo = null
      this.permissions = []
      this.token = null
      this.isLoggedIn = false
      
      // 清除 localStorage 中的认证信息
      localStorage.removeItem('token')
      localStorage.removeItem('loginInfo')
      localStorage.removeItem('rememberedAccount')
    },

    // 从 localStorage 恢复登录状态
    restoreLoginInfo() {
      try {
        const loginInfoStr = localStorage.getItem('loginInfo')
        if (loginInfoStr) {
          const loginInfo = JSON.parse(loginInfoStr)
          this.userInfo = loginInfo.userInfo
          this.permissions = loginInfo.permissions || []
          this.token = loginInfo.token
          this.isLoggedIn = true
          return true
        }
        return false
      } catch (error) {
        console.error('恢复登录信息失败:', error)
        return false
      }
    },

    // 检查 token 是否有效
    isTokenValid() {
      if (!this.token) return false
      
      try {
        const loginInfoStr = localStorage.getItem('loginInfo')
        if (!loginInfoStr) return false
        
        const loginInfo = JSON.parse(loginInfoStr)
        const loginTime = loginInfo.loginTime || 0
        const tokenTTL = 7 * 24 * 60 * 60 * 1000 // 7天
        
        return Date.now() - loginTime < tokenTTL
      } catch (error) {
        console.error('检查 token 有效性失败:', error)
        return false
      }
    },

    // 更新用户信息
    updateUserInfo(userInfo) {
      this.userInfo = userInfo
      
      // 同步更新 localStorage
      const loginInfoStr = localStorage.getItem('loginInfo')
      if (loginInfoStr) {
        try {
          const loginInfo = JSON.parse(loginInfoStr)
          loginInfo.userInfo = userInfo
          localStorage.setItem('loginInfo', JSON.stringify(loginInfo))
        } catch (error) {
          console.error('更新用户信息失败:', error)
        }
      }
    },

    // 检查是否有特定权限
    hasPermission(permission) {
      return this.permissions.includes(permission)
    }
  }
})
