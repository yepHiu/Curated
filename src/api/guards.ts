import type {
  ActorMergeAssociationSummaryDTO,
  ActorMergeAuditDTO,
  ActorMergeAuditListDTO,
  ActorMergePreviewDTO,
  ConnectedClientAccessKind,
  ConnectedClientDeviceType,
  ConnectedClientDTO,
  ConnectedClientsDTO,
  BackupManifestDTO,
  BackupRestorePreflightDTO,
  BackupVerificationDTO,
  HealthDTO,
  HomepageDailyRecommendationsDTO,
  HomepageRecommendationFeedbackEffectDTO,
  HomepageRecommendationItemDTO,
  HomepageRecommendationReasonDTO,
  LibraryHealthFindingDTO,
  LibraryHealthRepairDTO,
  LibraryHealthReportDTO,
  LibraryPathStorageStatusDTO,
  MovieDetailDTO,
  MovieListItemDTO,
  MoviesPageDTO,
  ImportMovieCodeCheckDTO,
  PersonalInsightsBreakdownDTO,
  PersonalInsightsBreakdownItemDTO,
  PersonalInsightsOverviewDTO,
  SavedViewDTO,
  SavedViewFiltersV1,
  SavedViewsDTO,
  RecommendationFeedbackDTO,
  RecommendationFeedbackListDTO,
} from "./types"

export class InvalidApiResponseError extends Error {
  readonly endpoint: string
  readonly value: unknown

  constructor(endpoint: string, value: unknown) {
    super(`Invalid API response for ${endpoint}`)
    this.name = "InvalidApiResponseError"
    this.endpoint = endpoint
    this.value = value
  }
}

type Guard<T> = (value: unknown) => value is T

export function assertApiResponse<T>(
  endpoint: string,
  value: unknown,
  guard: Guard<T>,
): T {
  if (!guard(value)) {
    throw new InvalidApiResponseError(endpoint, value)
  }
  return value
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function isString(value: unknown): value is string {
  return typeof value === "string"
}

function isBoolean(value: unknown): value is boolean {
  return typeof value === "boolean"
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value)
}

function isNonNegativeInteger(value: unknown): value is number {
  return isFiniteNumber(value) && Number.isInteger(value) && value >= 0
}

function isNonNegativeFiniteNumber(value: unknown): value is number {
  return isFiniteNumber(value) && value >= 0
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every(isString)
}

function isOptionalString(value: unknown): boolean {
  return value === undefined || isString(value)
}

function isOptionalStringArray(value: unknown): boolean {
  return value === undefined || isStringArray(value)
}

function isOptionalNullableNumber(value: unknown): boolean {
  return value === undefined || value === null || isFiniteNumber(value)
}

function isOptionalFiniteNumber(value: unknown): boolean {
  return value === undefined || isFiniteNumber(value)
}

function isOptionalStringRecord(value: unknown): boolean {
  if (value === undefined) {
    return true
  }
  if (!isRecord(value)) {
    return false
  }
  return Object.values(value).every(isString)
}

export function isHealthDTO(value: unknown): value is HealthDTO {
  return (
    isRecord(value) &&
    isString(value.name) &&
    isString(value.version) &&
    isString(value.transport) &&
    isString(value.databasePath) &&
    isOptionalString(value.channel) &&
    isOptionalString(value.installerVersion)
  )
}

function isActorMergeAssociationSummaryDTO(
  value: unknown,
): value is ActorMergeAssociationSummaryDTO {
  return (
    isRecord(value) &&
    isNonNegativeInteger(value.sourceCount) &&
    isNonNegativeInteger(value.targetCount) &&
    isNonNegativeInteger(value.duplicateCount) &&
    isNonNegativeInteger(value.resultCount)
  )
}

function isActorMergeValuesSummaryDTO(value: unknown): boolean {
  return (
    isRecord(value) &&
    isStringArray(value.source) &&
    isStringArray(value.target) &&
    isStringArray(value.result)
  )
}

function isActorMergeProfileSelection(value: unknown): value is "source" | "target" {
  return value === "source" || value === "target"
}

