// Curated's Cocoa/libmpv view. Only the trusted Electron main process loads this module.
#import <Cocoa/Cocoa.h>
#import <OpenGL/gl3.h>
#include <node_api.h>
#include <mpv/client.h>
#include <mpv/render_gl.h>
#include <dlfcn.h>
#include <string>
#include <vector>
#include <map>
#include <set>

// Resolve the user's library locally; do not link to a developer's Homebrew cellar.
struct MpvApi {
  void *library = nullptr;
#define MPV_FUNCTION(name) decltype(&name) name = nullptr
  MPV_FUNCTION(mpv_create);
  MPV_FUNCTION(mpv_client_api_version);
  MPV_FUNCTION(mpv_set_option_string);
  MPV_FUNCTION(mpv_initialize);
  MPV_FUNCTION(mpv_terminate_destroy);
  MPV_FUNCTION(mpv_wait_event);
  MPV_FUNCTION(mpv_render_context_create);
  MPV_FUNCTION(mpv_render_context_set_update_callback);
  MPV_FUNCTION(mpv_render_context_update);
  MPV_FUNCTION(mpv_render_context_render);
  MPV_FUNCTION(mpv_render_context_report_swap);
  MPV_FUNCTION(mpv_render_context_free);
#undef MPV_FUNCTION
  bool load(const char *filename) {
    static std::map<std::string, MpvApi> loaded;
    auto previous = loaded.find(filename);
    if (previous != loaded.end()) { *this = previous->second; return true; }
    library = dlopen(filename, RTLD_NOW | RTLD_LOCAL);
    if (!library) return false;
#define LOAD(name) name = reinterpret_cast<decltype(name)>(dlsym(library, #name)); if (!name) { dlclose(library); library = nullptr; return false; }
    LOAD(mpv_create); LOAD(mpv_client_api_version); LOAD(mpv_set_option_string); LOAD(mpv_initialize);
    LOAD(mpv_terminate_destroy); LOAD(mpv_wait_event); LOAD(mpv_render_context_create);
    LOAD(mpv_render_context_set_update_callback); LOAD(mpv_render_context_update); LOAD(mpv_render_context_render);
    LOAD(mpv_render_context_report_swap); LOAD(mpv_render_context_free);
#undef LOAD
    if ((mpv_client_api_version() >> 16) != (MPV_CLIENT_API_VERSION >> 16)) {
      dlclose(library); library = nullptr; return false;
    }
    loaded.emplace(filename, *this);
    return true;
  }
  // Keep the library loaded for process lifetime: system graphics callbacks may outlive a session.
};

@interface CuratedMpvView : NSOpenGLView {
@public
  MpvApi api;
  mpv_handle *player;
  mpv_render_context *renderer;
}
- (void)shutdown;
@end

static void *glAddress(void *, const char *name) { return dlsym(RTLD_DEFAULT, name); }
static void renderUpdate(void *context) {
  // A queued block retains the view, but never a freed mpv context.
  CuratedMpvView *view = (__bridge CuratedMpvView *)context;
  dispatch_async(dispatch_get_main_queue(), ^{
    if (!view->renderer) return;
    [view.openGLContext makeCurrentContext];
    // Advanced control services GPU screenshots even while the window is occluded.
    view->api.mpv_render_context_update(view->renderer);
    [NSOpenGLContext clearCurrentContext];
    [view setNeedsDisplay:YES];
  });
}

@implementation CuratedMpvView
- (BOOL)isOpaque { return YES; }
- (NSView *)hitTest:(NSPoint)point { (void)point; return nil; }
- (void)reshape {
  [super reshape];
  [self.openGLContext update];
  [self setNeedsDisplay:YES];
}
- (void)viewDidChangeBackingProperties {
  [super viewDidChangeBackingProperties];
  [self reshape];
}
- (void)drawRect:(NSRect)dirty {
  (void)dirty;
  if (!renderer) return;
  [self.openGLContext makeCurrentContext];
  NSRect pixels = [self convertRectToBacking:self.bounds];
  if (pixels.size.width < 1 || pixels.size.height < 1) return;
  mpv_opengl_fbo fbo = {0, (int)pixels.size.width, (int)pixels.size.height, 0};
  int flip = 1, block = 0;
  mpv_render_param params[] = {
    {MPV_RENDER_PARAM_OPENGL_FBO, &fbo}, {MPV_RENDER_PARAM_FLIP_Y, &flip},
    {MPV_RENDER_PARAM_BLOCK_FOR_TARGET_TIME, &block}, {MPV_RENDER_PARAM_INVALID, nullptr}
  };
  api.mpv_render_context_render(renderer, params);
  [self.openGLContext flushBuffer];
  api.mpv_render_context_report_swap(renderer);
  [NSOpenGLContext clearCurrentContext];
}
- (void)shutdown {
  if (renderer) {
    api.mpv_render_context_set_update_callback(renderer, nullptr, nullptr);
    [self.openGLContext makeCurrentContext];
    api.mpv_render_context_free(renderer);
    renderer = nullptr;
    [NSOpenGLContext clearCurrentContext];
  }
  if (player) { api.mpv_terminate_destroy(player); player = nullptr; }
  [self removeFromSuperview];
}
@end

