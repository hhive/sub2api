import { apiClient } from './client'
import type { ChatCompletionRequest, ChatCompletionResponse, ChatModel } from '@/types/chat'

interface ChatModelsResponse {
  models: ChatModel[]
}

export async function getChatModels(): Promise<ChatModel[]> {
  const { data } = await apiClient.get<ChatModelsResponse>('/chat/models')
  return data.models
}

export async function sendChatCompletion(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
  const { data } = await apiClient.post<ChatCompletionResponse>('/chat/completions', request)
  return data
}

export const chatAPI = {
  getModels: getChatModels,
  sendCompletion: sendChatCompletion,
}
