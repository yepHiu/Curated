#define WIN32_LEAN_AND_MEAN
#define _WIN32_WINNT 0x0A00
#define NTDDI_VERSION 0x0A000003
#ifndef UNICODE
#define UNICODE
#endif
#define _UNICODE
#include <windows.h>
#include <shellapi.h>
#include <dwmapi.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static HWND host_window, overlay_window;
static HANDLE parent_process;
static DWORD parent_pid;
static BOOL fullscreen, closing;
static WINDOWPLACEMENT saved_placement = { .length = sizeof(WINDOWPLACEMENT) };
static DWORD saved_style;
#define HOST_COMMAND (WM_APP + 1)
#define HOST_INPUT_END (WM_APP + 2)

/* 宿主独占客户端物理几何；主进程仅根据报告同步透明层可见性。 */
static void report_state(const char *event) {
    RECT rect;
    POINT point = {0, 0};
    GetClientRect(host_window, &rect);
    ClientToScreen(host_window, &point);
    /* 原生侧同步位置消除拖动时两个窗口的管道往返延迟。 */
    if (overlay_window && !IsIconic(host_window)) {
        SetWindowPos(overlay_window, NULL, point.x, point.y, rect.right, rect.bottom,
                     SWP_NOZORDER | SWP_NOACTIVATE);
    }
    printf("{\"event\":\"%s\",\"handle\":\"%llu\",\"x\":%ld,\"y\":%ld,"
           "\"width\":%ld,\"height\":%ld,\"dpi\":%u,\"visible\":%s,\"fullscreen\":%s,\"maximized\":%s}\n",
           event, (unsigned long long)(uintptr_t)host_window, point.x, point.y,
           rect.right, rect.bottom, GetDpiForWindow(host_window),
           IsWindowVisible(host_window) && !IsIconic(host_window) ? "true" : "false", fullscreen ? "true" : "false",
           !fullscreen && IsZoomed(host_window) ? "true" : "false");
    fflush(stdout);
}

/* 内部管道使用固定字段；不解析媒体 URL、文件路径或通用原生命令。 */
static unsigned long long number_field(const char *json, const char *key) {
    char needle[64];
    snprintf(needle, sizeof(needle), "\"%s\"", key);
    const char *value = strstr(json, needle);
    if (!value || !(value = strchr(value + strlen(needle), ':'))) return 0;
    value++;
    while (*value == ' ' || *value == '\t' || *value == '"') value++;
    return strtoull(value, NULL, 10);
}

/* 只接受受限的动作名，字段长度上限由读取线程约束。 */
static BOOL action_field(const char *json, char *action, size_t capacity) {
    const char *value = strstr(json, "\"action\"");
    if (!value || !(value = strchr(value + 8, ':'))) return FALSE;
    value++;
    while (*value == ' ' || *value == '\t') value++;
    if (*value++ != '"') return FALSE;
    const char *end = strchr(value, '"');
    if (!end || (size_t)(end - value) >= capacity) return FALSE;
    memcpy(action, value, (size_t)(end - value));
    action[end - value] = 0;
    return TRUE;
}

/* 只修改自有宿主标题；有界 UTF-16 十六进制字段不含 JSON 转义。 */
static BOOL set_title(const char *json) {
    const char *value = strstr(json, "\"titleHex\"");
    if (!value || !(value = strchr(value + 10, ':'))) return FALSE;
    value++;
    while (*value == ' ' || *value == '\t') value++;
    if (*value++ != '"') return FALSE;
    WCHAR title[481];
    size_t length = 0;
    while (*value && *value != '"') {
        if (length >= 480) return FALSE;
        unsigned int unit = 0;
        for (int i = 0; i < 4; i++) {
            char digit = *value++;
            if (digit >= '0' && digit <= '9') unit = unit * 16 + (unsigned int)(digit - '0');
            else if (digit >= 'a' && digit <= 'f') unit = unit * 16 + (unsigned int)(digit - 'a' + 10);
            else return FALSE;
        }
        if (unit < 32 || unit == 127) return FALSE;
        title[length++] = (WCHAR)unit;
    }
    if (*value != '"' || !length) return FALSE;
    title[length] = 0;
    return SetWindowTextW(host_window, title);
}

