import { describe, expect, it } from "vitest"
import * as navigationIntent from "@/lib/navigation-intent"
import {
  buildDetailRouteFromActor,
  buildDetailRouteFromBrowse,
  buildPlayerRouteFromBrowseIntent,
  getNavigationBackTarget,
  resolveNavigationBackLink,
} from "@/lib/navigation-intent"

describe("navigation intent helpers", () => {
  it("returns wishlist details to their list filters and rejects external return paths", () => {
    expect(resolveNavigationBackLink({ name: 'wishlist-detail', query: { back: '/wishlist?status=all&q=TEST' } })).toEqual({
      to: '/wishlist?status=all&q=TEST', labelKey: 'wishlist.back',
    })
    for (const back of ['https://example.com', '//example.com', '/library', '/wishlist/other']) {
      expect(resolveNavigationBackLink({ name: 'wishlist-detail', query: { back } })).toEqual({
        to: { name: 'wishlist' }, labelKey: 'wishlist.back',
      })
    }
  })

  it("builds detail routes with browse context separated from return intent", () => {
    expect(
      buildDetailRouteFromBrowse(
        "movie-1",
        {
          q: "Mina",
          selected: "old-id",
          tab: "new",
          autoplay: "1",
          t: "123",
        },
        "favorites",
      ),
    ).toEqual({
      name: "detail",
      params: { id: "movie-1" },
      query: {
        browse: "favorites",
        q: "Mina",
        selected: "movie-1",
        tab: "new",
      },
    })
  })

  it("builds detail-origin filtered browse routes with a detail return intent", () => {
    const build = (
      navigationIntent as unknown as {
        buildFilteredBrowseRouteFromDetail?: (input: {
          movieId: string
          currentQuery: Record<string, string>
          sourceMode: "library" | "favorites" | "recent" | "tags" | "trash"
          kind: "tag" | "actor" | "studio"
          value: string
        }) => unknown
      }
    ).buildFilteredBrowseRouteFromDetail

    expect(build).toBeTypeOf("function")
    if (!build) return

    expect(
      build({
        movieId: "movie-1",
        currentQuery: {
          browse: "favorites",
          q: "Mina",
          selected: "movie-1",
          tab: "top-rated",
        },
        sourceMode: "favorites",
        kind: "actor",
        value: "Actor A",
      }),
    ).toEqual({
      name: "favorites",
      query: {
        actor: "Actor A",
        back: "detail",
        browse: "favorites",
        selected: "movie-1",
      },
    })
  })

  it("builds browse-launched player routes that return directly to browse", () => {
    expect(
      buildPlayerRouteFromBrowseIntent(
        "movie-1",
        {
          q: "Mina",
          selected: "old-id",
          tab: "new",
        },
        "favorites",
        "browse",
      ),
    ).toEqual({
      name: "player",
      params: { id: "movie-1" },
      query: {
        autoplay: "1",
        back: "browse",
        browse: "favorites",
        q: "Mina",
        selected: "movie-1",
        tab: "new",
      },
    })
  })

  it("builds detail-launched player routes that return to detail", () => {
    expect(
      buildPlayerRouteFromBrowseIntent(
        "movie-1",
        {
          browse: "favorites",
          q: "Mina",
          selected: "movie-1",
        },
        "favorites",
        "detail",
      ),
    ).toEqual({
      name: "player",
      params: { id: "movie-1" },
      query: {
        autoplay: "1",
        back: "detail",
        browse: "favorites",
        q: "Mina",
        selected: "movie-1",
      },
    })
  })

  it("preserves the new actor page across detail and player round trips", () => {
    const actorDetailRoute = buildDetailRouteFromActor("movie-1", "Mina Kaze") as {
      query: Record<string, string>
    }
    const playerRoute = buildPlayerRouteFromBrowseIntent(
      "movie-1",
      actorDetailRoute.query,
      "library",
      "detail",
    ) as { query: Record<string, string> }

    expect(playerRoute.query).toEqual({
      actor: "Mina Kaze",
      autoplay: "1",
      back: "detail",
      browse: "library",
      detailBack: "actor",
      selected: "movie-1",
    })

    const returnedDetail = resolveNavigationBackLink(
      { name: "player", query: playerRoute.query },
      "movie-1",
    ).to as { query: Record<string, string> }

    expect(returnedDetail).toEqual({
      name: "detail",
      params: { id: "movie-1" },
      query: {
        actor: "Mina Kaze",
        back: "actor",
        browse: "library",
        selected: "movie-1",
      },
    })
    expect(resolveNavigationBackLink(
      { name: "detail", query: returnedDetail.query },
      "movie-1",
    )).toEqual({
      labelKey: "shell.backActor",
      to: {
        name: "actor-detail",
        params: { actorName: "Mina Kaze" },
        query: { selected: "movie-1" },
      },
    })
  })

  it("keeps the detail parent when hopping to the next player movie", () => {
    const nextPlayerRoute = buildPlayerRouteFromBrowseIntent(
      "movie-2",
      {
        actor: "Mina Kaze",
        autoplay: "1",
        back: "detail",
        browse: "library",
        detailBack: "actor",
        selected: "movie-1",
      },
      "library",
      "detail",
    ) as { query: Record<string, string> }

    expect(nextPlayerRoute.query).toEqual({
      actor: "Mina Kaze",
      autoplay: "1",
      back: "detail",
      browse: "library",
      detailBack: "actor",
      selected: "movie-2",
    })
  })

  it("parses explicit and legacy back-target query semantics", () => {
    expect(getNavigationBackTarget({ back: "home" })).toBe("home")
    expect(getNavigationBackTarget({ back: "browse" })).toBe("browse")
    expect(getNavigationBackTarget({ back: "actor" })).toBe("actor")
    expect(getNavigationBackTarget({ from: "history" })).toBe("history")
    expect(getNavigationBackTarget({ from: "curated-frames" })).toBe("curated-frames")
    expect(getNavigationBackTarget({ from: "favorites" })).toBe("detail")
  })

  it("resolves browse-launched player back links to the browse page", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: {
            back: "browse",
            browse: "favorites",
            q: "Mina",
            selected: "movie-1",
            tab: "new",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backLibrary",
      to: {
        name: "favorites",
        query: {
          q: "Mina",
          selected: "movie-1",
          tab: "new",
        },
      },
    })
  })

  it("resolves detail-launched player back links to the detail page", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: {
            back: "detail",
            browse: "favorites",
            q: "Mina",
            selected: "movie-1",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backDetail",
      to: {
        name: "detail",
        params: { id: "movie-1" },
        query: {
          browse: "favorites",
          q: "Mina",
          selected: "movie-1",
        },
      },
    })
  })

  it("resolves actor filmography back links to the actor detail page", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "detail",
          query: {
            actor: "Mina Kaze",
            back: "actor",
            selected: "movie-1",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backActor",
      to: {
        name: "actor-detail",
        params: { actorName: "Mina Kaze" },
        query: {
          selected: "movie-1",
        },
      },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: {
            actor: "Mina Kaze",
            back: "actor",
            selected: "movie-1",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backActor",
      to: {
        name: "actor-detail",
        params: { actorName: "Mina Kaze" },
        query: {
          selected: "movie-1",
        },
      },
    })
  })

  it("resolves actor detail pages back to the actor library", () => {
    expect(
      resolveNavigationBackLink({
        name: "actor-detail",
        query: {},
      }),
    ).toEqual({
      labelKey: "shell.backActors",
      to: { name: "actors" },
    })
  })

  it("resolves detail-origin actor pages back to the originating detail page", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "actor-detail",
          query: {
            back: "detail",
            browse: "favorites",
            q: "star",
            selected: "movie-1",
            tab: "top-rated",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backDetail",
      to: {
        name: "detail",
        params: { id: "movie-1" },
        query: {
          browse: "favorites",
          q: "star",
          selected: "movie-1",
          tab: "top-rated",
        },
      },
    })
  })

  it("resolves special player sources and detail routes via the same helper", () => {
    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: { back: "home" },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backHome",
      to: { name: "home" },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "player",
          query: { back: "history" },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backHistory",
      to: { name: "history" },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "detail",
          query: {
            browse: "trash",
            q: "Mina",
            selected: "movie-1",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backLibrary",
      to: {
        name: "trash",
        query: {
          q: "Mina",
          selected: "movie-1",
        },
      },
    })

    expect(
      resolveNavigationBackLink(
        {
          name: "detail",
          query: {
            back: "home",
          },
        },
        "movie-1",
      ),
    ).toEqual({
      labelKey: "shell.backHome",
      to: { name: "home" },
    })
  })

  it.each([
    ["photo-detail", undefined, { name: "photos" }, "shell.backPhotos"],
    ["photo-viewer", undefined, { name: "photos" }, "shell.backPhotos"],
    ["photo-viewer", "/photos/photo-1", "/photos/photo-1", "shell.backDetail"],
    ["photo-viewer", "/photos?q=sample", "/photos?q=sample", "shell.backPhotos"],
    ["photo-viewer", "https://example.com", { name: "photos" }, "shell.backPhotos"],
    ["photo-viewer", "//example.com", { name: "photos" }, "shell.backPhotos"],
  ])("keeps %s return navigation in the photo flow (%s)", (name, returnTo, to, labelKey) => {
    expect(resolveNavigationBackLink({ name, query: returnTo ? { returnTo } : {} })).toEqual({ to, labelKey })
  })

  it("resolves comic detail back links to the comic library", () => {
    expect(
      resolveNavigationBackLink({
        name: "comic-detail",
        query: {},
      }),
    ).toEqual({
      labelKey: "shell.backComics",
      to: { name: "comics" },
    })
  })

  it("resolves comic reader back links to their recorded source route", () => {
    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {
          returnTo: "/comics/comic-1",
        },
      }),
    ).toEqual({
      labelKey: "shell.backDetail",
      to: "/comics/comic-1",
    })

    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {
          returnTo: "/comics?q=artist&sort=fileName",
        },
      }),
    ).toEqual({
      labelKey: "shell.backComics",
      to: "/comics?q=artist&sort=fileName",
    })

    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {
          returnTo: "/settings?section=comics",
        },
      }),
    ).toEqual({
      labelKey: "shell.backPrevious",
      to: "/settings?section=comics",
    })
  })

  it("falls back from direct comic reader links to the comic library", () => {
    expect(
      resolveNavigationBackLink({
        name: "comic-reader",
        query: {},
      }),
    ).toEqual({
      labelKey: "shell.backComics",
      to: { name: "comics" },
    })
  })
})
