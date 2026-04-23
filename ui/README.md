# EdgeOS UI

基于 Vue 3 + Tailwind CSS 的工业边缘网关前端项目

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **TypeScript** - 类型安全
- **Tailwind CSS** - 实用优先的 CSS 框架
- **Vite** - 下一代前端构建工具
- **Vue Router** - 官方路由管理器
- **Lucide Vue** - 现代化图标库

## 项目结构

```
ui/
├── src/
│   ├── api/              # API 接口
│   ├── assets/           # 静态资源
│   │   └── css/         # 全局样式
│   ├── components/       # 组件
│   │   ├── edge/        # 工业级组件
│   │   └── layout/      # 布局组件
│   ├── lib/             # 工具函数
│   ├── router/          # 路由配置
│   ├── types/           # 类型定义
│   ├── views/           # 页面组件
│   ├── App.vue          # 根组件
│   └── main.ts          # 入口文件
├── index.html
├── package.json
├── tailwind.config.js
├── tsconfig.json
└── vite.config.js
```

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## 样式规范

项目遵循 [样式规范.md](../docs/样式规范.md) 中定义的工业级 UI 标准：

- 直角设计，无圆角
- 无阴影，使用边框分隔
- 高对比度配色
- 大触控区域（适配工业手套）
- 清晰的状态指示（脉冲动画）

## 组件说明

### 工业级组件

- **StatusIndicator** - 状态指示器，支持脉冲动画
- **ProtocolBadge** - 协议类型徽章
- **MetricCard** - 实时数值卡片
- **DataTable** - 高密度数据表格
- **DangerDialog** - 危险操作确认对话框
- **Toast** - 消息通知

### 布局组件

- **AppLayout** - 主布局容器
- **AppSidebar** - 侧边栏导航
- **AppHeader** - 顶部导航栏

## 认证

项目实现了基于 JWT 的认证机制：

- 登录页面：`/login`
- 受保护路由需要有效的 JWT token
- Token 存储在 localStorage 中
- 自动重定向到登录页面（未认证时）

## API 集成

所有 API 请求通过 `/api` 前缀代理到后端服务器：

- 开发环境：`http://localhost:8080`
- 自动添加 JWT token 到请求头
- 统一错误处理

## 浏览器支持

- Chrome (推荐)
- Firefox
- Edge
- Safari

## License

MIT