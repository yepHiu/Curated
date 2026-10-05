param([Parameter(Mandatory=$true)][int]$ElectronProcessId)
$ErrorActionPreference = 'Stop'
# 保留 UTF-8 BOM，Windows PowerShell 5.1 必须准确读取下面的中文注释。
Add-Type @'
using System;
using System.Runtime.InteropServices;
public class CuratedWindowProbe {
    [StructLayout(LayoutKind.Sequential)] public struct Rect { public int Left, Top, Right, Bottom; }
    [StructLayout(LayoutKind.Sequential)] public struct Point { public int X, Y; }
    public delegate bool Callback(IntPtr h, IntPtr p);
    [DllImport("user32.dll")] public static extern bool EnumWindows(Callback c, IntPtr p);
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint p);
    [DllImport("user32.dll")] public static extern bool GetClientRect(IntPtr h, out Rect r);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out Rect r);
    [DllImport("user32.dll")] public static extern bool ClientToScreen(IntPtr h, ref Point p);
    [DllImport("user32.dll")] public static extern IntPtr GetWindow(IntPtr h, uint c);
    [DllImport("user32.dll")] public static extern IntPtr GetWindowLongPtr(IntPtr h, int i);
    [DllImport("user32.dll")] public static extern IntPtr SendMessage(IntPtr h, uint m, IntPtr w, IntPtr l);
    [DllImport("user32.dll", EntryPoint="PostMessageW", SetLastError=true)] private static extern bool PostMessageW(IntPtr h, uint m, IntPtr w, IntPtr l);
    [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int width, int height, uint flags);
    [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int cmd);
    [DllImport("user32.dll")] public static extern uint GetDpiForWindow(IntPtr h);
    [DllImport("user32.dll")] public static extern int GetSystemMetricsForDpi(int index, uint dpi);
    [DllImport("user32.dll")] public static extern bool SetProcessDpiAwarenessContext(IntPtr v);

    /* 测试只选本次 Electron 的 helper 窗口，不触碰现有播放器。 */
    public static IntPtr FindHost(uint pid) {
        IntPtr result = IntPtr.Zero;
        EnumWindows((h,p) => { uint current; GetWindowThreadProcessId(h,out current); if(current == pid){result=h;return false;}return true; },IntPtr.Zero);
        return result;
    }
    /* owner 和 PID 同时匹配，不能选到别的应用或别的 prototype。 */
    public static IntPtr FindOverlay(IntPtr host, uint pid) {
        IntPtr result = IntPtr.Zero;
        EnumWindows((h,p) => { uint current; GetWindowThreadProcessId(h,out current); if(GetWindow(h,4)==host && current==pid){result=h;return false;}return true; },IntPtr.Zero);
        return result;
    }
    /* Windows 实际非客户区命中结果，旧 resizable 透明层在角上返回 13/17。 */
    public static int HitTest(IntPtr h, int x, int y) {
        return (int)SendMessage(h,0x84,IntPtr.Zero,new IntPtr((y<<16)|(x&0xffff)));
    }
    /* 直接读物理几何，测试不沿用主进程坐标转换以免同错同过。 */
    public static int[] ClientBounds(IntPtr h) {
        Rect r; Point p = new Point(); GetClientRect(h,out r); ClientToScreen(h,ref p);
        return new int[]{p.X,p.Y,r.Right,r.Bottom};
    }
    public static int[] WindowBounds(IntPtr h) {
        Rect r;GetWindowRect(h,out r);return new int[]{r.Left,r.Top,r.Right-r.Left,r.Bottom-r.Top};
    }
    public static int[] OverlayHits(IntPtr h) {
        int[] r=WindowBounds(h);int left=r[0]+1,top=r[1]+1,right=r[0]+r[2]-2,bottom=r[1]+r[3]-2;
        int x=r[0]+r[2]/2,y=r[1]+r[3]/2;
        return new int[]{HitTest(h,left,top),HitTest(h,x,top),HitTest(h,right,top),HitTest(h,left,y),HitTest(h,right,y),HitTest(h,left,bottom),HitTest(h,x,bottom),HitTest(h,right,bottom)};
    }
    /* 关闭消息由明确的 Unicode Win32 入口发送，失败不能伪装成正常退出。 */
    public static void CloseHost(IntPtr h) {
        if (!PostMessageW(h, 0x10, IntPtr.Zero, IntPtr.Zero)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
    }
}
'@
[CuratedWindowProbe]::SetProcessDpiAwarenessContext([IntPtr](-4)) | Out-Null
$taskDeadline = [DateTime]::UtcNow.AddSeconds(8)
$taskHost = [IntPtr]::Zero
$taskOverlay = [IntPtr]::Zero
while ([DateTime]::UtcNow -lt $taskDeadline) {
    $taskProcess = Get-CimInstance Win32_Process -Filter "Name = 'native-player-host.exe' AND ParentProcessId = $ElectronProcessId" | Select-Object -First 1
    if ($taskProcess) {
        $taskHost = [CuratedWindowProbe]::FindHost($taskProcess.ProcessId)
        $taskOverlay = [CuratedWindowProbe]::FindOverlay($taskHost,$ElectronProcessId)
        if ($taskOverlay -ne [IntPtr]::Zero -and [CuratedWindowProbe]::IsWindowVisible($taskOverlay)) { break }
    }
    Start-Sleep -Milliseconds 80
}
if ($taskOverlay -eq [IntPtr]::Zero -or -not [CuratedWindowProbe]::IsWindowVisible($taskOverlay)) { throw 'Own prototype not ready' }

