import { apiClient } from '../client'

export interface DefaultModelPricingItem {
  model: string
  provider: string
  mode: string
  input_cost_per_token: number | null
  input_cost_per_token_priority: number | null
  output_cost_per_token: number | null
  output_cost_per_token_priority: number | null
  cache_creation_input_token_cost: number | null
  cache_creation_input_token_cost_priority: number | null
  cache_creation_input_token_cost_above_1hr: number | null
  cache_read_input_token_cost: number | null
  cache_read_input_token_cost_priority: number | null
  output_cost_per_image: number | null
  output_cost_per_image_token: number | null
  long_context_input_token_threshold: number | null
  long_context_input_cost_multiplier: number | null
  long_context_output_cost_multiplier: number | null
  supports_service_tier: boolean
  supports_prompt_caching: boolean
  token_pricing_absent: boolean
}

export interface DefaultModelPricingResponse {
  items: DefaultModelPricingItem[]
  total: number
  page: number
  page_size: number
  providers: string[]
  modes: string[]
  status: { model_count: number; last_updated: string | null; local_hash: string }
}

export interface DefaultModelPricingParams {
  page?: number
  page_size?: number
  search?: string
  provider?: string
  mode?: string
  sort_by?: 'model' | 'provider' | 'mode'
  sort_order?: 'asc' | 'desc'
  signal?: AbortSignal
}

export async function list(params: DefaultModelPricingParams = {}): Promise<DefaultModelPricingResponse> {
  const { signal, ...query } = params
  const { data } = await apiClient.get<DefaultModelPricingResponse>('/admin/channels/default-model-pricing', { params: query, signal })
  return data
}

const defaultModelPricingAPI = { list }
export default defaultModelPricingAPI
