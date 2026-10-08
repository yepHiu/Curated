# macOS native view

`../native-player-macos.mm` is Curated's Node-API/Cocoa adapter. It dynamically loads a local libmpv and composites its OpenGL Render API view beneath Electron's transparent controls. No libmpv implementation or binary is vendored here.

`include/mpv/{client.h,render.h,render_gl.h}` are unchanged ISC-licensed public API headers from [mpv v0.41.0](https://github.com/mpv-player/mpv/tree/v0.41.0/include/mpv). Their copyright and permission notices remain in each file. SHA-256:

| Header | SHA-256 |
| --- | --- |
| client.h | 3c073236a09cb456c6e80587d5523c8771c32fb22ca014e0f99cc3d43905b75c |
| render.h | 192691941602052f00df0587f126246c48785a1cf21de68d22a92ea1908d1c55 |
| render_gl.h | 48662c0ed9872a14dd9e1684105c97f69f94a0414709c254c5d372adc41d2e69 |

Build with Xcode Command Line Tools and the installed Node distribution's `include/node` headers (`CURATED_NODE_INCLUDE` overrides the directory). Node-API v8 avoids Electron ABI-specific rebuilding. The module does not link against Homebrew dylib paths. The runtime is supplied separately by the user via Homebrew mpv or `CURATED_NATIVE_LIBMPV`; it is not copied into the macOS package. The header license does not cover the actual runtime or its dependencies. Self-contained distribution requires separate engine/source/license validation.
