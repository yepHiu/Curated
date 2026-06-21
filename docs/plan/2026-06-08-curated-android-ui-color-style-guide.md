# Curated Android UI 颜色与样式规范

日期：2026-06-08
状态：Android UI Foundation Draft
目标读者：后续开发 Curated Android App 的 Kotlin / Jetpack Compose 开发者
事实来源：`src/style.css`、`src/lib/theme-storage.ts`、`index.html`

## 1. 总结

Curated Android App 应沿用当前 Web/Electron 的视觉语言：**默认深色、内容优先、低干扰媒体库界面、粉色品牌主色**。

核心结论：

- 默认主题：深色。
- 品牌主色：`#FE628E`。
- 深色主背景：`#0D0F1A`。
- 浅色主背景：`#F4F6FC`。
- Kotlin 组件内禁止散落硬编码 hex，应通过语义 token 或 `MaterialTheme.colorScheme` 使用。
- 深色和浅色都要实现，不允许只实现深色后用系统反色推导浅色。
- 媒体封面、视频画面可以使用局部黑色遮罩；普通业务 UI 不应引入新的主色系。

## 2. Android 主题策略

Android 首版应支持三种用户偏好：

| 偏好 | 行为 |
|---|---|
| `dark` | 强制深色，首版默认值 |
| `light` | 强制浅色 |
| `system` | 跟随系统深浅色 |

本地保存建议：

- DataStore key：`curated_ui_theme`
- 可选值：`dark`、`light`、`system`
- 默认值：`dark`

不要把主题偏好和后端账号、PIN 解锁状态绑定。主题是设备本地偏好。

## 3. 颜色 Token

### 3.1 品牌色

| Token | Hex / 表达式 | 用途 |
|---|---:|---|
| `Primary` | `#FE628E` | 品牌主色、主按钮、选中态、链接、评分星、重点图标 |
| `OnPrimary` | `#1D0910` | 主色背景上的文字和图标 |
| `FocusRingDark` | `Primary 45%` | 深色模式焦点轮廓 |
| `FocusRingLight` | `Primary 42%` | 浅色模式焦点轮廓 |

`Primary` 在深色和浅色主题中保持一致，不做饱和度或明度分叉。

### 3.2 深色主题

| Token | Hex / 表达式 | Kotlin 建议名 | 用途 |
|---|---:|---|---|
| `Background` | `#0D0F1A` | `DarkBackground` | 应用主背景 |
| `OnBackground` | `#F8F7FB` | `DarkOnBackground` | 主文字 |
| `Card` | `#141826` | `DarkCard` | 卡片、列表容器 |
| `Surface` | `#141826` | `DarkSurface` | 常规业务表面 |
| `SurfaceElevated` | `#171B2B` | `DarkSurfaceElevated` | Dialog、BottomSheet、弹层 |
| `SurfaceMuted` | `#121827` | `DarkSurfaceMuted` | 弱化块、输入底色 |
| `Popover` | `#171B2B` | `DarkPopover` | 菜单、轻弹层 |
| `Secondary` | `#1B2234` | `DarkSecondary` | 次级按钮背景 |
| `Muted` | `#121827` | `DarkMuted` | Skeleton、弱化区域 |
| `OnMuted` | `#A2ABC2` | `DarkOnMuted` | 次级文字 |
| `Accent` | `#252D43` | `DarkAccent` | hover/pressed/selected 弱强调背景 |
| `OnAccent` | `#FFF4F7` | `DarkOnAccent` | accent 背景上的文字 |
| `Border` | `White 10%` | `DarkBorder` | 默认分割线和边框 |
| `BorderStrong` | `White 18%` | `DarkBorderStrong` | 强分割线、重要容器边框 |
| `InputBorder` | `White 14%` | `DarkInput` | 输入框边框 |

### 3.3 浅色主题

