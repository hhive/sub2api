import { apiClient } from './client'
import type {
  ChatCompletionRequest,
  ChatCompletionResponse,
  ChatModel,
  ImageGenerationRequest,
  ImageGenerationResponse,
} from '@/types/chat'

interface ChatModelsResponse {
  models: ChatModel[]
}

export async function getChatModels(): Promise<ChatModel[]> {
  const { data } = await apiClient.get<ChatModelsResponse>('/chat/models')
  return data.models
}

export async function getTextChatModels(): Promise<ChatModel[]> {
  const { data } = await apiClient.get<ChatModelsResponse>('/chat/models/chat')
  return data.models
}

export async function getImageGenerationModels(): Promise<ChatModel[]> {
  const { data } = await apiClient.get<ChatModelsResponse>('/chat/models/images')
  return data.models
}

export async function sendChatCompletion(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
  const { data } = await apiClient.post<ChatCompletionResponse>('/chat/completions', request)
  return data
}

export async function sendImageGeneration(request: ImageGenerationRequest): Promise<ImageGenerationResponse> {
  const { data } = await apiClient.post<ImageGenerationResponse>('/chat/images/generations', request)
  return data
}

export const chatAPI = {
  getModels: getChatModels,
  getTextModels: getTextChatModels,
  getImageModels: getImageGenerationModels,
  sendCompletion: sendChatCompletion,
  sendImageGeneration,
}
