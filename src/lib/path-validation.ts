/**
 * Client-side check aligned with backend: library roots must be absolute.
 * Windows: `C:\...`, `D:/...`, UNC `\\server\share\...`
 * Unix: `/...`
 */
export function isAbsoluteLibraryPath(path: string): boolean {
  const s = path.trim()
  if (!s) return false
  if (s.startsWith("\\\\")) return true
  if (/^[A-Za-z]:[\\/]/.test(s)) return true
  if (s.startsWith("/")) return true
  return false
}