| Token | Hex / 表达式 | Kotlin 建议名 | 用途 |
|---|---:|---|---|
| `Background` | `#F4F6FC` | `LightBackground` | 应用主背景 |
| `OnBackground` | `#0F1219` | `LightOnBackground` | 主文字 |
| `Card` | `#FFFFFF` | `LightCard` | 卡片、列表容器 |
| `Surface` | `#FFFFFF` | `LightSurface` | 常规业务表面 |
| `SurfaceElevated` | `#FFFFFF` | `LightSurfaceElevated` | Dialog、BottomSheet、弹层 |
| `SurfaceMuted` | `#EBEEF5` | `LightSurfaceMuted` | 弱化块、输入底色 |
| `Popover` | `#FFFFFF` | `LightPopover` | 菜单、轻弹层 |
| `Secondary` | `#E8ECF6` | `LightSecondary` | 次级按钮背景 |
| `Muted` | `#EBEEF5` | `LightMuted` | Skeleton、弱化区域 |
| `OnMuted` | `#5A6378` | `LightOnMuted` | 次级文字 |
| `Accent` | `#DFE5F2` | `LightAccent` | pressed/selected 弱强调背景 |
| `OnAccent` | `#141A24` | `LightOnAccent` | accent 背景上的文字 |
| `Border` | `#0F1219 9%` | `LightBorder` | 默认分割线和边框 |
| `BorderStrong` | `#0F1219 16%` | `LightBorderStrong` | 强分割线、重要容器边框 |
| `InputBorder` | `#0F1219 12%` | `LightInput` | 输入框边框 |

### 3.4 状态色

| 语义 | 深色 | 浅色 | 用途 |
|---|---:|---:|---|
| `Success` | `#8FD6BF` | `#2F9E78` | 成功、可用、已完成 |
| `Warning` | `#F5B971` | `#D89A1B` | 警告、待处理、需要注意 |
| `Danger` | `#FF6F87` | `#E14B6D` | 删除、失败、破坏性操作 |
| `Info` | `#8B9CFF` | `#5B6FD4` | 信息、连接状态、说明 |
| `Destructive` | `#FF6F87` | `#E11D48` | 破坏性按钮和确认操作 |

状态色不能只靠颜色表达含义，必须配合文字、图标或状态标签。

## 4. Jetpack Compose 推荐结构

建议使用 Material 3 `ColorScheme` 承载基础颜色，再用 `CompositionLocal` 扩展 Web 里已有但 Material 3 不完整覆盖的 token，例如边框、状态色、媒体遮罩。

```kotlin
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

object CuratedPalette {
    val Primary = Color(0xFFFE628E)
    val OnPrimary = Color(0xFF1D0910)

    val DarkBackground = Color(0xFF0D0F1A)
    val DarkOnBackground = Color(0xFFF8F7FB)
    val DarkCard = Color(0xFF141826)
    val DarkSurfaceElevated = Color(0xFF171B2B)
    val DarkSurfaceMuted = Color(0xFF121827)
    val DarkSecondary = Color(0xFF1B2234)
    val DarkMuted = Color(0xFF121827)
    val DarkOnMuted = Color(0xFFA2ABC2)
    val DarkAccent = Color(0xFF252D43)
    val DarkOnAccent = Color(0xFFFFF4F7)

    val LightBackground = Color(0xFFF4F6FC)
    val LightOnBackground = Color(0xFF0F1219)
    val LightCard = Color.White
    val LightSurfaceMuted = Color(0xFFEBEEF5)
    val LightSecondary = Color(0xFFE8ECF6)
    val LightMuted = Color(0xFFEBEEF5)
    val LightOnMuted = Color(0xFF5A6378)
    val LightAccent = Color(0xFFDFE5F2)
    val LightOnAccent = Color(0xFF141A24)
}

@Immutable
data class CuratedExtraColors(
    val border: Color,
    val borderStrong: Color,
    val inputBorder: Color,
    val focusRing: Color,
    val success: Color,
    val warning: Color,
    val danger: Color,
    val info: Color,
    val mediaScrim: Color,
)

val LocalCuratedExtraColors = staticCompositionLocalOf {
    CuratedExtraColors(
        border = Color.White.copy(alpha = 0.10f),
        borderStrong = Color.White.copy(alpha = 0.18f),
        inputBorder = Color.White.copy(alpha = 0.14f),
        focusRing = CuratedPalette.Primary.copy(alpha = 0.45f),
        success = Color(0xFF8FD6BF),
        warning = Color(0xFFF5B971),
        danger = Color(0xFFFF6F87),
        info = Color(0xFF8B9CFF),
        mediaScrim = Color.Black.copy(alpha = 0.45f),
    )
}
```

