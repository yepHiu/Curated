import { useServerLocalAccess } from "@/composables/use-server-local-access"

/** Directory configuration belongs to Server. Mock data stays editable. */
export function useLibraryPathAccess() {
  const { isServerLocal } = useServerLocalAccess()
  return { canManagePaths: isServerLocal }
}
