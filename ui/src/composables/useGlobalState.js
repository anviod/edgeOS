// 简单的全局消息提示
export function showMessage(message, type = 'info') {
  console.log(`[${type.toUpperCase()}] ${message}`)

  // 简单的 toast 实现
  const toast = document.createElement('div')
  toast.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 transition-all duration-300 transform translate-x-full ${
    type === 'error' ? 'bg-red-500 text-white' :
    type === 'success' ? 'bg-green-500 text-white' :
    'bg-blue-500 text-white'
  }`
  toast.textContent = message

  document.body.appendChild(toast)

  // 动画进入
  requestAnimationFrame(() => {
    toast.classList.remove('translate-x-full')
  })

  // 3秒后移除
  setTimeout(() => {
    toast.classList.add('translate-x-full')
    setTimeout(() => {
      document.body.removeChild(toast)
    }, 300)
  }, 3000)
}