Material 3 映射建议：

```kotlin
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme

val CuratedDarkColorScheme = darkColorScheme(
    primary = CuratedPalette.Primary,
    onPrimary = CuratedPalette.OnPrimary,
    background = CuratedPalette.DarkBackground,
    onBackground = CuratedPalette.DarkOnBackground,
    surface = CuratedPalette.DarkCard,
    onSurface = CuratedPalette.DarkOnBackground,
    surfaceVariant = CuratedPalette.DarkSurfaceMuted,
    onSurfaceVariant = CuratedPalette.DarkOnMuted,
    secondary = CuratedPalette.DarkSecondary,
    onSecondary = CuratedPalette.DarkOnBackground,
    tertiary = Color(0xFF8B9CFF),
    error = Color(0xFFFF6F87),
    onError = Color(0xFF2B0811),
)

val CuratedLightColorScheme = lightColorScheme(
    primary = CuratedPalette.Primary,
    onPrimary = CuratedPalette.OnPrimary,
    background = CuratedPalette.LightBackground,
    onBackground = CuratedPalette.LightOnBackground,
    surface = CuratedPalette.LightCard,
    onSurface = CuratedPalette.LightOnBackground,
    surfaceVariant = CuratedPalette.LightSurfaceMuted,
    onSurfaceVariant = CuratedPalette.LightOnMuted,
    secondary = CuratedPalette.LightSecondary,
    onSecondary = Color(0xFF1A1F2E),
    tertiary = Color(0xFF5B6FD4),
    error = Color(0xFFE11D48),
    onError = Color(0xFFFFFFFF),
)
```

Android 主题入口建议：

```kotlin
@Composable
fun CuratedTheme(
    preference: ThemePreference = ThemePreference.Dark,
    content: @Composable () -> Unit,
) {
    val darkTheme = when (preference) {
        ThemePreference.Dark -> true
        ThemePreference.Light -> false
        ThemePreference.System -> isSystemInDarkTheme()
    }

    val colorScheme = if (darkTheme) CuratedDarkColorScheme else CuratedLightColorScheme
    val extraColors = if (darkTheme) CuratedDarkExtraColors else CuratedLightExtraColors

    CompositionLocalProvider(LocalCuratedExtraColors provides extraColors) {
        MaterialTheme(
            colorScheme = colorScheme,
            typography = CuratedTypography,
            shapes = CuratedShapes,
            content = content,
        )
    }
}
```

## 5. 组件用色规则

### 5.1 页面背景

所有主页面根容器使用：

- 深色：`MaterialTheme.colorScheme.background`
- 浅色：`MaterialTheme.colorScheme.background`

Android 可用非常轻的顶部品牌光晕，但不能让背景变成大面积渐变主题。

推荐做法：

- 普通页面：纯 `background`。
- 首页、播放相关入口页：允许顶部 `Primary` 低透明度 radial/vertical overlay。
- 播放器页：以视频画面和黑色控制层为主，不使用普通页面背景。

### 5.2 卡片和列表

媒体卡片、演员卡片、设置卡片：

- 背景：`surface`
- 文字：`onSurface`
- 次级文字：`onSurfaceVariant` 或 `LocalCuratedExtraColors.current` 中的 muted text 映射
- 边框：`LocalCuratedExtraColors.current.border`
- 选中态：`primary` 或 `primary.copy(alpha = 0.12f)`

