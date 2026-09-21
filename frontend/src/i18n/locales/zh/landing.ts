export default {
  batchImageGuide: {
    title: '图片批量生成',
    description: '一次提交多条提示词，任务完成后可统一下载图片结果'
  },
  // Home Page
  home: {
    viewOnGithub: '在 GitHub 上查看',
    viewDocs: '查看文档',
    docs: '文档',
    switchToLight: '切换到浅色模式',
    switchToDark: '切换到深色模式',
    dashboard: '控制台',
    login: '登录',
    register: '免费注册',
    getStarted: '立即开始',
    goToDashboard: '进入控制台',
    navigation: '主导航',
    openMenu: '打开菜单',
    modelPricing: '模型与价格',
    channelStatus: '查看渠道状态',
    quickStart: '接入指南',
    heroEyebrow: '多模型 API 服务',
    overview: '接入、用量与状态',
    balanceNote: '所有密钥共用账户余额 · 客户端价格已包含计费倍率',
    connection: {
      title: '从这里连接你的客户端',
      endpoint: 'API 服务地址',
      copyEndpoint: '复制 API 服务地址',
      description: '在密钥页选择「使用密钥」，获取对应客户端的完整配置和接口地址。',
      action: '获取密钥与配置'
    },
    faq: {
      title: '开始前，你可能想了解',
      subtitle: '先了解额度和接入方式，配置一次即可开始使用。',
      balance: {
        question: '账户余额和 API Key 额度是什么关系？',
        answer: '所有 API Key 共用账户的可用余额，创建密钥不会增加余额，也不需要再次分配余额。可用余额耗尽后，新请求会停止；调用明细可在使用记录中查看。'
      },
      pricing: {
        question: '页面上的价格还需要乘以倍率吗？',
        answer: '用户页面展示的是已包含计费倍率的价格，不需要重复相乘。不同模型的价格与倍率可能不同，实际费用以调用记录为准；输入、输出与缓存按对应价格统计。'
      },
      groups: {
        question: '套餐、分组和渠道监控分别有什么用？',
        answer: '套餐说明可购买的额度、有效期或模型权限，以套餐详情为准。分组决定密钥可使用的模型范围和计费规则，常规接入使用默认可用分组。渠道监控展示检测结果、延迟与检测时间，帮助判断当前服务状态。'
      },
      client: {
        question: '怎样接入 Codex？必须安装 CC-Switch 吗？',
        answer: '在 API Keys 页面点击「使用密钥」，选择客户端后复制完整配置，配置中已填入本站地址与该密钥。CC-Switch 是可选的导入工具，也可以直接复制配置；未安装时不会弹出安装失败提示。'
      }
    },
    // 新增：面向用户的价值主张
    heroSubtitle: '一个密钥，畅用多个 AI 模型',
    seoTitle: 'Starbridge AI 多模型 AI API 中转站',
    heroDescription: '通过统一入口接入 OpenAI、DeepSeek、Claude 等模型。在密钥页复制客户端配置，在使用记录中查看每次调用与费用。可用模型以账户权限为准。',
    tags: {
      subscriptionToApi: '多模型中转',
      stickySession: '会话保持',
      realtimeBilling: '按量计费'
    },
    // 用户痛点区块
    painPoints: {
      title: '你是否也遇到这些问题？',
      items: {
        expensive: {
          title: '订阅费用高',
          desc: '每个 AI 服务都要单独订阅，每月支出越来越多'
        },
        complex: {
          title: '多账号难管理',
          desc: '不同平台的账号、密钥分散各处，管理起来很麻烦'
        },
        unstable: {
          title: '服务不稳定',
          desc: '单一账号容易触发限制，影响正常使用'
        },
        noControl: {
          title: '用量无法控制',
          desc: '不知道钱花在哪了，也无法限制团队成员的使用'
        }
      }
    },
    // 解决方案区块
    solutions: {
      title: '我们帮你解决',
      subtitle: '简单三步，开始省心使用 AI'
    },
    workflow: {
      title: '三步开始使用',
      subtitle: '登录、创建密钥，然后把配置复制到你的客户端。所有密钥共用账户余额，额度清晰可控。',
      steps: {
        account: {
          title: '登录账户',
          description: '登录或注册星桥 AI 账户，进入个人控制台。'
        },
        key: {
          title: '创建 API Key',
          description: '在 API Keys 页面创建密钥并选择可用分组。'
        },
        connect: {
          title: '复制配置并连接',
          description: '点击「使用密钥」复制端点和配置，也可导入 Codex 或 CC-Switch。'
        }
      }
    },
    features: {
      unifiedGateway: '一处配置，多种客户端',
      unifiedGatewayDesc: '为 Codex、Claude Code 与兼容客户端生成对应配置，地址与密钥一起复制。',
      multiAccount: '渠道状态随时可查',
      multiAccountDesc: '查看渠道可用情况、模型响应延迟与检测时间，遇到问题先定位服务状态。',
      balanceQuota: '每次调用，费用有据',
      balanceQuotaDesc: '统一账户余额，查看请求、Token 和实付费用；可用余额耗尽后停止新请求。'
    },
    // 优势对比
    comparison: {
      title: '为什么选择我们？',
      headers: {
        feature: '对比项',
        official: '官方订阅',
        us: '本平台'
      },
      items: {
        pricing: {
          feature: '付费方式',
          official: '固定月费，用不完也付',
          us: '按量付费，用多少付多少'
        },
        models: {
          feature: '模型选择',
          official: '单一服务商',
          us: '多模型随意切换'
        },
        management: {
          feature: '账号管理',
          official: '每个服务单独管理',
          us: '统一密钥，一站管理'
        },
        stability: {
          feature: '服务稳定性',
          official: '单账号易触发限制',
          us: '多账号池，自动切换'
        },
        control: {
          feature: '用量控制',
          official: '无法限制',
          us: '可设配额、查明细'
        }
      }
    },
    providers: {
      title: '已支持的 AI 模型',
      description: '一个 API，多种选择',
      supported: '已支持',
      soon: '即将推出',
      claude: 'Claude',
      gemini: 'Gemini',
      antigravity: 'Antigravity',
      more: '更多'
    },
    // CTA 区块
    cta: {
      title: '准备好开始了吗？',
      description: '注册即可获得免费试用额度，体验一站式 AI 服务',
      button: '免费注册'
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
    title: 'Starbridge AI 安装向导',
    description: '配置您的 Starbridge AI 实例',
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
