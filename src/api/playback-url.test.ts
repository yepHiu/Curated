import { describe, expect, it } from "vitest"
import { buildMoviePlaybackUrl, resolveMoviePlaybackSourceUrl } from "./playback-url"

describe("buildMoviePlaybackUrl", () => {
  it("bypasses the Vite proxy for loopback Web API dev playback streams", () => {
    expect(
      buildMoviePlaybackUrl("movie 1", {
        env: {
          DEV: true,
          VITE_USE_WEB_API: "true",
        },
        origin: "http://127.0.0.1:5173",
      }),
    ).toBe("http://127.0.0.1:8080/api/library/movies/movie%201/stream")
  })

  it("keeps same-origin playback URLs outside Web API dev mode", () => {
    expect(
      buildMoviePlaybackUrl("movie-1", {
        env: {
          DEV: false,
          VITE_USE_WEB_API: "true",
        },
        origin: "http://127.0.0.1:5173",
      }),
    ).toBe("http://127.0.0.1:5173/api/library/movies/movie-1/stream")
  })

  it("keeps an explicit API base URL override", () => {
    expect(
      buildMoviePlaybackUrl("movie-1", {
        env: {
          DEV: true,
          VITE_API_BASE_URL: "http://192.168.1.10:8081/api/",
          VITE_USE_WEB_API: "true",
        },
        origin: "http://127.0.0.1:5173",
      }),
    ).toBe("http://192.168.1.10:8081/api/library/movies/movie-1/stream")
  })

  it("rewrites same-origin direct stream descriptor URLs through the playback stream URL resolver", () => {
    expect(
      resolveMoviePlaybackSourceUrl("movie-1", "/api/library/movies/movie-1/stream", {
        env: {
          DEV: true,
          VITE_USE_WEB_API: "true",
        },
        origin: "http://127.0.0.1:5173",
      }),
    ).toBe("http://127.0.0.1:8080/api/library/movies/movie-1/stream")
  })

  it("rewrites playback session descriptor URLs through the playback stream URL resolver", () => {
    expect(
      resolveMoviePlaybackSourceUrl("movie-1", "/api/playback/sessions/sess-1/hls/index.m3u8", {
        env: {
          DEV: true,
          VITE_USE_WEB_API: "true",
        },
        origin: "http://127.0.0.1:5173",
      }),
    ).toBe("http://127.0.0.1:8080/api/playback/sessions/sess-1/hls/index.m3u8")
  })
})
