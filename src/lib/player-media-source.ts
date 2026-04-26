export type ManagedPlaybackMode = "direct" | "hls"

type ResettableVideoElement = Pick<HTMLVideoElement, "pause" | "removeAttribute" | "load">

type PlaybackStartReadinessInput = {
  readyState: number
  haveCurrentData: number
  playbackMode: ManagedPlaybackMode
  resumeRequested: boolean
}

export function shouldResetVideoElementBeforeModeAttach(
  previousMode: ManagedPlaybackMode | undefined,
  nextMode: ManagedPlaybackMode,
): boolean {
  return nextMode === "hls" && previousMode !== "hls"
}

export function resetVideoElementPlaybackPipeline(video: ResettableVideoElement): void {
  video.pause()
  video.removeAttribute("src")
  video.load()
}

export function shouldDeferPlaybackStartUntilCurrentData({
  readyState,
  haveCurrentData,
  playbackMode,
  resumeRequested,
}: PlaybackStartReadinessInput): boolean {
  if (readyState >= haveCurrentData) {
    return false
  }
  return !(playbackMode === "hls" && resumeRequested)
}
