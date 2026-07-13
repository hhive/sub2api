export default {
  videoPlayground: {
    title: 'Media Site Video Models',
    description: 'Configure media-site video models, upstream protocols, per-call pricing, and failure refunds',
    createModel: 'Create model',
    editModel: 'Edit model',
    reuse: 'Reuse',
    reuseFrom: 'Reuse existing model',
    reuseFromPlaceholder: 'Select an existing model to fill this form',
    deleteTitle: 'Delete video model',
    enabledToast: 'Video model enabled',
    disabledToast: 'Video model disabled',
    loadFailed: 'Failed to load video models',
    saveFailed: 'Failed to save video model',
    deleteFailed: 'Failed to delete video model',
    deleteConfirm: 'Delete video model "{name}"?',
    callRecords: {
      button: 'Call records',
      title: 'Call records',
      description: 'Latest media-site video upstream calls in reverse chronological order.',
      loadFailed: 'Failed to load call records',
      previous: 'Previous',
      next: 'Next',
      pageInfo: 'Page {page}, {total} records',
      columns: {
        createdAt: 'Time',
        task: 'Task',
        user: 'User',
        model: 'Model',
        endpoint: 'Endpoint',
        httpStatus: 'HTTP',
        elapsed: 'Elapsed',
        responseBytes: 'Response size',
        error: 'Error'
      }
    },
    taskRecords: {
      button: 'Task records',
      title: 'Video task records',
      detailTitle: 'Video task details',
      allStatuses: 'All statuses',
      loadFailed: 'Failed to load video tasks',
      previous: 'Previous',
      next: 'Next',
      pageInfo: 'Page {page}, {total} records',
      status: 'Status',
      error: 'Failure reason',
      refund: 'Refund',
      upstream: 'Upstream task',
      columns: { task: 'Task' }
    },
    keyConfigured: 'Configured: {mask}. Leave blank to keep unchanged',
    keyNotCopied: 'Original key is configured ({mask}); reused models do not copy secrets',
    invalidJSON: '{field} is not valid JSON',
    jsonMustBeObject: '{field} must be a JSON object',
    billingModes: {
      balance_prepaid: 'Prepaid balance per call'
    },
    apiModes: {
      openai_videos: 'OpenAI Videos API',
      openai_videos_v2: 'OpenAI Videos API2',
      seedance_content_generation: 'Seedance Content Generation API'
    },
    columns: {
      name: 'Model',
      apiMode: 'API mode',
      provider: 'Provider',
      upstream: 'Upstream',
      price: 'Price',
      billingMode: 'Billing',
      refund: 'Refund',
      enabled: 'Enabled'
    },
    fields: {
      displayName: 'Display name',
      model: 'Model ID',
      apiMode: 'API mode',
      providerName: 'Provider',
      upstreamBaseURL: 'Upstream domain',
      upstreamAPIKey: 'Upstream API Key',
      priceQuota: 'Fixed per-call price',
      billingMode: 'Billing mode',
      refundEnabled: 'Refund failed generations automatically',
      timeoutSeconds: 'Timeout seconds',
      sortOrder: 'Sort order',
      enabled: 'Enable model'
    }
  },
  mediaPlaygroundImage: {
    title: 'Media Site Image Models',
    description: 'Configure media-site image models, upstream domains, 1k/2k/4k pricing, and enabled state',
    createModel: 'Create model',
    editModel: 'Edit model',
    reuse: 'Reuse',
    reuseFrom: 'Reuse existing model',
    reuseFromPlaceholder: 'Select an existing model to fill this form',
    deleteTitle: 'Delete image model',
    keyConfigured: 'Configured: {mask}. Leave blank to keep unchanged',
    keyNotCopied: 'Original key is configured ({mask}); reused models do not copy secrets',
    enabledToast: 'Image model enabled',
    disabledToast: 'Image model disabled',
    loadFailed: 'Failed to load image models',
    saveFailed: 'Failed to save image model',
    deleteFailed: 'Failed to delete image model',
    deleteConfirm: 'Delete image model "{name}"?',
    callRecords: {
      button: 'Call records',
      title: 'Call records',
      description: 'Latest media-site image upstream calls in reverse chronological order.',
      loadFailed: 'Failed to load call records',
      previous: 'Previous',
      next: 'Next',
      pageInfo: 'Page {page}, {total} records',
      columns: {
        createdAt: 'Time',
        task: 'Task',
        user: 'User / API key',
        model: 'Model',
        upstream: 'Upstream',
        status: 'Status',
        httpStatus: 'HTTP',
        responseBytes: 'Response size',
        imageCount: 'Images',
        error: 'Error'
      }
    },
    sizeRequired: 'Select at least one supported size',
    columns: {
      name: 'Model',
      apiMode: 'API mode',
      provider: 'Provider',
      upstream: 'Upstream',
      prices: 'Tier prices',
      sizes: 'Sizes',
      sortOrder: 'Sort order',
      health: 'Health',
      enabled: 'Enabled'
    },
    fields: {
      displayName: 'Display name',
      model: 'Model ID',
      apiMode: 'API mode',
      providerName: 'Provider',
      upstreamBaseURL: 'Upstream domain',
      upstreamAPIKey: 'Upstream API Key',
      price1k: '1k price',
      price2k: '2k price',
      price4k: '4k price',
      supportedSizes: 'Supported sizes',
      timeoutSeconds: 'Timeout seconds',
      sortOrder: 'Sort order',
      enabled: 'Enable model',
      fallbackToResponses: 'Retry once with Responses API config when generation fails'
    },
    apiModes: {
      images: 'Images API',
      responses: 'Responses API',
      geminiGenerateContent: 'Gemini GenerateContent API'
    },
    health: {
      cooldownUntil: 'Cooldown until',
      failures: 'Failures',
      cooldowns: 'Cooldowns',
      halfOpenAttempts: 'Half-open',
      lastError: 'Last error:',
      status: {
        available: 'Available',
        temporaryUnavailable: 'Cooling down',
        halfOpen: 'Half-open probe',
        disabled: 'Disabled'
      }
    },
    probeRuns: {
      button: 'Probe records',
      runButton: 'Run probe',
      singleRunButton: 'Probe',
      title: 'Probe records',
      description: 'Latest media-site image model probe attempts in reverse chronological order.',
      loadFailed: 'Failed to load probe records',
      runSuccess: 'Probe started',
      runFailed: 'Failed to run probe',
      singleRunSuccess: '{name} probe started',
      singleRunFailed: 'Failed to run model probe',
      previous: 'Previous',
      next: 'Next',
      pageInfo: 'Page {page}, {total} records',
      columns: {
        createdAt: 'Time',
        model: 'Model',
        apiMode: 'API mode',
        upstream: 'Upstream',
        attempt: 'Attempt',
        status: 'Status',
        httpStatus: 'HTTP',
        elapsed: 'Elapsed',
        responseBytes: 'Response size',
        imageCount: 'Images',
        error: 'Error'
      },
      status: {
        success: 'Success',
        failed: 'Failed'
      }
    }
  }
}