function isActorMergeProfileSelectionRecord(value: unknown): boolean {
  return isRecord(value) && Object.values(value).every(isActorMergeProfileSelection)
}

export function isActorMergePreviewDTO(value: unknown): value is ActorMergePreviewDTO {
  return (
    isRecord(value) &&
    isString(value.previewToken) &&
    isRecord(value.source) &&
    isNonNegativeInteger(value.source.id) &&
    isString(value.source.name) &&
    isStringArray(value.source.aliases) &&
    isRecord(value.target) &&
    isNonNegativeInteger(value.target.id) &&
    isString(value.target.name) &&
    isStringArray(value.target.aliases) &&
    isActorMergeAssociationSummaryDTO(value.movies) &&
    isActorMergeValuesSummaryDTO(value.userTags) &&
    isActorMergeValuesSummaryDTO(value.externalLinks) &&
    isActorMergeAssociationSummaryDTO(value.recommendationFeedback) &&
    isNonNegativeInteger(value.curatedFramesAffected) &&
    isStringArray(value.aliasesToMove) &&
    Array.isArray(value.profileFields) &&
    value.profileFields.every(
      (field) =>
        isRecord(field) &&
        isString(field.field) &&
        isString(field.sourceValue) &&
        isString(field.targetValue) &&
        isActorMergeProfileSelection(field.defaultSelection) &&
        isBoolean(field.conflict),
    ) &&
    isBoolean(value.canApply) &&
    Array.isArray(value.blockingReasons) &&
    value.blockingReasons.every(
      (reason) => isRecord(reason) && isString(reason.code) && isString(reason.message),
    ) &&
    isStringArray(value.requiredDecisions)
  )
}

export function isActorMergeAuditDTO(value: unknown): value is ActorMergeAuditDTO {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isNonNegativeInteger(value.sourceActorId) &&
    isNonNegativeInteger(value.targetActorId) &&
    isString(value.sourceName) &&
    isString(value.targetName) &&
    isString(value.previewToken) &&
    isString(value.appliedAt) &&
    isRecord(value.summary) &&
    isActorMergeAssociationSummaryDTO(value.summary.movies) &&
    isStringArray(value.summary.userTags) &&
    isStringArray(value.summary.externalLinks) &&
    isStringArray(value.summary.aliases) &&
    isActorMergeAssociationSummaryDTO(value.summary.recommendationFeedback) &&
    isNonNegativeInteger(value.summary.curatedFramesAffected) &&
    isActorMergeProfileSelectionRecord(value.summary.profileDecisions)
  )
}

export function isActorMergeAuditListDTO(value: unknown): value is ActorMergeAuditListDTO {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isActorMergeAuditDTO) &&
    isNonNegativeInteger(value.total) &&
    isNonNegativeInteger(value.limit) &&
    isNonNegativeInteger(value.offset)
  )
}

function isNumberRecord(value: unknown): boolean {
  return isRecord(value) && Object.values(value).every(isFiniteNumber)
}

function isLibraryPathStorageStatusDTO(value: unknown): value is LibraryPathStorageStatusDTO {
  return (
    isRecord(value) &&
    isString(value.libraryPathId) &&
    isString(value.path) &&
    isString(value.title) &&
    ["online", "offline", "volume_mismatch", "path_missing", "permission_denied", "unknown"].includes(String(value.status)) &&
    isString(value.message) &&
    isString(value.checkedAt) &&
    isBoolean(value.canRescan) &&
    isBoolean(value.canImport)
  )
}

function isLibraryHealthFindingDTO(value: unknown): value is LibraryHealthFindingDTO {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.category) &&
    (value.severity === "critical" || value.severity === "warning" || value.severity === "info") &&
    isString(value.entityType) &&
    isOptionalString(value.entityId) &&
    isString(value.label) &&
    isOptionalString(value.path) &&
    isString(value.message) &&
    (value.details === undefined || isRecord(value.details)) &&
    isOptionalStringArray(value.repairActions)
  )
}

