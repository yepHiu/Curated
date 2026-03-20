export const formatRuntime = (minutes: number) => {
  const hours = Math.floor(minutes / 60)
  const remainder = minutes % 60

  return `${hours}h ${remainder}m`
}

export const formatAddedDate = (value: string) =>
  new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value))
