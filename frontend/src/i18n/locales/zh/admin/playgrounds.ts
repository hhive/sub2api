export default {
  videoPlayground: {
    title: '视频模型配置',
    description: '配置独立视频站可用模型、上游域名、单次价格与失败退款策略',
    createModel: '新建模型',
    editModel: '编辑模型',
    reuse: '复用',
    reuseFrom: '从已有模型复用',
    reuseFromPlaceholder: '选择已有模型快速填充配置',
    deleteTitle: '删除视频模型',
    enabledToast: '视频模型已启用',
    disabledToast: '视频模型已停用',
    loadFailed: '加载视频模型失败',
    saveFailed: '保存视频模型失败',
    deleteFailed: '删除视频模型失败',
    deleteConfirm: '确定删除视频模型“{name}”？',
    callRecords: {
      button: '调用记录',
      title: '调用记录',
      description: '按时间倒序展示视频站最近请求上游的记录。',
      loadFailed: '加载调用记录失败',
      previous: '上一页',
      next: '下一页',
      pageInfo: '第 {page} 页，共 {total} 条',
      columns: {
        createdAt: '时间',
        task: '任务',
        user: '用户',
        model: '模型',
        endpoint: '端点',
        httpStatus: 'HTTP',
        elapsed: '耗时',
        responseBytes: '响应大小',
        error: '错误'
      }
    },
    keyConfigured: '已配置：{mask}，留空则不变',
    keyNotCopied: '原配置已设置 Key（{mask}），复用时不会复制密钥',
    invalidJSON: '{field} 不是有效 JSON',
    jsonMustBeObject: '{field} 必须是 JSON 对象',
    billingModes: {
      balance_prepaid: '按次预扣余额'
    },
    templates: {
      baseModel: '基础模型'
    },
    modelKinds: {
      t2v: '文生视频',
      i2v: '图生视频',
      reference_video: '参考图视频',
      extend: '视频延长'
    },
    columns: {
      name: '模型',
      apiMode: 'API 模式',
      provider: '供应商',
      upstream: '上游域名',
      price: '单次价格',
      billingMode: '扣款方式',
      refund: '失败退款',
      enabled: '启用'
    },
    fields: {
      displayName: '显示名',
      model: '模型 ID',
      studioTemplate: '即梦能力模板',
      modelKind: '模型类型',
      providerName: '供应商',
      upstreamBaseURL: '上游域名',
      upstreamAPIKey: '上游 API Key',
      priceQuota: '固定单次价格',
      billingMode: '扣款方式',
      refundEnabled: '视频生成失败时自动退款',
      timeoutSeconds: '超时秒数',
      sortOrder: '排序',
      enabled: '启用模型',
      inputSchemaJSON: '输入 Schema JSON',
      payloadMappingJSON: 'Payload 映射 JSON'
    }
  },
  imagePlayground: {
    title: '图片模型配置',
    description: '配置独立图片站可用模型、上游域名、1k/2k/4k 价格与启用状态',
    createModel: '新建模型',
    editModel: '编辑模型',
    reuse: '复用',
    reuseFrom: '从已有模型复用',
    reuseFromPlaceholder: '选择已有模型快速填充配置',
    deleteTitle: '删除图片模型',
    keyConfigured: '已配置：{mask}，留空则不变',
    keyNotCopied: '原配置已设置 Key（{mask}），复用时不会复制密钥',
    enabledToast: '图片模型已启用',
    disabledToast: '图片模型已停用',
    loadFailed: '加载图片模型失败',
    saveFailed: '保存图片模型失败',
    deleteFailed: '删除图片模型失败',
    deleteConfirm: '确定删除图片模型“{name}”？',
    callRecords: {
      button: '调用记录',
      title: '调用记录',
      description: '按时间倒序展示图片站最近请求上游的记录。',
      loadFailed: '加载调用记录失败',
      previous: '上一页',
      next: '下一页',
      pageInfo: '第 {page} 页，共 {total} 条',
      columns: {
        createdAt: '时间',
        task: '任务',
        user: '用户 / API Key',
        model: '模型',
        upstream: '上游域名',
        status: '状态',
        httpStatus: 'HTTP',
        responseBytes: '响应大小',
        imageCount: '图片数',
        error: '错误'
      }
    },
    sizeRequired: '至少选择一个支持尺寸',
    columns: {
      name: '模型',
      apiMode: 'API 模式',
      provider: '供应商',
      upstream: '上游域名',
      prices: '档位价格',
      sizes: '尺寸',
      sortOrder: '排序',
      health: '健康状态',
      enabled: '启用'
    },
    fields: {
      displayName: '显示名',
      model: '模型 ID',
      apiMode: 'API 模式',
      providerName: '供应商',
      upstreamBaseURL: '上游域名',
      upstreamAPIKey: '上游 API Key',
      price1k: '1k 价格',
      price2k: '2k 价格',
      price4k: '4k 价格',
      supportedSizes: '支持尺寸',
      timeoutSeconds: '超时秒数',
      sortOrder: '排序',
      enabled: '启用模型',
      fallbackToResponses: '生成失败时使用 Responses API 配置兜底重试一次'
    },
    apiModes: {
      images: 'Images API',
      responses: 'Responses API',
      geminiGenerateContent: 'Gemini GenerateContent API'
    },
    health: {
      cooldownUntil: '冷却至',
      failures: '失败',
      cooldowns: '冷却',
      halfOpenAttempts: '半开',
      lastError: '最后错误：',
      status: {
        available: '可用',
        temporaryUnavailable: '冷却中',
        halfOpen: '半开探测',
        disabled: '已禁用'
      }
    },
    probeRuns: {
      button: '探测记录',
      runButton: '主动探测',
      singleRunButton: '探测',
      title: '探测记录',
      description: '按时间倒序展示图片站最近的模型探测 attempt 记录。',
      loadFailed: '加载探测记录失败',
      runSuccess: '主动探测已开始',
      runFailed: '主动探测失败',
      singleRunSuccess: '{name} 探测已开始',
      singleRunFailed: '模型探测失败',
      previous: '上一页',
      next: '下一页',
      pageInfo: '第 {page} 页，共 {total} 条',
      columns: {
        createdAt: '时间',
        model: '模型',
        apiMode: 'API 模式',
        upstream: '上游域名',
        attempt: '轮次',
        status: '状态',
        httpStatus: 'HTTP',
        elapsed: '耗时',
        responseBytes: '响应大小',
        imageCount: '图片数',
        error: '错误'
      },
      status: {
        success: '成功',
        failed: '失败'
      }
    }
  }
}