export function isLibraryHealthReportDTO(value: unknown): value is LibraryHealthReportDTO {
  return (
    isRecord(value) &&
    isString(value.scannedAt) &&
    (value.status === "healthy" || value.status === "attention" || value.status === "critical") &&
    isRecord(value.database) &&
    isBoolean(value.database.quickCheckOk) &&
    isStringArray(value.database.quickCheckMessages) &&
    isBoolean(value.database.foreignKeyOk) &&
    isNonNegativeInteger(value.database.foreignKeyCount) &&
    Array.isArray(value.storageStatuses) &&
    value.storageStatuses.every(isLibraryPathStorageStatusDTO) &&
    isRecord(value.summary) &&
    isNonNegativeInteger(value.summary.totalFindings) &&
    isNonNegativeInteger(value.summary.criticalFindings) &&
    isNonNegativeInteger(value.summary.warningFindings) &&
    isNonNegativeInteger(value.summary.infoFindings) &&
    isNonNegativeInteger(value.summary.skippedOfflineFiles) &&
    isNumberRecord(value.summary.categoryCounts) &&
    Array.isArray(value.findings) &&
    value.findings.every(isLibraryHealthFindingDTO) &&
    isBoolean(value.truncated)
  )
}

export function isLibraryHealthRepairDTO(value: unknown): value is LibraryHealthRepairDTO {
  return (
    isRecord(value) &&
    isString(value.repairId) &&
    isString(value.taskId) &&
    value.action === "rescrape_metadata" &&
    isStringArray(value.categories) &&
    ["pending", "running", "completed", "partial_failed", "failed", "cancelled"].includes(String(value.status)) &&
    isNonNegativeInteger(value.totalItems) &&
    isNonNegativeInteger(value.completedItems) &&
    isNonNegativeInteger(value.succeededItems) &&
    isNonNegativeInteger(value.failedItems) &&
    isString(value.createdAt) &&
    isOptionalString(value.startedAt) &&
    isOptionalString(value.finishedAt) &&
    Array.isArray(value.items) &&
    value.items.every((item) =>
      isRecord(item) &&
      isNonNegativeInteger(item.ordinal) &&
      isString(item.findingId) &&
      isString(item.category) &&
      isString(item.movieId) &&
      isString(item.label) &&
      ["pending", "queued", "succeeded", "failed", "cancelled"].includes(String(item.status)) &&
      isOptionalString(item.childTaskId) &&
      isOptionalString(item.errorCode) &&
      isOptionalString(item.errorMessage) &&
      isOptionalString(item.startedAt) &&
      isOptionalString(item.finishedAt),
    )
  )
}

function isConnectedClientDeviceType(value: unknown): value is ConnectedClientDeviceType {
  return (
    value === "desktop" ||
    value === "laptop" ||
    value === "mobile" ||
    value === "tablet" ||
    value === "tool" ||
    value === "unknown"
  )
}

function isConnectedClientAccessKind(value: unknown): value is ConnectedClientAccessKind {
  return value === "local" || value === "remote"
}

export function isConnectedClientDTO(value: unknown): value is ConnectedClientDTO {
  return (
    isRecord(value) &&
    isString(value.key) &&
    isString(value.ip) &&
    isOptionalFiniteNumber(value.port) &&
    isOptionalString(value.hostname) &&
    isOptionalString(value.userAgent) &&
    isString(value.browser) &&
    isOptionalString(value.browserVersion) &&
    isString(value.os) &&
    isOptionalString(value.osVersion) &&
    isConnectedClientDeviceType(value.deviceType) &&
    isConnectedClientAccessKind(value.accessKind) &&
    isBoolean(value.isLocalMachine) &&
    isString(value.firstSeen) &&
    isString(value.lastSeen) &&
    isFiniteNumber(value.requestCount)
  )
}

export function isConnectedClientsDTO(value: unknown): value is ConnectedClientsDTO {
  return (
    isRecord(value) &&
    Array.isArray(value.clients) &&
    value.clients.every(isConnectedClientDTO) &&
    isFiniteNumber(value.total) &&
    isFiniteNumber(value.localCount) &&
    isFiniteNumber(value.remoteCount) &&
    isString(value.sampledAt)
  )
}

