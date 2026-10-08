# Bundled macOS playback engine materials

End users do not install Homebrew or MPV. These materials are for recipients who
want to inspect, modify or rebuild the third-party libraries shipped with Curated.
The arm64 runtime targets macOS 15+. It includes GPL/LGPL and other components;
their own license and copyright notices apply. See THIRD-PARTY-NOTICES.txt and
component license directories. Source archives also carry upstream notices.

`build-inputs.json` maps each package and recipe resource to its original HTTPS
URL, version or Git revision, and SHA256. `sources/<sha256>` contains the actual
archive or patch, without its original filename; identify its format with `file`,
verify its checksum with `shasum -a 256`, and unpack each archive separately with
`tar -xf sources/<sha256> -C <empty-directory>`. Preserve patches as files.
Git resources are upstream archives of the exact recorded revision.
`metadata/<package>/recipe.rb` is the recipe from the pinned binary bottle,
including build options, patches and resource placement. The adjacent SBOM records
upstream and original bottle provenance. These are not recipes resolved from
Homebrew's current moving HEAD.

To rebuild a component, use an Apple Silicon macOS build host with Xcode command
line tools and the build tools named in that recipe (Meson/Ninja, CMake or Autotools
as applicable). Extract the recorded main source, apply the recipe's stable
patches in order, unpack its resources into the locations in `def install`, and
run that recipe's configure/build/install commands against your dependency
prefix. Header resources for libplacebo and Vulkan are included. Shaderc's
recorded glslang and SPIRV resources belong in its third_party directories.
Homebrew DSL helpers such as `std_meson_args` and `std_cmake_args` set the install
prefix/build type; when building directly use your own prefix. Compilers and
common build tools are not redistributed. This is a source reconstruction guide,
not a claim of byte-for-byte reproduction of Homebrew's compiler/toolchain.

Curated changes only dylib paths, architecture selection and signatures; it does
not patch these libraries' source. The matching Curated repository contains
`scripts/release/release_lib/macos_native.py`: it copies the locked dylib closure,
rewrites IDs/dependencies to sibling `@loader_path` names, removes rpaths and signs
ad hoc. In a copy of Curated Desktop.app, libraries can be replaced with rebuilt
ABI-compatible arm64 versions in `Contents/Resources/app/native-player-macos/`;
re-sign the modified app with `codesign --force --deep --sign - <app>`. Production
packaging deliberately requires the reviewed lock and will reject unreviewed
modified inputs; update and review the lock when shipping an engine upgrade.

Build-time header provenance: libplacebo's bottle SBOM records Vulkan Headers
1.4.341.0. Its unrecorded header-only fast_float build input is supplied as 8.2.4,
the Homebrew formula version at that bottle's build date; the exact original
compiler environment was not preserved. Verify source correspondence before any
public redistribution if that upstream provenance is insufficient for your use.