$taskSamples = New-Object System.Collections.Generic.List[object]
function Add-GeometrySample([string]$Label) {
    # 等待同步完成后读取宿主和透明层两套实际 Win32 边界。
    Start-Sleep -Milliseconds 120
    $taskSamples.Add(@{label=$Label;hostClient=[CuratedWindowProbe]::ClientBounds($taskHost);overlay=[CuratedWindowProbe]::WindowBounds($taskOverlay)})
}

$taskInitial = [CuratedWindowProbe]::WindowBounds($taskHost)
$taskDpi = [CuratedWindowProbe]::GetDpiForWindow($taskHost)
$taskFrame = [CuratedWindowProbe]::GetSystemMetricsForDpi(33,$taskDpi) + [CuratedWindowProbe]::GetSystemMetricsForDpi(92,$taskDpi)
$taskCaption = [CuratedWindowProbe]::GetSystemMetricsForDpi(4,$taskDpi)
$taskHostHits = @(
    [CuratedWindowProbe]::HitTest($taskHost,($taskInitial[0]+2),($taskInitial[1]+2)),
    [CuratedWindowProbe]::HitTest($taskHost,($taskInitial[0]+$taskInitial[2]-2),($taskInitial[1]+$taskInitial[3]-2)),
    [CuratedWindowProbe]::HitTest($taskHost,($taskInitial[0]+$taskInitial[2]/2),($taskInitial[1]+$taskFrame+$taskCaption/2))
)
$taskInitialHits = [CuratedWindowProbe]::OverlayHits($taskOverlay)
Add-GeometrySample 'initial'
for ($taskIndex=0; $taskIndex -lt 4; $taskIndex++) {
    # 仅移动/缩放指定 helper，模拟系统改变外框；不用全局鼠标或用户窗口。
    $taskWidth = [Math]::Max([int](660*$taskDpi/96),$taskInitial[2]-71*($taskIndex+1))
    $taskHeight = [Math]::Max([int](500*$taskDpi/96),$taskInitial[3]-53*($taskIndex+1))
    [CuratedWindowProbe]::SetWindowPos($taskHost,[IntPtr]::Zero,($taskInitial[0]+17*$taskIndex),($taskInitial[1]+11*$taskIndex),$taskWidth,$taskHeight,0x14) | Out-Null
    Add-GeometrySample "move-resize-$taskIndex"
}
[CuratedWindowProbe]::ShowWindow($taskHost,6) | Out-Null
Start-Sleep -Milliseconds 150
$taskHidden = -not [CuratedWindowProbe]::IsWindowVisible($taskOverlay)
[CuratedWindowProbe]::ShowWindow($taskHost,9) | Out-Null
Add-GeometrySample 'restored'

$taskResult = @{overlayHits=$taskInitialHits;finalOverlayHits=[CuratedWindowProbe]::OverlayHits($taskOverlay);hostHits=$taskHostHits;overlayStyle=[CuratedWindowProbe]::GetWindowLongPtr($taskOverlay,-16).ToInt64();samples=$taskSamples.ToArray();overlayHiddenWhenMinimized=$taskHidden;overlayVisibleAfterRestore=[CuratedWindowProbe]::IsWindowVisible($taskOverlay)}
# 正常关闭交给主进程保存/回收；测试不直接杀正常运行的 helper。
[CuratedWindowProbe]::CloseHost($taskHost)
$taskResult | ConvertTo-Json -Depth 5 -Compress
