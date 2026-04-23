import { defineStore } from 'pinia'

export const configStore = defineStore('app', {
  state: () => ({
    configInfo: {
      name: 'EdgeOS',
      softVer: '1.0.0'
    },
    theme: 'light',
    language: 'zh-CN',
    sidebarCollapsed: false,
  }),

  getters: {
    // 获取系统配置
    getConfigInfo: (state) => state.configInfo,
    
    // 获取系统名称
    getAppName: (state) => state.configInfo.name || 'EdgeOS',
    
    // 获取版本号
    getVersion: (state) => state.configInfo.softVer || '1.0.0',
    
    // 获取主题
    getTheme: (state) => state.theme,
    
    // 获取语言
    getLanguage: (state) => state.language
  },

  actions: {
    // 设置配置信息
    setConfigInfo(configInfo) {
      this.configInfo = {
        ...this.configInfo,
        ...configInfo
      }
      
      // 保存到 localStorage
      localStorage.setItem('configInfo', JSON.stringify(this.configInfo))
    },

    // 设置主题
    setTheme(theme) {
      this.theme = theme
      localStorage.setItem('theme', theme)
      
      // 应用主题到 document
      if (typeof document !== 'undefined') {
        document.documentElement.setAttribute('data-theme', theme)
      }
    },

    // 设置语言
    setLanguage(language) {
      this.language = language
      localStorage.setItem('language', language)
    },

    // 切换侧边栏折叠状态
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
      localStorage.setItem('sidebarCollapsed', String(this.sidebarCollapsed))
    },

    // 从 localStorage 恢复配置
    restoreConfig() {
      try {
        const configInfoStr = localStorage.getItem('configInfo')
        if (configInfoStr) {
          this.configInfo = JSON.parse(configInfoStr)
        }
        
        const theme = localStorage.getItem('theme')
        if (theme) {
          this.setTheme(theme)
        }
        
        const language = localStorage.getItem('language')
        if (language) {
          this.language = language
        }

        const collapsed = localStorage.getItem('sidebarCollapsed')
        if (collapsed !== null) {
          this.sidebarCollapsed = collapsed === 'true'
        }
      } catch (error) {
        console.error('恢复配置失败:', error)
      }
    }
  }
})

// 别名导出，方便在 TS 组件中使用
export const useAppStore = configStore

