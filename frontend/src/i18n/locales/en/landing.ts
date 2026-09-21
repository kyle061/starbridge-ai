export default {
  batchImageGuide: {
    title: 'Batch Image Generation',
    description: 'Submit multiple prompts in one job and download the generated images when complete'
  },
  // Home Page
  home: {
    viewOnGithub: 'View on GitHub',
    viewDocs: 'View Documentation',
    docs: 'Docs',
    switchToLight: 'Switch to Light Mode',
    switchToDark: 'Switch to Dark Mode',
    dashboard: 'Dashboard',
    login: 'Login',
    register: 'Sign Up Free',
    getStarted: 'Get Started',
    goToDashboard: 'Go to Dashboard',
    navigation: 'Main navigation',
    openMenu: 'Open menu',
    modelPricing: 'Models & pricing',
    channelStatus: 'Channel status',
    quickStart: 'Connection guide',
    heroEyebrow: 'Multi-model API service',
    overview: 'Connection, usage and status',
    balanceNote: 'All keys share your balance · Customer prices include billing multipliers',
    connection: {
      title: 'Connect your client',
      endpoint: 'API service address',
      copyEndpoint: 'Copy API service address',
      description: 'Choose “Use Key” on the API Keys page for the complete configuration and endpoint for your client.',
      action: 'Get a key and configuration'
    },
    faq: {
      title: 'Before you get started',
      subtitle: 'Understand your balance and set up your client in one place.',
      balance: {
        question: 'How do my account balance and API keys work together?',
        answer: 'All API keys share your available account balance. Creating a key does not add or require allocating balance. New requests stop when available balance runs out. Usage records show individual calls.'
      },
      pricing: {
        question: 'Do I need to apply the billing multiplier again?',
        answer: 'Customer prices already include the billing multiplier. Do not multiply them again. Prices and multipliers can vary by model; usage records show the actual charge. Input, output and cache usage have their corresponding rates.'
      },
      groups: {
        question: 'What are plans, groups and channel monitoring?',
        answer: 'Plans describe the balance, validity or model access you can purchase; see each plan for details. Groups determine which models a key can use and their billing rules. Use the default available group for a typical setup. Channel monitoring shows check results, latency and check times.'
      },
      client: {
        question: 'How do I connect Codex? Is CC-Switch required?',
        answer: 'Click “Use Key” on the API Keys page, select your client and copy its configuration. The site address and key are included. CC-Switch is an optional import tool; copying the configuration also works. No installation error dialog appears when CC-Switch is absent.'
      }
    },
    // User-focused value proposition
    heroSubtitle: 'One Key, All AI Models',
    seoTitle: 'Starbridge AI Multi-model AI API Gateway',
    heroDescription: 'Access OpenAI, DeepSeek, Claude and other models through one gateway. Copy client configurations from API Keys and review calls and charges in usage records. Model availability depends on your account permissions.',
    tags: {
      subscriptionToApi: 'Multi-model gateway',
      stickySession: 'Session Persistence',
      realtimeBilling: 'Pay As You Go'
    },
    // Pain points section
    painPoints: {
      title: 'Sound Familiar?',
      items: {
        expensive: {
          title: 'High Subscription Costs',
          desc: 'Paying for multiple AI subscriptions that add up every month'
        },
        complex: {
          title: 'Account Chaos',
          desc: 'Managing scattered accounts and API keys across different platforms'
        },
        unstable: {
          title: 'Service Interruptions',
          desc: 'Single accounts hitting rate limits and disrupting your workflow'
        },
        noControl: {
          title: 'No Usage Control',
          desc: "Can't track where your money goes or limit team member usage"
        }
      }
    },
    // Solutions section
    solutions: {
      title: 'We Solve These Problems',
      subtitle: 'Three simple steps to stress-free AI access'
    },
    workflow: {
      title: 'Get started in three steps',
      subtitle: 'Sign in, create a key, then copy the configuration into your client. All keys share your account balance.',
      steps: {
        account: {
          title: 'Sign in',
          description: 'Sign in or create a Starbridge AI account, then open your dashboard.'
        },
        key: {
          title: 'Create an API key',
          description: 'Create a key on the API Keys page and choose an available group.'
        },
        connect: {
          title: 'Copy and connect',
          description: 'Use “Use Key” to copy the endpoint and configuration, or import into Codex or CC-Switch.'
        }
      }
    },
    features: {
      unifiedGateway: 'One place to configure your clients',
      unifiedGatewayDesc: 'Get configurations for Codex, Claude Code and compatible clients, with the endpoint and key included.',
      multiAccount: 'Check your channel status',
      multiAccountDesc: 'Review availability, model response latency and check times to help diagnose service issues.',
      balanceQuota: 'Track every call and charge',
      balanceQuotaDesc: 'Share one account balance and review requests, tokens and billed costs. New requests stop when available balance runs out.'
    },
    // Comparison section
    comparison: {
      title: 'Why Choose Us?',
      headers: {
        feature: 'Comparison',
        official: 'Official Subscriptions',
        us: 'Our Platform'
      },
      items: {
        pricing: {
          feature: 'Pricing',
          official: 'Fixed monthly fee, pay even if unused',
          us: 'Pay only for what you use'
        },
        models: {
          feature: 'Model Selection',
          official: 'Single provider only',
          us: 'Switch between models freely'
        },
        management: {
          feature: 'Account Management',
          official: 'Manage each service separately',
          us: 'Unified key, one dashboard'
        },
        stability: {
          feature: 'Stability',
          official: 'Single account rate limits',
          us: 'Multi-account pool, auto-failover'
        },
        control: {
          feature: 'Usage Control',
          official: 'Not available',
          us: 'Quotas & detailed analytics'
        }
      }
    },
    providers: {
      title: 'Supported AI Models',
      description: 'One API, Multiple Choices',
      supported: 'Supported',
      soon: 'Soon',
      claude: 'Claude',
      gemini: 'Gemini',
      antigravity: 'Antigravity',
      more: 'More'
    },
    // CTA section
    cta: {
      title: 'Ready to Get Started?',
      description: 'Sign up now and get free trial credits to experience seamless AI access',
      button: 'Sign Up Free'
    },
    footer: {
      allRightsReserved: 'All rights reserved.'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key Usage',
    subtitle: 'Enter your API Key to view real-time spending and usage status',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: 'Query',
    querying: 'Querying...',
    privacyNote: 'Your Key is processed locally in the browser and will not be stored',
    dateRange: 'Date Range:',
    dateRangeToday: 'Today',
    dateRange7d: '7 Days',
    dateRange30d: '30 Days',
    dateRange90d: '90 Days',
    dateRangeCustom: 'Custom',
    apply: 'Apply',
    used: 'Used',
    detailInfo: 'Detail Information',
    tokenStats: 'Token Statistics',
    dailyDetail: 'Daily Detail',
    modelStats: 'Model Usage Statistics',
    // Table headers
    date: 'Date',
    model: 'Model',
    requests: 'Requests',
    inputTokens: 'Input Tokens',
    outputTokens: 'Output Tokens',
    cacheCreationTokens: 'Cache Creation',
    cacheReadTokens: 'Cache Read',
    cacheWriteTokens: 'Cache Write',
    totalTokens: 'Total Tokens',
    cost: 'Cost',
    // Status
    quotaMode: 'Key Quota Mode',
    walletBalance: 'Wallet Balance',
    // Ring card titles
    totalQuota: 'Total Quota',
    limit5h: '5-Hour Limit',
    limitDaily: 'Daily Limit',
    limit7d: '7-Day Limit',
    limitWeekly: 'Weekly Limit',
    limitMonthly: 'Monthly Limit',
    // Detail rows
    remainingQuota: 'Remaining Quota',
    expiresAt: 'Expires At',
    todayExpires: '(expires today)',
    daysLeft: '({days} days)',
    usedQuota: 'Used Quota',
    resetNow: 'Resetting soon',
    subscriptionType: 'Subscription Type',
    billingType: 'Billing Type',
    subscriptionExpires: 'Subscription Expires',
    // Usage stat cells
    todayRequests: 'Today Requests',
    todayInputTokens: 'Today Input',
    todayOutputTokens: 'Today Output',
    todayTokens: 'Today Tokens',
    todayCacheCreation: 'Today Cache Creation',
    todayCacheRead: 'Today Cache Read',
    todayCost: 'Today Cost',
    rpmTpm: 'RPM / TPM',
    totalRequests: 'Total Requests',
    totalInputTokens: 'Total Input',
    totalOutputTokens: 'Total Output',
    totalTokensLabel: 'Total Tokens',
    totalCacheCreation: 'Total Cache Creation',
    totalCacheRead: 'Total Cache Read',
    totalCost: 'Total Cost',
    avgDuration: 'Avg Duration',
    // Messages
    enterApiKey: 'Please enter an API Key',
    querySuccess: 'Query successful',
    queryFailed: 'Query failed',
    queryFailedRetry: 'Query failed, please try again later',
    noDailyUsage: 'No daily usage data',
  },

  // Setup Wizard
  setup: {
    title: 'Starbridge AI Setup',
    description: 'Configure your Starbridge AI instance',
    database: {
      title: 'Database Configuration',
      description: 'Connect to your PostgreSQL database',
      host: 'Host',
      port: 'Port',
      username: 'Username',
      password: 'Password',
      databaseName: 'Database Name',
      sslMode: 'SSL Mode',
      passwordPlaceholder: 'Password',
      ssl: {
        disable: 'Disable',
        require: 'Require',
        verifyCa: 'Verify CA',
        verifyFull: 'Verify Full'
      }
    },
    redis: {
      title: 'Redis Configuration',
      description: 'Connect to your Redis server',
      host: 'Host',
      port: 'Port',
      username: 'Username (optional)',
      password: 'Password (optional)',
      database: 'Database',
      usernamePlaceholder: 'Leave empty for default user',
      passwordPlaceholder: 'Password',
      enableTls: 'Enable TLS',
      enableTlsHint: 'Use TLS when connecting to Redis (public CA certs)'
    },
    admin: {
      title: 'Admin Account',
      description: 'Create your administrator account',
      email: 'Email',
      password: 'Password',
      confirmPassword: 'Confirm Password',
      passwordPlaceholder: 'Min 8 characters',
      confirmPasswordPlaceholder: 'Confirm password',
      passwordMismatch: 'Passwords do not match'
    },
    ready: {
      title: 'Ready to Install',
      description: 'Review your configuration and complete setup',
      database: 'Database',
      redis: 'Redis',
      adminEmail: 'Admin Email'
    },
    status: {
      testing: 'Testing...',
      success: 'Connection Successful',
      testConnection: 'Test Connection',
      installing: 'Installing...',
      completeInstallation: 'Complete Installation',
      completed: 'Installation completed!',
      redirecting: 'Redirecting to login page...',
      restarting: 'Service is restarting, please wait...',
      timeout: 'Service restart is taking longer than expected. Please refresh the page manually.'
    }
  },

  // Common
}
