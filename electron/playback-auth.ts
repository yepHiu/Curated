/** Server 以 423 表示 PIN 锁定；兼容认证网关使用的 401/403。 */
export function isPlaybackAuthStatus(status: number): boolean {
  return status === 401 || status === 403 || status === 423
}
