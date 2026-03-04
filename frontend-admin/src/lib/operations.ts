export const OPERATION_OPTIONS = [
  { value: 'chat.completions', label: 'chat.completions' },
  { value: 'completions', label: 'completions' },
  { value: 'embeddings', label: 'embeddings' },
  { value: 'images.generations', label: 'images.generations' },
  { value: 'videos.generations', label: 'videos.generations' },
  { value: 'audio.transcriptions', label: 'audio.transcriptions' },
  { value: 'audio.translations', label: 'audio.translations' },
  { value: 'audio.speech', label: 'audio.speech' },
] as const

export type OperationOption = (typeof OPERATION_OPTIONS)[number]

