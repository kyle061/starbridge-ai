import type { OpenAIResponsesMode } from '@/types'

export interface OpenAIProviderPreset {
  id: string
  label: string
  baseUrl: string
  responsesMode: OpenAIResponsesMode
}

// Base URLs are editable. Model IDs and API keys always come from the operator.
// Chat-only providers must not be probed/routed as native Responses providers.
export const OPENAI_PROVIDER_PRESETS: readonly OpenAIProviderPreset[] = [
  { id: 'openai', label: 'OpenAI', baseUrl: 'https://api.openai.com', responsesMode: 'force_responses' },
  { id: 'deepseek', label: 'DeepSeek', baseUrl: 'https://api.deepseek.com', responsesMode: 'force_chat_completions' },
  { id: 'siliconflow', label: 'SiliconFlow', baseUrl: 'https://api.siliconflow.cn/v1', responsesMode: 'force_chat_completions' },
  { id: 'openrouter', label: 'OpenRouter', baseUrl: 'https://openrouter.ai/api/v1', responsesMode: 'force_chat_completions' },
  { id: 'ollama', label: 'Ollama', baseUrl: 'http://host.docker.internal:11434/v1', responsesMode: 'force_chat_completions' },
]
