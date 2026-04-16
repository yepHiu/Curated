export function extractPreviousPackageHistoryCommit(markdown: string): string | null

export function buildPackageHistoryChangeSummary(options: {
  historyMarkdown: string
  currentCommit: string
  resolveCommit: (commit: string) => Promise<string>
  loadGitLog: (previousCommit: string, currentCommit: string) => Promise<string>
}): Promise<string>