/* 全屏切换保留原窗口位置及最大化状态，覆盖宿主所属显示器。 */
static void toggle_fullscreen(void) {
    if (!fullscreen) {
        saved_style = (DWORD)GetWindowLongPtrW(host_window, GWL_STYLE);
        GetWindowPlacement(host_window, &saved_placement);
        MONITORINFO monitor = { .cbSize = sizeof(MONITORINFO) };
        GetMonitorInfoW(MonitorFromWindow(host_window, MONITOR_DEFAULTTONEAREST), &monitor);
        fullscreen = TRUE;
        SetWindowLongPtrW(host_window, GWL_STYLE, saved_style & ~(DWORD)WS_OVERLAPPEDWINDOW);
        SetWindowPos(host_window, NULL, monitor.rcMonitor.left, monitor.rcMonitor.top,
                     monitor.rcMonitor.right - monitor.rcMonitor.left,
                     monitor.rcMonitor.bottom - monitor.rcMonitor.top,
                     SWP_NOZORDER | SWP_FRAMECHANGED);
    } else {
        fullscreen = FALSE;
        SetWindowLongPtrW(host_window, GWL_STYLE, saved_style);
        SetWindowPlacement(host_window, &saved_placement);
        SetWindowPos(host_window, NULL, 0, 0, 0, 0,
                     SWP_NOMOVE | SWP_NOSIZE | SWP_NOZORDER | SWP_FRAMECHANGED);
    }
    report_state("bounds");
}

/* 浏览器 HWND 必须属于启动本 helper 的 Electron，拒绝接管其它应用。 */
static BOOL attach_overlay(unsigned long long handle) {
    HWND candidate = (HWND)(uintptr_t)handle;
    DWORD candidate_pid = 0;
    GetWindowThreadProcessId(candidate, &candidate_pid);
    if (!IsWindow(candidate) || candidate_pid != parent_pid || candidate == host_window) return FALSE;
    SetLastError(0);
    LONG_PTR previous = SetWindowLongPtrW(candidate, GWLP_HWNDPARENT, (LONG_PTR)host_window);
    if (!previous && GetLastError()) return FALSE;
    overlay_window = candidate;
    SetWindowPos(overlay_window, HWND_TOP, 0, 0, 0, 0,
                 SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE);
    /* 首次展示前也按宿主客户端物理区域对齐，Electron 不再重复设置 DIP 边界。 */
    report_state("bounds");
    return TRUE;
}

/* 命令只在 UI 线程执行；响应按 ID 与主进程的请求关联。 */
static void apply_command(const char *json) {
    char action[32];
    unsigned long long id = number_field(json, "id");
    BOOL ok = action_field(json, action, sizeof(action));
    if (ok && strcmp(action, "attach") == 0) {
        ok = attach_overlay(number_field(json, "handle"));
    } else if (ok && strcmp(action, "title") == 0) {
        ok = set_title(json);
    } else if (ok && strcmp(action, "fullscreen") == 0) {
        toggle_fullscreen();
    } else if (ok && strcmp(action, "minimize") == 0) {
        ShowWindow(host_window, SW_MINIMIZE);
    } else if (ok && strcmp(action, "maximize") == 0) {
        if (fullscreen) toggle_fullscreen();
        else ShowWindow(host_window, IsZoomed(host_window) ? SW_RESTORE : SW_MAXIMIZE);
        report_state("bounds");
    } else if (ok && strcmp(action, "focus") == 0) {
        /* 聚焦不改变最大化状态；最小化时按原先的位置恢复。 */
        if (IsIconic(host_window)) ShowWindow(host_window, SW_RESTORE);
        SetForegroundWindow(host_window);
        report_state("bounds");
    } else if (ok && strcmp(action, "restore") == 0) {
        ShowWindow(host_window, SW_RESTORE);
        report_state("bounds");
    } else if (ok && strcmp(action, "resize") == 0) {
        unsigned long long width = number_field(json, "width"), height = number_field(json, "height");
        ok = width >= 640 && width <= 4096 && height >= 480 && height <= 2160 && !fullscreen;
        if (ok) SetWindowPos(host_window, NULL, 0, 0, (int)width, (int)height, SWP_NOMOVE | SWP_NOZORDER);
    } else if (ok && strcmp(action, "quit") == 0) {
        /* 主进程先保存进度/停止 mpv，再销毁宿主。 */
        DestroyWindow(host_window);
    } else {
        ok = FALSE;
    }
    printf("{\"event\":\"ack\",\"id\":%llu,\"ok\":%s}\n", id, ok ? "true" : "false");
    fflush(stdout);
}

/* stdin 是仅父进程可写的管道；EOF 或超长消息使宿主退出。 */
static DWORD WINAPI read_commands(LPVOID unused) {
    (void)unused;
    char line[2048];
    while (fgets(line, sizeof(line), stdin)) {
        if (!strchr(line, '\n')) break;
        char *copy = _strdup(line);
        if (!copy || !PostMessageW(host_window, HOST_COMMAND, 0, (LPARAM)copy)) {
            free(copy);
            break;
        }
    }
    PostMessageW(host_window, HOST_INPUT_END, 0, 0);
    return 0;
}

