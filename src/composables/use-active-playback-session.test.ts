import { describe, expect, it } from "vitest"

import {
  activePlaybackSession,
  clearActivePlaybackSession,
  dismissActivePlaybackSession,
  updateActivePlaybackSession,
} from "./use-active-playback-session"

function resetActivePlaybackSession() {
  clearActivePlaybackSession()
}

describe("useActivePlaybackSession", () => {
  it("stores a resumable active playback session with a player route target", () => {
    resetActivePlaybackSession()

    updateActivePlaybackSession({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 42.6,
      durationSec: 120,
      status: "playing",
      routeQuery: {
        back: "browse",
        browse: "library",
        autoplay: "0",
        t: "12",
      },
    })

    expect(activePlaybackSession.value).toMatchObject({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 42.6,
      durationSec: 120,
      progressPercent: 35.5,
      status: "playing",
      resumeRouteTarget: {
        name: "player",
        params: { id: "movie-1" },
        query: {
          back: "browse",
          browse: "library",
          autoplay: "1",
          t: "43",
        },
      },
    })
  })

  it("hides tiny and near-ended sessions", () => {
    resetActivePlaybackSession()

    updateActivePlaybackSession({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 4.9,
      durationSec: 120,
      status: "paused",
      routeQuery: {},
    })
    expect(activePlaybackSession.value).toBeNull()

    updateActivePlaybackSession({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 115,
      durationSec: 120,
      status: "paused",
      routeQuery: {},
    })
    expect(activePlaybackSession.value).toBeNull()
  })

  it("dismisses the current session until a newer update arrives", () => {
    resetActivePlaybackSession()

    updateActivePlaybackSession({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 40,
      durationSec: 120,
      status: "paused",
      routeQuery: {},
    })
    expect(activePlaybackSession.value?.movieId).toBe("movie-1")

    dismissActivePlaybackSession("movie-1")
    expect(activePlaybackSession.value).toBeNull()

    updateActivePlaybackSession({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 44,
      durationSec: 120,
      status: "playing",
      routeQuery: {},
    })
    expect(activePlaybackSession.value?.positionSec).toBe(44)
  })

  it("clears only the matching session when a movie id is supplied", () => {
    resetActivePlaybackSession()

    updateActivePlaybackSession({
      movieId: "movie-1",
      title: "Movie title",
      positionSec: 40,
      durationSec: 120,
      status: "paused",
      routeQuery: {},
    })

    clearActivePlaybackSession("other-movie")
    expect(activePlaybackSession.value?.movieId).toBe("movie-1")

    clearActivePlaybackSession("movie-1")
    expect(activePlaybackSession.value).toBeNull()
  })

  it("skips sub-epsilon position updates but never skips status changes", () => {
    resetActivePlaybackSession()

    const publish = (positionSec: number, status: "playing" | "paused" = "playing") => {
      updateActivePlaybackSession({
        movieId: "movie-1",
        title: "Movie title",
        positionSec,
        durationSec: 120,
        status,
        routeQuery: {},
      })
    }

    publish(40)
    expect(activePlaybackSession.value?.positionSec).toBe(40)

    // timeupdate ticks land far below the epsilon and must not republish
    publish(40.25)
    publish(40.5)
    expect(activePlaybackSession.value?.positionSec).toBe(40)

    publish(41)
    expect(activePlaybackSession.value?.positionSec).toBe(41)

    // a pause at the same position is a material change and publishes immediately
    publish(41, "paused")
    expect(activePlaybackSession.value?.status).toBe("paused")
    expect(activePlaybackSession.value?.positionSec).toBe(41)
  })
})