export function isBackupManifestDTO(value: unknown): value is BackupManifestDTO {
  return (
    isRecord(value) &&
    isString(value.format) &&
    isFiniteNumber(value.formatVersion) &&
    isString(value.createdAt) &&
    isString(value.appVersion) &&
    isString(value.appChannel) &&
    isRecord(value.scope) &&
    isBoolean(value.scope.databaseIncluded) &&
    isBoolean(value.scope.libraryConfigIncluded) &&
    isBoolean(value.scope.userAssetsIncluded) &&
    isBoolean(value.scope.mediaFilesIncluded) &&
    isStringArray(value.schemaMigrations) &&
    Array.isArray(value.files) &&
    value.files.every((file) =>
      isRecord(file) &&
      isString(file.kind) &&
      isString(file.path) &&
      isFiniteNumber(file.sizeBytes) &&
      isString(file.sha256),
    )
  )
}

export function isBackupVerificationDTO(value: unknown): value is BackupVerificationDTO {
  return (
    isRecord(value) &&
    isBoolean(value.valid) &&
    isString(value.checkedAt) &&
    (value.manifest === undefined || isBackupManifestDTO(value.manifest)) &&
    isRecord(value.databaseIntegrity) &&
    isString(value.databaseIntegrity.quickCheck) &&
    isFiniteNumber(value.databaseIntegrity.foreignKeyViolations) &&
    isStringArray(value.errors) &&
    isStringArray(value.warnings)
  )
}

export function isBackupRestorePreflightDTO(value: unknown): value is BackupRestorePreflightDTO {
  return (
    isRecord(value) &&
    isBoolean(value.canRestore) &&
    isString(value.checkedAt) &&
    isBackupVerificationDTO(value.verification) &&
    isString(value.targetDatabase) &&
    isBoolean(value.targetDatabaseExists) &&
    isOptionalString(value.targetConfig) &&
    isBoolean(value.targetConfigExists) &&
    isFiniteNumber(value.requiredBytes) &&
    isFiniteNumber(value.availableBytes) &&
    isBoolean(value.availableBytesKnown) &&
    isStringArray(value.unsupportedMigrations) &&
    isStringArray(value.errors) &&
    isStringArray(value.warnings)
  )
}

export function isMovieListItemDTO(value: unknown): value is MovieListItemDTO {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.title) &&
    isString(value.code) &&
    isString(value.studio) &&
    isStringArray(value.actors) &&
    isStringArray(value.tags) &&
    isOptionalStringArray(value.userTags) &&
    isFiniteNumber(value.runtimeMinutes) &&
    isFiniteNumber(value.rating) &&
    isOptionalNullableNumber(value.userRating) &&
    isBoolean(value.isFavorite) &&
    isString(value.addedAt) &&
    isString(value.location) &&
    isString(value.resolution) &&
    isFiniteNumber(value.year) &&
    isOptionalString(value.releaseDate) &&
    isOptionalString(value.coverUrl) &&
    isOptionalString(value.thumbUrl) &&
    isOptionalString(value.trashedAt)
  )
}

export function isMovieDetailDTO(value: unknown): value is MovieDetailDTO {
  if (!isMovieListItemDTO(value) || !isRecord(value)) {
    return false
  }
  return (
    isString(value.summary) &&
    isFiniteNumber(value.metadataRating) &&
    isOptionalNullableNumber(value.userRating) &&
    isOptionalStringArray(value.previewImages) &&
    isOptionalString(value.previewVideoUrl) &&
    isOptionalStringRecord(value.actorAvatarUrls) &&
    isOptionalString(value.metadataProvider)
  )
}

export function isMoviesPageDTO(value: unknown): value is MoviesPageDTO {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(isMovieListItemDTO) &&
    isFiniteNumber(value.total) &&
    isFiniteNumber(value.limit) &&
    isFiniteNumber(value.offset)
  )
}

function isImportMovieCodeMatchDTO(value: unknown): boolean {
  return (
    isRecord(value) &&
    isString(value.movieId) &&
    isString(value.code) &&
    isString(value.title) &&
    (value.matchKind === "exact" || value.matchKind === "similar")
  )
}

