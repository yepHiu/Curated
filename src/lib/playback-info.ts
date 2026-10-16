export interface PlaybackInfoMetric { number: string; unit: string; reserveUnit: boolean }
export interface PlaybackInfoRow { label: string; value: string; metrics?: PlaybackInfoMetric[] }
export interface PlaybackInfoGroup { title: string; rows: PlaybackInfoRow[] }