不要在卡片里使用大面积 `Primary` 背景。`Primary` 应只用于主操作、选中、强调点。

### 5.3 输入框

输入框需要保持可见边界，尤其是在深色 Dialog 和浅色 Card 上：

- 容器：`surfaceVariant` 或 `SurfaceMuted`
- 边框：`inputBorder`
- 聚焦边框：`primary`
- 光晕：`focusRing`
- 占位符：次级文字色
- 错误：`error` + 错误文案

禁止只有透明背景加极淡边框的主输入框。

### 5.4 按钮

| 类型 | 背景 | 文字 | 用途 |
|---|---|---|---|
| Primary | `primary` | `onPrimary` | 单屏唯一主操作 |
| Secondary | `secondary` | `onSecondary` | 次级操作 |
| Ghost/Text | transparent | `primary` 或 `onSurface` | 轻量操作、链接 |
| Destructive | `error` / `danger` | 对应 `onError` | 删除、清空、不可逆 |

同一屏最多一个视觉重量最高的 primary CTA。批量操作条可有一个 primary 和多个 secondary/ghost。

### 5.5 标签、Chip、选中态

默认标签：

- 背景：`surfaceVariant`
- 文字：`onSurfaceVariant`
- 边框：`border`

选中标签：

- 背景：`primary`
- 文字：`onPrimary`
- 不建议再加高饱和描边。

弱选中或筛选预览：

- 背景：`primary.copy(alpha = 0.10f)` 到 `0.14f`
- 文字：`primary`
- 边框：`primary.copy(alpha = 0.25f)` 到 `0.35f`

### 5.6 媒体封面和播放器

媒体区域允许使用局部专用颜色：

- 图片占位背景：深色用 `#121827`，浅色用 `#EBEEF5`
- 图片上文字遮罩：`Color.Black.copy(alpha = 0.45f)` 到 `0.65f`
- 播放器 HUD：以黑色半透明为主，文字用白色或白色透明度
- 关键播放动作可以使用 `primary`

播放器控制层的黑色遮罩是例外区域，不应反向扩散到普通业务页面。

## 6. 圆角、间距和阴影

Web 当前基础圆角为 `0.625rem`，Android 建议映射：

| Token | dp | 用途 |
|---|---:|---|
| `RadiusSmall` | `6.dp` | 小标签、内嵌状态 |
| `RadiusMedium` | `8.dp` | 按钮、输入框 |
| `RadiusLarge` | `10.dp` | 普通卡片 |
| `RadiusXLarge` | `14.dp` | Dialog、底部 Sheet、媒体大卡 |

间距使用 4/8dp 节奏：

| Token | dp |
|---|---:|
| `SpaceXs` | `4.dp` |
| `SpaceSm` | `8.dp` |
| `SpaceMd` | `12.dp` |
| `SpaceLg` | `16.dp` |
| `SpaceXl` | `24.dp` |
| `Space2Xl` | `32.dp` |

触控目标：

- Android 最小点击区域：`48.dp x 48.dp`
- 小图标视觉尺寸可以是 `20.dp` 或 `24.dp`，但 hit area 必须扩展到 `48.dp`
- 列表行、卡片主点击区必须有明确 pressed state

阴影策略：

- 深色主题优先用边框、层级色和轻微 elevation，不依赖重阴影。
- 浅色主题可使用低透明度阴影，但不要使用浓重浮层阴影。
- BottomSheet/Dialog 使用 `SurfaceElevated` + scrim，而不是靠大阴影强行分层。

## 7. 字体和排版

Web 使用 Outfit 作为 Curated 品牌字标字体。Android 可选策略：

1. 如果能合法随 App 打包 Outfit：品牌字标、关键标题使用 Outfit。
2. 如果不打包字体：正文和标题使用系统默认 sans-serif，保持平台原生体验。

