// =====================
// 模型列表（硬编码，与 new-api 一致）
// =====================

// OpenAI 当前支持的 GPT 系列。
// 这里只放可直接用于账号白名单/映射的公开模型，不把旧版、内部探针或
// 未确认支持的预览别名混进通用模型选择器。
const openaiModels = [
  // GPT-5.5 系列
  'gpt-5.5',
  // GPT-5.6 系列
  'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna',
  // GPT-6 系列
  'gpt-6-astra', 'gpt-6-sol', 'gpt-6-luna',
  // 当前支持的 GPT Image 系列
  'gpt-image-2', 'gpt-image-2.5-flare', 'gpt-image-2.5-sunburst'
]

// Anthropic Claude：只保留当前公开目录中的稳定模型 ID。
export const claudeModels = [
  'claude-fable-5-1', 'claude-fable-5',
  'claude-opus-4-5-20251101', 'claude-opus-4-6', 'claude-opus-4-7',
  'claude-opus-4-8', 'claude-opus-5',
  'claude-sonnet-4-5-20250929', 'claude-sonnet-4-6', 'claude-sonnet-5',
  'claude-haiku-4-5-20251001'
]

// Google Gemini
const geminiModels = [
  'gemini-3.1-flash-image',
  'gemini-2.5-flash-image',
  'gemini-3.5-flash',
  'gemini-2.5-flash',
  'gemini-2.5-pro'
]

// Antigravity 官方支持的模型（精确匹配）
// 基于官方 API 返回的模型列表，只支持 Claude 4.5+ 和 Gemini 2.5+
const antigravityModels = [
  // Claude 4.5+ 系列
  'claude-fable-5-1',
  'claude-fable-5',
  'claude-opus-4-6',
  'claude-opus-4-6-thinking',
  'claude-opus-4-7',
  'claude-opus-4-8',
  'claude-opus-4-5-thinking',
  'claude-sonnet-4-6',
  'claude-sonnet-4-5',
  'claude-sonnet-4-5-thinking',
  // Gemini 2.5 系列
  'gemini-3.1-flash-image',
  'gemini-2.5-flash-image',
  'gemini-2.5-flash',
  'gemini-2.5-flash-lite',
  'gemini-2.5-flash-thinking',
  'gemini-2.5-pro',
  // Gemini 3 系列
  'gemini-3-flash',
  'gemini-3-pro-high',
  'gemini-3-pro-low',
  // Gemini 3.1 系列
  'gemini-3.1-pro-high',
  'gemini-3.1-pro-low',
  // Gemini 3.6+ 是 Antigravity 当前公开的分档模型。
  'gemini-3.6-flash', 'gemini-3.6-flash-high', 'gemini-3.6-flash-low',
  'gemini-3.6-flash-medium', 'gemini-3.6-flash-tiered',
  'gemini-3.7-flash', 'gemini-3.7-flash-high', 'gemini-3.7-flash-low',
  'gemini-3.7-flash-medium', 'gemini-3.7-flash-tiered',
  'gemini-3.8-flash', 'gemini-3.8-flash-high', 'gemini-3.8-flash-low',
  'gemini-3.8-flash-medium', 'gemini-3.8-flash-tiered'
]

// 智谱 GLM
const zhipuModels = [
  'glm-4.5', 'glm-4.5-air', 'glm-4.5-flash',
  'glm-4.6', 'glm-4.7', 'glm-4.7-flash',
  'glm-5', 'glm-5-turbo', 'glm-5.1', 'glm-5.2',
  'glm-5.3', 'glm-5.3-flash'
]

// 阿里 通义千问
const qwenModels = [
  'qwen-turbo', 'qwen-plus', 'qwen-max', 'qwen-long', 'qwen3-235b-a22b'
]

// DeepSeek
const deepseekModels = [
  'deepseek-chat', 'deepseek-reasoner',
  'deepseek-v4-pro', 'deepseek-v4-flash', 'deepseek-flash'
]

// Mistral
const mistralModels = [
  'mistral-small-latest', 'mistral-medium-latest', 'mistral-large-latest',
  'codestral-latest', 'pixtral-large-latest'
]

// Meta Llama
const metaModels = [
  'llama-3.3-70b-instruct'
]

