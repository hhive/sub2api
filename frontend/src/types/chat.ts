export interface ChatModel {
  id: string
  name: string
  provider: string
}

export type ChatConsoleMode = 'chat' | 'image'

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface ChatCompletionRequest {
  model: string
  messages: ChatMessage[]
  stream?: boolean
}

export interface ChatCompletionChoice {
  index?: number
  message: ChatMessage
  finish_reason?: string
}

export interface ChatCompletionResponse {
  id: string
  object?: string
  created?: number
  model?: string
  choices: ChatCompletionChoice[]
}

export interface ImageGenerationRequest {
  model: string
  prompt: string
  response_format?: 'url' | 'b64_json'
  size?: string
  quality?: string
}

export interface ImageGenerationData {
  url?: string
  b64_json?: string
  revised_prompt?: string
}

export interface ImageGenerationResponse {
  created?: number
  model?: string
  data: ImageGenerationData[]
}