建议字号：

| 角色 | sp | 权重 |
|---|---:|---|
| `TitleLarge` | `22.sp` | `SemiBold` |
| `TitleMedium` | `18.sp` | `SemiBold` |
| `BodyLarge` | `16.sp` | `Normal` |
| `BodyMedium` | `14.sp` | `Normal` |
| `LabelMedium` | `12.sp` | `Medium` |
| `MetaSmall` | `11.sp` | `Medium` |

正文不低于 `14.sp`，主要阅读文本建议 `16.sp`。必须支持系统字体缩放，不能固定截断长文本。

## 8. 可访问性和对比度

最低要求：

- 正文对比度：至少 4.5:1。
- 次级文字：至少 3:1，重要次级信息仍建议 4.5:1。
- 图标按钮必须有 `contentDescription`，纯装饰图标使用 `contentDescription = null`。
- 状态不能只靠颜色表达，例如失败状态需要红色 + 错误文字。
- 所有交互控件保留 focus/pressed/disabled 状态。

深色模式不要把次级文字压得过暗；当前标准次级文字是 `#A2ABC2`，不要随意换成更低对比的灰色。

## 9. Kotlin 命名规范

推荐文件组织：

```text
ui/theme/
  Color.kt
  Theme.kt
  Type.kt
  Shape.kt
  Spacing.kt
```

推荐命名：

- `CuratedPalette`：原始品牌色和基础色。
- `CuratedDarkColorScheme` / `CuratedLightColorScheme`：Material 3 映射。
- `CuratedExtraColors`：Material 3 不覆盖的扩展 token。
- `LocalCuratedExtraColors`：Compose 读取扩展 token 的入口。
- `CuratedTheme`：全局主题函数。
- `CuratedSpacing`：统一间距 token。
- `CuratedShapes`：统一圆角 token。

组件里允许这样用：

```kotlin
val extra = LocalCuratedExtraColors.current

Card(
    colors = CardDefaults.cardColors(
        containerColor = MaterialTheme.colorScheme.surface,
        contentColor = MaterialTheme.colorScheme.onSurface,
    ),
    border = BorderStroke(1.dp, extra.border),
) {
    // content
}
```

组件里禁止这样用：

```kotlin
Text(color = Color(0xFFFE628E))
Box(Modifier.background(Color(0xFF0D0F1A)))
```

除非是在 `ui/theme/Color.kt` 中定义 token。

## 10. Do / Don't

Do：

- 用语义 token 表达 UI 角色，而不是直接表达颜色。
- 优先适配深色，因为 Curated 默认深色。
- 同时维护浅色，确保设置页切换主题后可读。
- 媒体内容优先，普通 UI 控制保持克制。
- 选中、聚焦、错误、禁用状态都要有明确视觉差异。

Don't：

- 不要在业务 Composable 里直接写 hex。
- 不要新增紫蓝渐变、棕橙、纯黑红等新的主视觉体系。
- 不要把 `Primary` 用成大片背景色。
- 不要用透明输入框加极淡边框。
- 不要只为深色写状态色，然后让浅色自动沿用。
- 不要用颜色作为唯一信息来源。

## 11. 验收清单

开发一个 Android 页面或组件前后都应检查：

- 使用 `CuratedTheme` 包裹。
- 页面根背景来自 `MaterialTheme.colorScheme.background`。
- 卡片、Dialog、Sheet 使用 `surface` / `SurfaceElevated`。
- 主操作使用 `primary`，且同屏不滥用。
- 文本颜色来自 `onBackground`、`onSurface`、`onSurfaceVariant` 或扩展 token。
- 边框来自 `LocalCuratedExtraColors.current.border`。
- 输入框有清晰背景、边框、聚焦、错误状态。
- Touch target 不小于 `48.dp`。
- 深色和浅色都检查过。
- 大字号和系统字体缩放下不重叠、不截断关键操作。