// xAI Grok
const xaiModels = [
  'grok-4.6',
  'grok-4.5',
  'grok-4.3',
  'grok-build-0.1',
  'grok-composer-2.5-fast',
  'grok-4.20-0309-reasoning',
  'grok-4.20-0309-non-reasoning',
  'grok-4.20-multi-agent-0309',
  'grok-imagine-image-quality',
  'grok-imagine-image',
  'grok-imagine-image-2.0',
  'grok-imagine-video',
  'grok-imagine-video-1.5'
]

// Cohere
const cohereModels = [
  'command-a-03-2025',
  'command-r', 'command-r-plus'
]

// Yi (01.AI)
const yiModels = [
  'yi-large', 'yi-large-turbo', 'yi-vision'
]

// Moonshot/Kimi
const moonshotModels = [
  'kimi-latest',
  'kimi-for-coding',
  'kimi-k2'
]

// 字节跳动 豆包
const doubaoModels = [
  'doubao-1.5-pro-256k', 'doubao-1.5-pro-32k', 'doubao-1.5-lite-32k',
  'doubao-1.5-pro-vision-32k', 'doubao-1.5-thinking-pro'
]

// MiniMax
const minimaxModels = [
  'MiniMax-M3',
  'MiniMax-M2.7',
  'MiniMax-M2.7-highspeed',
  'MiniMax-M2.5',
  'MiniMax-M2.5-highspeed',
  'MiniMax-M2.1',
  'MiniMax-M2.1-highspeed'
]

// 百度 文心
const baiduModels = [
  'ernie-4.0-8k-latest', 'ernie-4.0-turbo-8k',
  'ernie-speed-128k', 'ernie-speed-pro-128k', 'ernie-lite-pro-128k'
]

// 讯飞 星火
const sparkModels = [
  'spark-desk-v4.0', 'spark-pro', 'spark-max', 'spark-ultra'
]

// 腾讯 混元
const hunyuanModels = [
  'hunyuan-pro', 'hunyuan-turbo', 'hunyuan-large',
  'hunyuan-vision', 'hunyuan-code'
]

// Perplexity
const perplexityModels = [
  'sonar', 'sonar-pro', 'sonar-reasoning'
]

// OpenCode Go 的公开模型目录。只保留当前可用的正式模型，不包含 preview、alpha
// 或贡献者实验模型。
const opencodeGoModels = [
  'grok-4.6', 'gpt-5.6-luna',
  'glm-5.3-flash', 'glm-5.3', 'glm-5.2', 'glm-5.1',
  'kimi-k3', 'kimi-k2.7-code', 'kimi-k2.6',
  'longcat-2.0',
  'deepseek-v4-pro', 'deepseek-v4-flash',
  'mimo-v2.5', 'mimo-v2.5-pro',
  'minimax-m3', 'minimax-m2.7', 'minimax-m2.5',
  'qwen3.8-max', 'qwen3.8-flash', 'qwen3.7-max', 'qwen3.7-plus', 'qwen3.6-plus',
  'hy3'
]

// 所有模型（去重）
const allModelsList: string[] = Array.from(new Set([
  ...openaiModels,
  ...claudeModels,
  ...geminiModels,
  ...zhipuModels,
  ...qwenModels,
  ...deepseekModels,
  ...mistralModels,
  ...metaModels,
  ...xaiModels,
  ...cohereModels,
  ...yiModels,
  ...moonshotModels,
  ...doubaoModels,
  ...minimaxModels,
  ...baiduModels,
  ...sparkModels,
  ...hunyuanModels,
  ...perplexityModels,
  ...opencodeGoModels
]))

// 转换为下拉选项格式
export const allModels = allModelsList.map(m => ({ value: m, label: m }))

// =====================
// 预设映射
// =====================