/* Windows 消息只管理本次视频宿主；关闭请求先交给父进程有序回收。 */
static LRESULT CALLBACK window_proc(HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam) {
    switch (message) {
    case WM_CLOSE:
        if (!closing) { closing = TRUE; printf("{\"event\":\"close\"}\n"); fflush(stdout); }
        return 0;
    case WM_MOVE:
    case WM_SIZE:
        if (host_window) report_state("bounds");
        return 0;
    case WM_DPICHANGED: {
        RECT *suggested = (RECT *)lparam;
        SetWindowPos(hwnd, NULL, suggested->left, suggested->top,
                     suggested->right - suggested->left, suggested->bottom - suggested->top,
                     SWP_NOZORDER | SWP_NOACTIVATE);
        report_state("bounds");
        return 0;
    }
    case WM_GETMINMAXINFO: {
        MINMAXINFO *limits = (MINMAXINFO *)lparam;
        limits->ptMinTrackSize.x = MulDiv(640, (int)GetDpiForWindow(hwnd), 96);
        limits->ptMinTrackSize.y = MulDiv(480, (int)GetDpiForWindow(hwnd), 96);
        return 0;
    }
    case WM_TIMER:
        if (WaitForSingleObject(parent_process, 0) == WAIT_OBJECT_0) DestroyWindow(hwnd);
        return 0;
    case HOST_COMMAND:
        apply_command((const char *)lparam);
        free((void *)lparam);
        return 0;
    case HOST_INPUT_END:
        DestroyWindow(hwnd);
        return 0;
    case WM_DESTROY:
        printf("{\"event\":\"closed\"}\n"); fflush(stdout);
        PostQuitMessage(0);
        return 0;
    }
    return DefWindowProcW(hwnd, message, wparam, lparam);
}

/* 无安装/注册副作用的独立宿主；父进程死亡或控制管道关闭即退出。 */
int WINAPI wWinMain(HINSTANCE instance, HINSTANCE previous, PWSTR command_line, int show) {
    (void)previous; (void)command_line; (void)show;
    int count = 0;
    LPWSTR *args = CommandLineToArgvW(GetCommandLineW(), &count);
    if (!args || count != 2) return 2;
    parent_pid = wcstoul(args[1], NULL, 10);
    LocalFree(args);
    parent_process = OpenProcess(SYNCHRONIZE, FALSE, parent_pid);
    if (!parent_process) return 3;
    SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2);
    setvbuf(stdout, NULL, _IONBF, 0);
    WNDCLASSW window_class = {0};
    window_class.lpfnWndProc = window_proc;
    window_class.hInstance = instance;
    window_class.lpszClassName = L"CuratedNativeVideoHost";
    window_class.hCursor = LoadCursorW(NULL, IDC_ARROW);
    window_class.hIcon = LoadIconW(instance, MAKEINTRESOURCEW(100));
    if (!window_class.hIcon) window_class.hIcon = LoadIconW(NULL, IDI_APPLICATION);
    window_class.hbrBackground = (HBRUSH)GetStockObject(BLACK_BRUSH);
    if (!RegisterClassW(&window_class)) return 4;
    /* 初始尺寸与最小尺寸都用 DIP，避免高 DPI 屏幕启动时只留下狭窄控件区。 */
    UINT system_dpi = GetDpiForSystem();
    MONITORINFO initial_monitor = { .cbSize = sizeof(MONITORINFO) };
    POINT initial_point = {0, 0};
    GetMonitorInfoW(MonitorFromPoint(initial_point, MONITOR_DEFAULTTOPRIMARY), &initial_monitor);
    int initial_width = MulDiv(1000, (int)system_dpi, 96);
    int initial_height = MulDiv(800, (int)system_dpi, 96);
    int work_width = initial_monitor.rcWork.right - initial_monitor.rcWork.left;
    int work_height = initial_monitor.rcWork.bottom - initial_monitor.rcWork.top;
    if (initial_width > work_width) initial_width = work_width;
    if (initial_height > work_height) initial_height = work_height;
    host_window = CreateWindowExW(0, window_class.lpszClassName, L"Curated · Native Playback",
                                 WS_OVERLAPPEDWINDOW | WS_CLIPCHILDREN,
                                 CW_USEDEFAULT, CW_USEDEFAULT, initial_width, initial_height,
                                 NULL, NULL, instance, NULL);
    if (!host_window) return 5;
    BOOL dark = TRUE;
    DwmSetWindowAttribute(host_window, 20, &dark, sizeof(dark));
    SetTimer(host_window, 1, 250, NULL);
    HANDLE reader = CreateThread(NULL, 0, read_commands, NULL, 0, NULL);
    if (!reader) return 6;
    ShowWindow(host_window, SW_SHOW);
    /* windowsHide 的 STARTUPINFO 可覆盖第一次 ShowWindow，第二次显式展示交互宿主。 */
    ShowWindow(host_window, SW_SHOW);
    report_state("ready");
    MSG message;
    while (GetMessageW(&message, NULL, 0, 0) > 0) {
        TranslateMessage(&message);
        DispatchMessageW(&message);
    }
    CloseHandle(reader);
    CloseHandle(parent_process);
    return 0;
}