export function isImportMovieCodeCheckDTO(value: unknown): value is ImportMovieCodeCheckDTO {
  return (
    isRecord(value) &&
    Array.isArray(value.items) &&
    value.items.every(
      (item) =>
        isRecord(item) &&
        isString(item.name) &&
        isOptionalString(item.extractedCode) &&
        Array.isArray(item.matches) &&
        item.matches.every(isImportMovieCodeMatchDTO),
    ) &&
    isNonNegativeInteger(value.matchedCount)
  )
}

function isSavedViewFiltersV1(value: unknown): value is SavedViewFiltersV1 {
  if (!isRecord(value) || value.schemaVersion !== 1) {
    return false
  }
  const mode = value.mode
  const tab = value.tab
  const playState = value.playState
  return (
    (mode === undefined || ["library", "favorites", "recent", "tags", "trash"].includes(String(mode))) &&
    (tab === undefined || ["all", "new", "top-rated"].includes(String(tab))) &&
    (playState === undefined || ["all", "unwatched", "in-progress", "completed"].includes(String(playState))) &&
    isOptionalString(value.q) &&
    isOptionalString(value.tag) &&
    isOptionalString(value.actor) &&
    isOptionalString(value.studio) &&
    isOptionalFiniteNumber(value.userRating) &&
    (value.unrated === undefined || value.unrated === true || value.unrated === false) &&
    isOptionalString(value.resolution) &&
    (value.addedWithinDays === undefined || isNonNegativeInteger(value.addedWithinDays)) &&
    (value.year === undefined || value.year === "unknown" || /^\d{4}$/.test(String(value.year))) &&
    (value.runtime === undefined || ["short", "standard", "long"].includes(String(value.runtime))) &&
    (value.catalog === undefined || ["unscraped", "no-cover"].includes(String(value.catalog)))
  )
}

export function isSavedViewDTO(value: unknown): value is SavedViewDTO {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.name) &&
    isSavedViewFiltersV1(value.filters) &&
    isNonNegativeInteger(value.sortOrder) &&
    isString(value.createdAt) &&
    isString(value.updatedAt)
  )
}

export function isSavedViewsDTO(value: unknown): value is SavedViewsDTO {
  return isRecord(value) && Array.isArray(value.items) && value.items.every(isSavedViewDTO)
}

const recommendationReasonCodes = new Set([
  "high_user_rating",
  "favorite",
  "recently_added",
  "well_rated",
  "rediscovery",
  "catalog_discovery",
  "shared_actor",
  "shared_studio",
  "shared_tag",
])

function isHomepageRecommendationReasonDTO(value: unknown): value is HomepageRecommendationReasonDTO {
  return (
    isRecord(value) &&
    isString(value.code) &&
    recommendationReasonCodes.has(value.code) &&
    (value.entityType === undefined || ["actor", "studio", "tag"].includes(String(value.entityType))) &&
    isOptionalString(value.entityValue)
  )
}

function isHomepageRecommendationFeedbackEffectDTO(value: unknown): value is HomepageRecommendationFeedbackEffectDTO {
  return (
    isRecord(value) &&
    isString(value.feedbackId) &&
    ["actor", "studio", "tag"].includes(String(value.targetType)) &&
    isString(value.targetValue) &&
    value.effect === "weight_reduced"
  )
}

function isHomepageRecommendationItemDTO(value: unknown): value is HomepageRecommendationItemDTO {
  return (
    isRecord(value) &&
    isString(value.movieId) &&
    Array.isArray(value.reasons) &&
    value.reasons.length > 0 &&
    value.reasons.every(isHomepageRecommendationReasonDTO) &&
    Array.isArray(value.feedbackEffects) &&
    value.feedbackEffects.every(isHomepageRecommendationFeedbackEffectDTO)
  )
}

export function isHomepageDailyRecommendationsDTO(value: unknown): value is HomepageDailyRecommendationsDTO {
  if (!isRecord(value)) return false
  const recommendationMovieIds = value.recommendationMovieIds
  const recommendations = value.recommendations
  return (
    isString(value.dateUtc) &&
    isString(value.generatedAt) &&
    isOptionalString(value.generationVersion) &&
    isStringArray(value.heroMovieIds) &&
    isStringArray(recommendationMovieIds) &&
    Array.isArray(recommendations) &&
    recommendations.every(isHomepageRecommendationItemDTO) &&
    recommendations.length === recommendationMovieIds.length &&
    recommendations.every((item, index) => item.movieId === recommendationMovieIds[index])
  )
}

