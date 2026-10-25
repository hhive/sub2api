export default {
  batchImageGuide: {
    title: 'Batch Image Generation',
    description: 'Submit multiple prompts in one job and download the generated images when complete'
  },
  // Home Page
  home: {
    viewDocs: 'View Documentation',
    docs: 'Docs',
    switchToLight: 'Switch to Light Mode',
    switchToDark: 'Switch to Dark Mode',
    dashboard: 'Dashboard',
    login: 'Login',
    goToDashboard: 'Go to Dashboard',
    // Immersive hero
    hero: {
      primaryCta: 'Sign in',
      primaryCtaAuthed: 'Open dashboard',
      secondaryCta: 'Take a look',
      moreMenu: 'More',
      reelLabel: 'Model plaza',
      contact: 'Contact sales',
      contactCopied: 'Contact copied',
      contactCopyFailed: 'Copy failed, please select the contact manually',
      carouselPrev: 'Previous headline',
      carouselNext: 'Next headline',
      slides: {
        s1: {
          prefix: 'One key',
          verb: 'to every AI',
          subtitle: 'From a prompt to a call.\nWorkbench, compute and model market in one place.',
          left: {
            kicker: 'Access / Models',
            title: 'Claude · GPT · Gemini',
            desc: 'One key covers the major vendors. Switching is just a model name.'
          },
          right: {
            kicker: 'Billing / Usage',
            title: 'Metered and clear',
            desc: 'Pay for what you use, set quota caps as you like.'
          }
        },
        s2: {
          prefix: 'Subscribe to API',
          verb: 'one account goes far',
          subtitle: 'Your subscription stops idling.\nTurn it into API keys and plug in your own tools.',
          left: {
            kicker: 'Account / Migration',
            title: 'Plans become keys',
            desc: 'Existing subscriptions convert directly, no double billing.'
          },
          right: {
            kicker: 'Session / Stability',
            title: 'Sticky sessions',
            desc: 'Long contexts stay alive across multi-turn work.'
          }
        },
        s3: {
          prefix: 'Model plaza',
          verb: 'prices at a glance',
          subtitle: 'Image, video and chat models side by side.\nTiered pricing lives on the page.',
          left: {
            kicker: 'Catalog / Models',
            title: 'One catalog, everywhere',
            desc: 'Images by size and quality, video by second or by clip.'
          },
          right: {
            kicker: 'Pricing / Transparent',
            title: 'Cost and price kept apart',
            desc: 'Every price change leaves a trace.'
          }
        },
        s4: {
          prefix: 'Console',
          verb: 'usage you can reconcile',
          subtitle: 'Balance, quota and call history refresh live.\nSee where it went and what is left.',
          left: {
            kicker: 'Board / Usage',
            title: 'Live balance and quota',
            desc: 'Broken down by key and model, drillable to each call.'
          },
          right: {
            kicker: 'Keys / Control',
            title: 'Groups and caps, self-served',
            desc: 'One key, one ceiling, blocked automatically beyond it.'
          }
        },
        s5: {
          prefix: 'Team',
          verb: 'multi-user with clear roles',
          subtitle: 'Members, keys and quota in layers.\nWho used what stays on record.',
          left: {
            kicker: 'Members / Roles',
            title: 'Grant only what is needed',
            desc: 'Admins, operators and users each see their own scope.'
          },
          right: {
            kicker: 'Audit / Logs',
            title: 'Every call traceable',
            desc: 'Logins, password changes and calls all audited.'
          }
        }
      }
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
    title: 'Sub2API Setup',
    description: 'Configure your Sub2API instance',
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
