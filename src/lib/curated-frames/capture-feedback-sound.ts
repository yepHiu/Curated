let audioContext: AudioContext | null = null

function getAudioContext(): AudioContext | null {
  if (audioContext) return audioContext
  if (typeof window === "undefined") return null
  const AudioContextConstructor =
    window.AudioContext ??
    (window as Window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
  if (!AudioContextConstructor) return null
  audioContext = new AudioContextConstructor()
  return audioContext
}

/**
 * Plays a deliberately quiet, short shutter cue. Failure to create/resume audio
 * must never block the actual frame capture.
 */
export async function playCuratedCaptureTriggerCue(enabled = true): Promise<void> {
  if (!enabled) return
  try {
    const context = getAudioContext()
    if (!context) return
    if (context.state === "suspended") {
      await context.resume()
    }

    const now = context.currentTime
    const noiseBuffer = context.createBuffer(1, Math.ceil(context.sampleRate * 0.09), context.sampleRate)
    const noiseData = noiseBuffer.getChannelData(0)
    for (let index = 0; index < noiseData.length; index += 1) {
      const fade = Math.pow(1 - index / noiseData.length, 1.6)
      noiseData[index] = (Math.random() * 2 - 1) * fade
    }

    const noise = context.createBufferSource()
    const noiseFilter = context.createBiquadFilter()
    const noiseGain = context.createGain()
    noise.buffer = noiseBuffer
    noiseFilter.type = "lowpass"
    noiseFilter.frequency.setValueAtTime(1700, now)
    noiseFilter.Q.setValueAtTime(0.25, now)
    noiseGain.gain.setValueAtTime(0.0001, now)
    noiseGain.gain.exponentialRampToValueAtTime(0.026, now + 0.006)
    noiseGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.09)
    noise.connect(noiseFilter)
    noiseFilter.connect(noiseGain)
    noiseGain.connect(context.destination)
    noise.start(now)
    noise.stop(now + 0.095)

    const oscillator = context.createOscillator()
    const oscillatorGain = context.createGain()
    oscillator.type = "sine"
    oscillator.frequency.setValueAtTime(430, now)
    oscillator.frequency.exponentialRampToValueAtTime(300, now + 0.09)
    oscillatorGain.gain.setValueAtTime(0.0001, now)
    oscillatorGain.gain.exponentialRampToValueAtTime(0.032, now + 0.008)
    oscillatorGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.09)
    oscillator.connect(oscillatorGain)
    oscillatorGain.connect(context.destination)
    oscillator.start(now)
    oscillator.stop(now + 0.095)
  } catch {
    // Browser autoplay/audio support is optional; visual/a11y feedback remains.
  }
}

export function disposeCuratedCaptureFeedbackAudio() {
  const context = audioContext
  audioContext = null
  if (!context) return
  void context.close().catch(() => {})
}