struct Instance { CFTypeRef view = nullptr; };
static std::set<Instance *> instances;
static void destroy(Instance *instance) {
  if (!instance || !instance->view) return;
  CuratedMpvView *view = (__bridge CuratedMpvView *)instance->view;
  [view shutdown];
  CFRelease(instance->view);
  instance->view = nullptr;
}
static void finalize(napi_env, void *data, void *) { auto instance = static_cast<Instance *>(data); destroy(instance); instances.erase(instance); delete instance; }
static void cleanup(void *) { for (auto instance : instances) destroy(instance); }
static napi_value fail(napi_env env, const char *code) { napi_throw_error(env, nullptr, code); return nullptr; }
static bool stringValue(napi_env env, napi_value value, std::string &result) {
  size_t size;
  if (napi_get_value_string_utf8(env, value, nullptr, 0, &size) != napi_ok || size > 8192) return false;
  std::vector<char> bytes(size + 1);
  if (napi_get_value_string_utf8(env, value, bytes.data(), bytes.size(), &size) != napi_ok) return false;
  result.assign(bytes.data(), size);
  return result.find('\0') == std::string::npos;
}

static napi_value create(napi_env env, napi_callback_info info) {
  size_t count = 3;
  napi_value args[3];
  napi_get_cb_info(env, info, &count, args, nullptr, nullptr);
  if (![NSThread isMainThread] || count != 3) return fail(env, "MAC_NATIVE_INVALID_ARGUMENT");
  bool isBuffer = false, isArray = false;
  napi_is_buffer(env, args[0], &isBuffer); napi_is_array(env, args[2], &isArray);
  void *buffer = nullptr; size_t length = 0; std::string library;
  if (!isBuffer || !isArray || !stringValue(env, args[1], library) || library.empty() || library[0] != '/') return fail(env, "MAC_NATIVE_INVALID_ARGUMENT");
  napi_get_buffer_info(env, args[0], &buffer, &length);
  if (length != sizeof(void *)) return fail(env, "MAC_NATIVE_INVALID_WINDOW");
  NSView *content = (__bridge NSView *)*static_cast<void **>(buffer);
  if (!content || !content.window) return fail(env, "MAC_NATIVE_INVALID_WINDOW");
  NSOpenGLPixelFormatAttribute attributes[] = {
    NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core, NSOpenGLPFADoubleBuffer,
    NSOpenGLPFAAccelerated, NSOpenGLPFAColorSize, 24, 0
  };
  NSOpenGLPixelFormat *format = [[NSOpenGLPixelFormat alloc] initWithAttributes:attributes];
  CuratedMpvView *view = [[CuratedMpvView alloc] initWithFrame:content.bounds pixelFormat:format];
  if (!view || !view.openGLContext) return fail(env, "MAC_NATIVE_OPENGL_UNAVAILABLE");
  if (!view->api.load(library.c_str())) return fail(env, "MAC_NATIVE_LIBRARY_UNAVAILABLE");
  view->player = view->api.mpv_create();
  if (!view->player) return fail(env, "MAC_NATIVE_ENGINE_FAILED");
  uint32_t options;
  napi_get_array_length(env, args[2], &options);
  for (uint32_t i = 0; i < options; i++) {
    napi_value item; std::string option;
    napi_get_element(env, args[2], i, &item);
    if (!stringValue(env, item, option) || option.rfind("--", 0) != 0) { [view shutdown]; return fail(env, "MAC_NATIVE_INVALID_OPTION"); }
    option.erase(0, 2);
    auto split = option.find('=');
    std::string key = option.substr(0, split), value = split == std::string::npos ? "yes" : option.substr(split + 1);
    if (key.rfind("no-", 0) == 0) { key.erase(0, 3); value = "no"; }
    if (view->api.mpv_set_option_string(view->player, key.c_str(), value.c_str()) < 0) { [view shutdown]; return fail(env, "MAC_NATIVE_OPTION_FAILED"); }
  }
  // Render API owns video output. All controls remain in the Curated UI.
  for (auto option : {std::pair{"vo", "libmpv"}, {"osc", "no"}, {"input-default-bindings", "no"}, {"hwdec", "auto-safe"}}) {
    if (view->api.mpv_set_option_string(view->player, option.first, option.second) < 0) { [view shutdown]; return fail(env, "MAC_NATIVE_OPTION_FAILED"); }
  }
  if (view->api.mpv_initialize(view->player) < 0) { [view shutdown]; return fail(env, "MAC_NATIVE_ENGINE_FAILED"); }
  view.wantsBestResolutionOpenGLSurface = YES;
  view.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
  // Electron's handle is its content NSView. Insert video behind Chromium's transparent children.
  [content addSubview:view positioned:NSWindowBelow relativeTo:nil];
  [view.openGLContext makeCurrentContext];
  GLint interval = 1;
  [view.openGLContext setValues:&interval forParameter:NSOpenGLContextParameterSwapInterval];
  mpv_opengl_init_params gl = {glAddress, nullptr};
  int advanced = 1;
  mpv_render_param params[] = {{MPV_RENDER_PARAM_API_TYPE, const_cast<char *>(MPV_RENDER_API_TYPE_OPENGL)},
    {MPV_RENDER_PARAM_OPENGL_INIT_PARAMS, &gl}, {MPV_RENDER_PARAM_ADVANCED_CONTROL, &advanced}, {MPV_RENDER_PARAM_INVALID, nullptr}};
  if (view->api.mpv_render_context_create(&view->renderer, view->player, params) < 0) { [view shutdown]; return fail(env, "MAC_NATIVE_RENDER_FAILED"); }
  view->api.mpv_render_context_set_update_callback(view->renderer, renderUpdate, (__bridge void *)view);
  [NSOpenGLContext clearCurrentContext];
  auto instance = new Instance{CFBridgingRetain(view)};
  instances.insert(instance);
  napi_value result;
  napi_create_external(env, instance, finalize, nullptr, &result);
  return result;
}

