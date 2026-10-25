export default {
  batchImageGuide: {
    title: '图片批量生成',
    description: '一次提交多条提示词，任务完成后可统一下载图片结果'
  },
  // Home Page
  home: {
    viewDocs: '查看文档',
    docs: '文档',
    switchToLight: '切换到浅色模式',
    switchToDark: '切换到深色模式',
    dashboard: '控制台',
    login: '登录',
    goToDashboard: '进入控制台',
    // 沉浸式首屏
    hero: {
      primaryCta: '登录',
      primaryCtaAuthed: '进入控制台',
      secondaryCta: '先逛逛',
      moreMenu: '更多',
      reelLabel: '模型广场',
      contact: '商务合作',
      contactCopied: '联系方式已复制',
      contactCopyFailed: '复制失败，请手动选择联系方式',
      carouselPrev: '上一组文案',
      carouselNext: '下一组文案',
      carouselPause: '暂停轮播',
      carouselPlay: '继续轮播',
      slides: {
        s1: {
          prefix: '一个密钥',
          verb: '连接所有 AI',
          subtitle: '从一句需求到一次调用。\n工作台、算力与模型市场，同一入口。',
          left: {
            kicker: '接入 / 模型',
            title: 'Claude · GPT · Gemini',
            desc: '一个密钥打通主流厂商，切换只改模型名。'
          },
          right: {
            kicker: '计费 / 用量',
            title: '按量计费 · 透明',
            desc: '用多少算多少，配额上限随手设。'
          }
        },
        s2: {
          prefix: '订阅转 API',
          verb: '一个账号通行',
          subtitle: '订阅额度不再闲置。\n转成 API Key，接入你自己的工具。',
          left: {
            kicker: '账号 / 迁移',
            title: '订阅额度转 Key',
            desc: '已有订阅直接折算，不必重复付费。'
          },
          right: {
            kicker: '会话 / 稳定',
            title: '粘性会话',
            desc: '长上下文不断线，多轮任务一路到底。'
          }
        },
        s3: {
          prefix: '模型广场',
          verb: '价格一目了然',
          subtitle: '图片、视频、对话模型同台比较。\n分档定价写在页面上，不用问人。',
          left: {
            kicker: '目录 / 模型',
            title: '一个目录全站共用',
            desc: '图片按尺寸与质量、视频按秒或按条。'
          },
          right: {
            kicker: '价格 / 透明',
            title: '售价与成本分开',
            desc: '每次调价都留痕，历史可查。'
          }
        },
        s4: {
          prefix: '控制台',
          verb: '用量随时对账',
          subtitle: '余额、配额、调用明细实时刷新。\n花在哪、还剩多少，一眼看清。',
          left: {
            kicker: '看板 / 用量',
            title: '余额与配额实时',
            desc: '按 Key、按模型拆分，逐条可下钻。'
          },
          right: {
            kicker: '密钥 / 管理',
            title: '分组与上限自助',
            desc: '一个 Key 一个额度，超限自动拦截。'
          }
        },
        s5: {
          prefix: '团队协作',
          verb: '多人分权管理',
          subtitle: '成员、密钥、额度分层管理。\n谁在用、用了多少，都有记录。',
          left: {
            kicker: '成员 / 权限',
            title: '角色按需授权',
            desc: '管理员、运营、普通用户各司其职。'
          },
          right: {
            kicker: '审计 / 日志',
            title: '每次调用可追溯',
            desc: '登录、改密、调用全留审计记录。'
          }
        }
      }
    },
    footer: {
      allRightsReserved: '保留所有权利。'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key 用量查询',
    subtitle: '输入您的 API Key 以查看实时消费金额与使用状态',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: '查询',
    querying: '查询中...',
    privacyNote: '您的 Key 仅在浏览器本地处理，不会被存储',
    dateRange: '统计范围:',
    dateRangeToday: '今日',
    dateRange7d: '7 天',
    dateRange30d: '30 天',
    dateRange90d: '90 天',
    dateRangeCustom: '自定义',
    apply: '应用',
    used: '已使用',
    detailInfo: '详细信息',
    tokenStats: 'Token 统计',
    dailyDetail: '按日明细',
    modelStats: '模型用量统计',
    // Table headers
    date: '日期',
    model: '模型',
    requests: '请求数',
    inputTokens: '输入 Tokens',
    outputTokens: '输出 Tokens',
    cacheCreationTokens: '缓存创建',
    cacheReadTokens: '缓存读取',
    cacheWriteTokens: '缓存写入',
    totalTokens: '总 Tokens',
    cost: '费用',
    // Status
    quotaMode: 'Key 限额模式',
    walletBalance: '钱包余额',
    // Ring card titles
    totalQuota: '总额度',
    limit5h: '5 小时限额',
    limitDaily: '日限额',
    limit7d: '7 天限额',
    limitWeekly: '周限额',
    limitMonthly: '月限额',
    // Detail rows
    remainingQuota: '剩余额度',
    expiresAt: '过期时间',
    todayExpires: '(今日到期)',
    daysLeft: '({days} 天)',
    usedQuota: '已用额度',
    resetNow: '即将重置',
    subscriptionType: '订阅类型',
    billingType: '计费方式',
    subscriptionExpires: '订阅到期',
    // Usage stat cells
    todayRequests: '今日请求',
    todayInputTokens: '今日输入',
    todayOutputTokens: '今日输出',
    todayTokens: '今日 Tokens',
    todayCacheCreation: '今日缓存创建',
    todayCacheRead: '今日缓存读取',
    todayCost: '今日费用',
    rpmTpm: 'RPM / TPM',
    totalRequests: '累计请求',
    totalInputTokens: '累计输入',
    totalOutputTokens: '累计输出',
    totalTokensLabel: '累计 Tokens',
    totalCacheCreation: '累计缓存创建',
    totalCacheRead: '累计缓存读取',
    totalCost: '累计费用',
    avgDuration: '平均耗时',
    // Messages
    enterApiKey: '请输入 API Key',
    querySuccess: '查询成功',
    queryFailed: '查询失败',
    queryFailedRetry: '查询失败，请稍后重试',
    noDailyUsage: '暂无按日用量数据',
  },

  // Setup Wizard
  setup: {
    title: 'Sub2API 安装向导',
    description: '配置您的 Sub2API 实例',
    database: {
      title: '数据库配置',
      description: '连接到您的 PostgreSQL 数据库',
      host: '主机',
      port: '端口',
      username: '用户名',
      password: '密码',
      databaseName: '数据库名称',
      sslMode: 'SSL 模式',
      passwordPlaceholder: '密码',
      ssl: {
        disable: '禁用',
        require: '要求',
        verifyCa: '验证 CA',
        verifyFull: '完全验证'
      }
    },
    redis: {
      title: 'Redis 配置',
      description: '连接到您的 Redis 服务器',
      host: '主机',
      port: '端口',
      username: '用户名（可选）',
      password: '密码（可选）',
      database: '数据库',
      usernamePlaceholder: '默认用户留空',
      passwordPlaceholder: '密码',
      enableTls: '启用 TLS',
      enableTlsHint: '连接 Redis 时使用 TLS（公共 CA 证书）'
    },
    admin: {
      title: '管理员账户',
      description: '创建您的管理员账户',
      email: '邮箱',
      password: '密码',
      confirmPassword: '确认密码',
      passwordPlaceholder: '至少 8 个字符',
      confirmPasswordPlaceholder: '确认密码',
      passwordMismatch: '密码不匹配'
    },
    ready: {
      title: '准备安装',
      description: '检查您的配置并完成安装',
      database: '数据库',
      redis: 'Redis',
      adminEmail: '管理员邮箱'
    },
    status: {
      testing: '测试中...',
      success: '连接成功',
      testConnection: '测试连接',
      installing: '安装中...',
      completeInstallation: '完成安装',
      completed: '安装完成！',
      redirecting: '正在跳转到登录页面...',
      restarting: '服务正在重启，请稍候...',
      timeout: '服务重启时间超出预期，请手动刷新页面。'
    }
  },

  // Common
}
