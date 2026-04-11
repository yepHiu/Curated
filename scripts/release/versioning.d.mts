export interface ReleaseVersionState {
  schema: 1
  current: {
    major: number
    minor: number
    patch: number
  }
}

export interface ReleaseVersionResult {
  state: ReleaseVersionState
  version: string
}

export function formatVersion(version: ReleaseVersionState["current"]): string
export function readVersionState(filePath: string): Promise<ReleaseVersionState>
export function allocateNextPatchInFile(filePath: string): Promise<ReleaseVersionResult>
export function setVersionBaseInFile(
  filePath: string,
  major: number,
  minor: number,
): Promise<ReleaseVersionResult>
