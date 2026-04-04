import type { HardwareEncoderPreference } from "@/api/types"

const HARDWARE_ENCODER_OPTIONS: readonly HardwareEncoderPreference[] = [
  "auto",
  "amf",
  "qsv",
  "nvenc",
  "videotoolbox",
  "software",
]

/** 与后端 `config.NormalizeHardwareEncoderPreference` 对齐，避免非法值搞乱 Select / 比较逻辑 */
export function normalizeHardwareEncoderPreference(
  raw: string | undefined | null,
): HardwareEncoderPreference {
  const v = (raw ?? "auto").trim().toLowerCase()
  return (HARDWARE_ENCODER_OPTIONS as readonly string[]).includes(v)
    ? (v as HardwareEncoderPreference)
    : "auto"
}