export function isRecommendationFeedbackDTO(value: unknown): value is RecommendationFeedbackDTO {
  return (
    isRecord(value) &&
    isString(value.id) &&
    ["not_interested", "snooze", "less"].includes(String(value.action)) &&
    ["movie", "actor", "studio", "tag"].includes(String(value.targetType)) &&
    isString(value.targetValue) &&
    isString(value.sourceMovieId) &&
    isOptionalString(value.expiresAt) &&
    isString(value.createdAt) &&
    isString(value.updatedAt)
  )
}

export function isRecommendationFeedbackListDTO(value: unknown): value is RecommendationFeedbackListDTO {
  return isRecord(value) && Array.isArray(value.items) && value.items.every(isRecommendationFeedbackDTO)
}

function isPersonalInsightsRange(value: unknown): boolean {
  return ["30d", "90d", "365d", "all"].includes(String(value))
}

function isPersonalInsightsDimension(value: unknown): boolean {
  return ["actor", "studio", "tag"].includes(String(value))
}

function isNullableString(value: unknown): boolean {
  return value === null || isString(value)
}

function isPersonalInsightsBreakdownItemDTO(value: unknown): value is PersonalInsightsBreakdownItemDTO {
  return (
    isRecord(value) &&
    isString(value.name) &&
    value.name.trim().length > 0 &&
    isNonNegativeFiniteNumber(value.watchedSeconds) &&
    isNonNegativeInteger(value.movieCount) &&
    isFiniteNumber(value.shareOfTotal) &&
    value.shareOfTotal >= 0 &&
    value.shareOfTotal <= 1
  )
}

export function isPersonalInsightsOverviewDTO(value: unknown): value is PersonalInsightsOverviewDTO {
  if (!isRecord(value)) return false
  const completionRateValid = value.startedMovies === 0
    ? value.completionRate === null
    : isFiniteNumber(value.completionRate) &&
      value.completionRate >= 0 &&
      value.completionRate <= 1 &&
      Math.abs(value.completionRate - Number(value.completedMovies) / Number(value.startedMovies)) < 1e-9
  const averageRatingValid = value.ratedMovies === 0
    ? value.averageUserRating === null
    : isFiniteNumber(value.averageUserRating)
  return (
    isPersonalInsightsRange(value.range) &&
    isString(value.from) &&
    isString(value.to) &&
    isString(value.timezone) &&
    isString(value.generatedAt) &&
    isNullableString(value.dataSince) &&
    isNonNegativeFiniteNumber(value.watchedSeconds) &&
    isNonNegativeInteger(value.startedMovies) &&
    isNonNegativeInteger(value.completedMovies) &&
    value.completedMovies <= value.startedMovies &&
    completionRateValid &&
    isFiniteNumber(value.completionThreshold) &&
    value.completionThreshold > 0 &&
    value.completionThreshold <= 1 &&
    isNonNegativeInteger(value.ratedMovies) &&
    value.ratedMovies <= value.startedMovies &&
    averageRatingValid
  )
}

export function isPersonalInsightsBreakdownDTO(value: unknown): value is PersonalInsightsBreakdownDTO {
  return (
    isRecord(value) &&
    isPersonalInsightsRange(value.range) &&
    isPersonalInsightsDimension(value.dimension) &&
    isString(value.from) &&
    isString(value.to) &&
    isString(value.timezone) &&
    isString(value.generatedAt) &&
    isNullableString(value.dataSince) &&
    isNonNegativeFiniteNumber(value.totalWatchedSeconds) &&
    value.attribution === "full-per-entity" &&
    isNonNegativeInteger(value.limit) &&
    value.limit >= 1 &&
    Array.isArray(value.items) &&
    value.items.length <= value.limit &&
    value.items.every(isPersonalInsightsBreakdownItemDTO)
  )
}