static Instance *argument(napi_env env, napi_callback_info info) {
  size_t count = 1; napi_value value; void *data = nullptr;
  napi_get_cb_info(env, info, &count, &value, nullptr, nullptr);
  if (count != 1 || napi_get_value_external(env, value, &data) != napi_ok) return nullptr;
  return static_cast<Instance *>(data);
}
static napi_value stop(napi_env env, napi_callback_info info) {
  auto instance = argument(env, info);
  if (!instance) return fail(env, "MAC_NATIVE_INVALID_INSTANCE");
  destroy(instance);
  napi_value result; napi_get_undefined(env, &result); return result;
}
static napi_value poll(napi_env env, napi_callback_info info) {
  auto instance = argument(env, info);
  if (!instance) return fail(env, "MAC_NATIVE_INVALID_INSTANCE");
  bool ended = !instance->view;
  if (instance->view) {
    CuratedMpvView *view = (__bridge CuratedMpvView *)instance->view;
    for (int i = 0; i < 64; i++) {
      mpv_event *event = view->api.mpv_wait_event(view->player, 0);
      if (event->event_id == MPV_EVENT_SHUTDOWN) { ended = true; break; }
      if (event->event_id == MPV_EVENT_NONE) break;
    }
  }
  napi_value result; napi_get_boolean(env, ended, &result); return result;
}
static napi_value initialize(napi_env env, napi_value exports) {
  napi_add_env_cleanup_hook(env, cleanup, nullptr);
  napi_property_descriptor properties[] = {
    {"create", nullptr, create, nullptr, nullptr, nullptr, napi_default, nullptr},
    {"stop", nullptr, stop, nullptr, nullptr, nullptr, napi_default, nullptr},
    {"poll", nullptr, poll, nullptr, nullptr, nullptr, napi_default, nullptr}
  };
  napi_define_properties(env, exports, 3, properties);
  return exports;
}
NAPI_MODULE(NODE_GYP_MODULE_NAME, initialize)
