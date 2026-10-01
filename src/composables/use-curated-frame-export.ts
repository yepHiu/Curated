import type { PostCuratedFramesExportBody } from "@/api/types"
import type { CuratedFrameDialogItem } from "@/lib/curated-frames/dialog-navigation"
import { curatedFrameImageUrl } from "@/lib/curated-frame-image-url"
import { triggerDownloadBlob } from "@/lib/curated-frames/export-file"
import { buildWatermarkedCuratedFramesExport } from "@/lib/curated-frames/watermarked-export"
import { useLibraryService } from "@/services/library-service"

export type CuratedExportFormat = NonNullable<PostCuratedFramesExportBody["format"]>
export type CuratedExportMode = "raw" | "watermarked"

/** Shared by library batch/context actions and the frame detail dialog. */
export function useCuratedFrameExport() {
  const libraryService = useLibraryService()

  async function exportFrames(
    items: readonly CuratedFrameDialogItem[],
    ids: string[],
    actorName: string | undefined,
    format: CuratedExportFormat,
    mode: CuratedExportMode,
  ) {
    if (mode === "watermarked") {
      const sources = ids.map((id) => {
        const item = items.find((candidate) => candidate.row.id === id)
        if (!item) throw new Error(`curated frame ${id} is not loaded`)
        return { row: item.row, imageUrl: item.row.imageBlob ? item.url : curatedFrameImageUrl(id) }
      })
      const result = await buildWatermarkedCuratedFramesExport(sources, format)
      triggerDownloadBlob(result.blob, result.filename)
    } else {
      const body: PostCuratedFramesExportBody = { ids, format }
      if (actorName) body.actorName = actorName
      const result = await libraryService.exportCuratedFrames(body)
      triggerDownloadBlob(result.blob, result.filename)
    }
  }

  return { exportFrames }
}