const anthropicPresetMappings = [
  { label: 'Fable 5.1', from: 'claude-fable-5-1', to: 'claude-fable-5-1', color: 'bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400' },
  { label: 'Fable 5', from: 'claude-fable-5', to: 'claude-fable-5', color: 'bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400' },
  { label: 'Sonnet 5', from: 'claude-sonnet-5', to: 'claude-sonnet-5', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: 'Sonnet 4.5', from: 'claude-sonnet-4-5-20250929', to: 'claude-sonnet-4-5-20250929', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: 'Sonnet 4.6', from: 'claude-sonnet-4-6', to: 'claude-sonnet-4-6', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: 'Opus 4.5', from: 'claude-opus-4-5-20251101', to: 'claude-opus-4-5-20251101', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: 'Opus 4.6', from: 'claude-opus-4-6', to: 'claude-opus-4-6', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: 'Opus 4.7', from: 'claude-opus-4-7', to: 'claude-opus-4-7', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: 'Opus 4.8', from: 'claude-opus-4-8', to: 'claude-opus-4-8', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: 'Opus 5', from: 'claude-opus-5', to: 'claude-opus-5', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: 'Haiku 4.5', from: 'claude-haiku-4-5-20251001', to: 'claude-haiku-4-5-20251001', color: 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400' },
  { label: 'Opus->Sonnet', from: 'claude-opus-4-6', to: 'claude-sonnet-4-5-20250929', color: 'bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400' }
]

const openaiPresetMappings = [
  { label: 'GPT-5.5', from: 'gpt-5.5', to: 'gpt-5.5', color: 'bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400' },
  { label: 'GPT-5.6 Sol', from: 'gpt-5.6-sol', to: 'gpt-5.6-sol', color: 'bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400' },
  { label: 'GPT-5.6 Terra', from: 'gpt-5.6-terra', to: 'gpt-5.6-terra', color: 'bg-lime-100 text-lime-700 hover:bg-lime-200 dark:bg-lime-900/30 dark:text-lime-400' },
  { label: 'GPT-5.6 Luna', from: 'gpt-5.6-luna', to: 'gpt-5.6-luna', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' },
  { label: 'GPT-6 Astra', from: 'gpt-6-astra', to: 'gpt-6-astra', color: 'bg-fuchsia-100 text-fuchsia-700 hover:bg-fuchsia-200 dark:bg-fuchsia-900/30 dark:text-fuchsia-400' },
  { label: 'GPT-6 Sol', from: 'gpt-6-sol', to: 'gpt-6-sol', color: 'bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/30 dark:text-violet-400' },
  { label: 'GPT-6 Luna', from: 'gpt-6-luna', to: 'gpt-6-luna', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' }
]

const geminiPresetMappings = [
  { label: '2.5 Flash', from: 'gemini-2.5-flash', to: 'gemini-2.5-flash', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: '2.5 Image', from: 'gemini-2.5-flash-image', to: 'gemini-2.5-flash-image', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' },
  { label: '2.5 Pro', from: 'gemini-2.5-pro', to: 'gemini-2.5-pro', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: '3.5 Flash', from: 'gemini-3.5-flash', to: 'gemini-3.5-flash', color: 'bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/30 dark:text-violet-400' },
  { label: '3.1 Image', from: 'gemini-3.1-flash-image', to: 'gemini-3.1-flash-image', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' }
]

const grokPresetMappings = [
  { label: 'Grok 4.6', from: 'grok-4.6', to: 'grok-4.6', color: 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-slate-800/50 dark:text-slate-300' },
  { label: 'Grok 4.5', from: 'grok-4.5', to: 'grok-4.5', color: 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-slate-800/50 dark:text-slate-300' },
  { label: 'Grok 4.3', from: 'grok-4.3', to: 'grok-4.3', color: 'bg-slate-100 text-slate-700 hover:bg-slate-200 dark:bg-slate-800/50 dark:text-slate-300' },
  { label: 'Grok Build', from: 'grok-build-0.1', to: 'grok-build-0.1', color: 'bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { label: 'Grok Composer', from: 'grok-composer-2.5-fast', to: 'grok-composer-2.5-fast', color: 'bg-teal-100 text-teal-700 hover:bg-teal-200 dark:bg-teal-900/30 dark:text-teal-400' },
  { label: '4.20 Reasoning', from: 'grok-4.20-0309-reasoning', to: 'grok-4.20-0309-reasoning', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: '4.20 Non Reasoning', from: 'grok-4.20-0309-non-reasoning', to: 'grok-4.20-0309-non-reasoning', color: 'bg-violet-100 text-violet-700 hover:bg-violet-900/30 dark:text-violet-400' },
  { label: 'Imagine Image', from: 'grok-imagine-image-quality', to: 'grok-imagine-image-quality', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' },
  { label: 'Imagine Video', from: 'grok-imagine-video-1.5', to: 'grok-imagine-video-1.5', color: 'bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400' }
]

// Antigravity 预设映射（支持通配符）
const antigravityPresetMappings = [
  // Claude 通配符映射
  { label: 'Claude→Sonnet', from: 'claude-*', to: 'claude-sonnet-4-5', color: 'bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400' },
  { label: 'Fable 5.1', from: 'claude-fable-5-1', to: 'claude-fable-5-1', color: 'bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400' },
  { label: 'Fable 5', from: 'claude-fable-5', to: 'claude-fable-5', color: 'bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400' },
  { label: 'Sonnet→Sonnet', from: 'claude-sonnet-*', to: 'claude-sonnet-4-5', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: 'Opus→Opus', from: 'claude-opus-*', to: 'claude-opus-4-6-thinking', color: 'bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400' },
  { label: 'Haiku→Sonnet', from: 'claude-haiku-*', to: 'claude-sonnet-4-5', color: 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400' },
  { label: 'Sonnet4.5→4.6', from: 'claude-sonnet-4-5-20250929', to: 'claude-sonnet-4-6', color: 'bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { label: '3-Pro-Low→3.1-Pro-Low', from: 'gemini-3-pro-low', to: 'gemini-3.1-pro-low', color: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400' },
  { label: '3.1-Pro-High透传', from: 'gemini-3.1-pro-high', to: 'gemini-3.1-pro-high', color: 'bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400' },
  { label: '3.1-Pro-Low透传', from: 'gemini-3.1-pro-low', to: 'gemini-3.1-pro-low', color: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400' },
  { label: '3-Pro-Image→3.1', from: 'gemini-3-pro-image', to: 'gemini-3.1-flash-image', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' },
  // Gemini 通配符映射
  { label: 'Gemini 3→Flash', from: 'gemini-3*', to: 'gemini-3-flash', color: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400' },
  { label: 'Gemini 2.5→Flash', from: 'gemini-2.5*', to: 'gemini-2.5-flash', color: 'bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400' },
  { label: '2.5-Flash-Image透传', from: 'gemini-2.5-flash-image', to: 'gemini-2.5-flash-image', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' },
  { label: '3.1-Flash-Image透传', from: 'gemini-3.1-flash-image', to: 'gemini-3.1-flash-image', color: 'bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400' },
  { label: '3-Flash透传', from: 'gemini-3-flash', to: 'gemini-3-flash', color: 'bg-lime-100 text-lime-700 hover:bg-lime-200 dark:bg-lime-900/30 dark:text-lime-400' },
  { label: '2.5-Flash-Lite透传', from: 'gemini-2.5-flash-lite', to: 'gemini-2.5-flash-lite', color: 'bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400' },
  // 精确映射
  { label: 'Sonnet 4.6', from: 'claude-sonnet-4-6', to: 'claude-sonnet-4-6', color: 'bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { label: 'Sonnet 4.5', from: 'claude-sonnet-4-5', to: 'claude-sonnet-4-5', color: 'bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { label: 'Opus 4.6', from: 'claude-opus-4-6', to: 'claude-opus-4-6-thinking', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Opus 4.6-thinking', from: 'claude-opus-4-6-thinking', to: 'claude-opus-4-6-thinking', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Opus 4.7', from: 'claude-opus-4-7', to: 'claude-opus-4-7', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Opus 4.8', from: 'claude-opus-4-8', to: 'claude-opus-4-8', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' }
]

// Bedrock 预设映射（与后端 DefaultBedrockModelMapping 保持一致）
const bedrockPresetMappings = [
  { label: 'Fable 5.1', from: 'claude-fable-5-1', to: 'anthropic.claude-fable-5-1', color: 'bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400' },
  { label: 'Fable 5', from: 'claude-fable-5', to: 'anthropic.claude-fable-5', color: 'bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400' },
  { label: 'Opus 4.6', from: 'claude-opus-4-6', to: 'us.anthropic.claude-opus-4-6-v1', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Opus 4.7', from: 'claude-opus-4-7', to: 'us.anthropic.claude-opus-4-7-v1', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Opus 4.8', from: 'claude-opus-4-8', to: 'us.anthropic.claude-opus-4-8-v1', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Opus 5', from: 'claude-opus-5', to: 'us.anthropic.claude-opus-5-v1', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Sonnet 5', from: 'claude-sonnet-5', to: 'us.anthropic.claude-sonnet-5-v1', color: 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400' },
  { label: 'Sonnet 4.6', from: 'claude-sonnet-4-6', to: 'us.anthropic.claude-sonnet-4-6', color: 'bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { label: 'Opus 4.5', from: 'claude-opus-4-5-20251101', to: 'us.anthropic.claude-opus-4-5-20251101-v1:0', color: 'bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400' },
  { label: 'Sonnet 4.5', from: 'claude-sonnet-4-5-20250929', to: 'us.anthropic.claude-sonnet-4-5-20250929-v1:0', color: 'bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400' },
  { label: 'Haiku 4.5', from: 'claude-haiku-4-5-20251001', to: 'us.anthropic.claude-haiku-4-5-20251001-v1:0', color: 'bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400' },
]

// Antigravity 默认映射（从后端 API 获取，与 constants.go 保持一致）
// 使用 fetchAntigravityDefaultMappings() 异步获取
let _antigravityDefaultMappingsCache: { from: string; to: string }[] | null = null

export async function fetchAntigravityDefaultMappings(): Promise<{ from: string; to: string }[]> {
  if (_antigravityDefaultMappingsCache !== null) {
    return _antigravityDefaultMappingsCache
  }
  try {
    const { getAntigravityDefaultModelMapping } = await import('@/api/admin/accounts')
    const mapping = await getAntigravityDefaultModelMapping()
    _antigravityDefaultMappingsCache = filterSupportedModelMappings(
      'antigravity',
      Object.entries(mapping).map(([from, to]) => ({ from, to }))
    )
  } catch (e) {
    console.warn('[fetchAntigravityDefaultMappings] API failed, using empty fallback', e)
    _antigravityDefaultMappingsCache = []
  }
  return _antigravityDefaultMappingsCache
}

// =====================
// 常用错误码
// =====================

export const commonErrorCodes = [
  { value: 401, label: 'Unauthorized' },
  { value: 403, label: 'Forbidden' },
  { value: 429, label: 'Rate Limit' },
  { value: 500, label: 'Server Error' },
  { value: 502, label: 'Bad Gateway' },
  { value: 503, label: 'Unavailable' },
  { value: 529, label: 'Overloaded' }
]

// =====================
// 辅助函数
// =====================

// 按平台获取模型
export function getModelsByPlatform(platform: string): string[] {
  platform = platform.trim().toLowerCase()
  switch (platform) {
    case 'openai': return openaiModels
    case 'anthropic':
    case 'claude': return claudeModels
    case 'gemini': return geminiModels
    case 'antigravity': return antigravityModels
    case 'zhipu': return zhipuModels
    case 'qwen': return qwenModels
    case 'deepseek': return deepseekModels
    case 'mistral': return mistralModels
    case 'meta': return metaModels
    case 'xai':
    case 'grok': return xaiModels
    case 'cohere': return cohereModels
    case 'yi': return yiModels
    case 'moonshot':
    case 'kimi': return moonshotModels
    case 'opencode_go': return opencodeGoModels
    case 'composite': return allModelsList
    case 'bedrock': return claudeModels
    case 'doubao': return doubaoModels
    case 'minimax': return minimaxModels
    case 'baidu': return baiduModels
    case 'spark': return sparkModels
    case 'hunyuan': return hunyuanModels
    case 'perplexity': return perplexityModels
    default: return claudeModels
  }
}

export function normalizeModelId(model: string): string {
  const trimmed = model.trim()
  return trimmed.startsWith('models/') ? trimmed.slice('models/'.length) : trimmed
}

export function filterSupportedModelIds(platform: string | string[], models: string[]): string[] {
  const platforms = Array.isArray(platform) ? platform : [platform]
  const supported = new Set(platforms.flatMap(getModelsByPlatform).map(normalizeModelId))
  return models
    .map(normalizeModelId)
    .filter(model => supported.has(model))
}

export function filterSupportedModels<T extends { id: string }>(platform: string | string[], models: T[]): T[] {
  const platforms = Array.isArray(platform) ? platform : [platform]
  const supported = new Set(platforms.flatMap(getModelsByPlatform).map(normalizeModelId))
  return models.filter(model => supported.has(normalizeModelId(model.id)))
}

export function filterSupportedModelMappings<T extends { from: string; to: string }>(
  platform: string,
  mappings: T[]
): T[] {
  const normalizedPlatform = platform.trim().toLowerCase()
  const supported = new Set(getModelsByPlatform(normalizedPlatform).map(normalizeModelId))
  const antigravityMappingAliases = new Set(['gemini-3-pro-image', 'gemini-3-pro-image-preview'])
  const targetIsSupported = (target: string) => {
    if (supported.has(target)) return true
    // Bedrock uses provider-qualified IDs such as
    // us.anthropic.claude-opus-4-6-v1. They still refer to the supported
    // Claude model and should remain available as mapping targets.
    return normalizedPlatform === 'bedrock' && [...supported].some(model => target.includes(model))
  }
  return mappings.filter(({ from, to }) => {
    const source = normalizeModelId(from)
    const target = normalizeModelId(to)
    const isAntigravityImageAlias = normalizedPlatform === 'antigravity' && antigravityMappingAliases.has(source)
    return (source.endsWith('*') || supported.has(source) || isAntigravityImageAlias) && targetIsSupported(target)
  })
}

// 按平台获取预设映射
export function getPresetMappingsByPlatform(platform: string) {
  const normalizedPlatform = platform.trim().toLowerCase()
  let mappings
  if (normalizedPlatform === 'openai') mappings = openaiPresetMappings
  else if (normalizedPlatform === 'gemini') mappings = geminiPresetMappings
  else if (normalizedPlatform === 'grok' || normalizedPlatform === 'xai') mappings = grokPresetMappings
  else if (normalizedPlatform === 'antigravity') mappings = antigravityPresetMappings
  else if (normalizedPlatform === 'bedrock') mappings = bedrockPresetMappings
  else mappings = anthropicPresetMappings
  return filterSupportedModelMappings(normalizedPlatform, mappings)
}

// =====================
// 构建模型映射对象（用于 API）
// =====================

// isValidWildcardPattern 校验通配符格式：* 只能放在末尾
// 导出供表单组件使用实时校验
export function isValidWildcardPattern(pattern: string): boolean {
  const starIndex = pattern.indexOf('*')
  if (starIndex === -1) return true // 无通配符，有效
  // * 必须在末尾，且只能有一个
  return starIndex === pattern.length - 1 && pattern.lastIndexOf('*') === starIndex
}

export type ModelRestrictionMode = 'whitelist' | 'mapping' | 'combined'

export interface ModelMappingEntry {
  from: string
  to: string
}

export function splitModelMappingObject(
  modelMapping?: Record<string, unknown> | null
): { allowedModels: string[]; modelMappings: ModelMappingEntry[] } {
  const allowedModels: string[] = []
  const modelMappings: ModelMappingEntry[] = []

  if (!modelMapping || typeof modelMapping !== 'object') {
    return { allowedModels, modelMappings }
  }

  for (const [rawFrom, rawTo] of Object.entries(modelMapping)) {
    if (typeof rawTo !== 'string') continue
    const from = rawFrom.trim()
    const to = rawTo.trim()
    if (!from || !to) continue

    if (from === to) {
      allowedModels.push(from)
    } else {
      modelMappings.push({ from, to })
    }
  }

  return { allowedModels, modelMappings }
}

export function buildModelMappingObject(
  mode: ModelRestrictionMode,
  allowedModels: string[],
  modelMappings: ModelMappingEntry[]
): Record<string, string> | null {
  const mapping: Record<string, string> = {}

  if (mode === 'whitelist' || mode === 'combined') {
    for (const model of allowedModels) {
      const normalizedModel = model.trim()
      if (!normalizedModel) continue
      // Identity wildcard mappings preserve the concrete model ID upstream.
      if (isValidWildcardPattern(normalizedModel)) {
        mapping[normalizedModel] = normalizedModel
      }
    }
  }

  if (mode === 'mapping' || mode === 'combined') {
    for (const m of modelMappings) {
      const from = m.from.trim()
      const to = m.to.trim()
      if (!from || !to) continue
      // 校验通配符格式：* 只能放在末尾
      if (!isValidWildcardPattern(from)) {
        console.warn(`[buildModelMappingObject] Invalid wildcard pattern, skipped: ${from}`)
        continue
      }
      // to 不允许包含通配符
      if (to.includes('*')) {
        console.warn(`[buildModelMappingObject] Target model cannot contain a wildcard, skipped: ${from} -> ${to}`)
        continue
      }
      mapping[from] = to
    }
  }

  return Object.keys(mapping).length > 0 ? mapping : null
}
