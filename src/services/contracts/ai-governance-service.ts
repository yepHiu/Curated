export interface AIGovernanceSettings {
  enabled: boolean
  readOnly: boolean
  privacy: "auto" | "minimal"
  stepLimit: number
  writePerMinute: number
  retentionDays: number
}

export const defaultAIGovernance = (): AIGovernanceSettings => ({
  enabled: false, readOnly: false, privacy: "auto", stepLimit: 15, writePerMinute: 10, retentionDays: 30,
})

export interface AIRun {
  id: string
  startedAt: string
  channel: string
  action: string
  sessionId: string
  provider: string
  model: string
  promptVersion: string
  status: string
  errorCode: string
  durationMs: number
  firstTextMs: number | null
  modelCalls: number
  usageCalls: number
  toolCalls: number
  promptTokens: number
  completionTokens: number
  totalTokens: number
}

export interface AISummary {
  runs: number
  failed: number
  partial: number
  cancelled: number
  modelCalls: number
  usageCalls: number
  toolCalls: number
  promptTokens: number
  completionTokens: number
  totalTokens: number
  avgDurationMs: number | null
  avgFirstTextMs: number | null
}

export interface AIAuditEntry {
  id: string
  createdAt: string
  channel: string
  sessionId: string
  tool: string
  permission: string
  result: string
  errorCode: string
  durationMs: number
}
export interface AIReportQuery { days?: number; channel?: string; status?: string; offset?: number; limit?: number }
export interface AIPage<T> { items: T[]; total: number; limit: number; offset: number }
export interface AIReport extends AIPage<AIRun> { summary: AISummary }
export interface AICleanup { runs: number; audit: number; receipts: number }
export interface AIGovernanceService {
  getSettings(): Promise<AIGovernanceSettings>
  saveSettings(value: AIGovernanceSettings): Promise<AIGovernanceSettings>
  getUsage(query: AIReportQuery): Promise<AIReport>
  getAudit(query: AIReportQuery): Promise<AIPage<AIAuditEntry>>
  cleanup(): Promise<AICleanup>
}
