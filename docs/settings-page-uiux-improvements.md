# 设置页面 UI/UX 改进建议

## 一、颜色统一性问题

### 1.1 背景色层级混乱

**位置**: `SettingsPage.vue` (多处)

**现状**:
当前设置页面使用了过多的背景色变体，导致视觉层级混乱：

```vue
<!-- 外层卡片 -->
<Card class="... bg-card/85 ...">

<!-- 内层统计卡片 -->
<Card class="... bg-background/40 ...">

<!-- 表单区域 -->
<div class="... bg-muted/15 ...">

<!-- 图标背景 -->
<span class="... bg-muted/35 ...">
```

**问题**:
1. `bg-card/85` (85%不透明度的卡片色) 与 `bg-background/40` 混合，透明度叠加后颜色难以预测
2. `bg-muted/15` 和 `bg-muted/35` 的微妙差异用户几乎无法察觉，却增加了代码复杂度
3. 多层嵌套卡片形成"卡片中的卡片"，每层都有不同的背景色

**改进建议**:

建立三层颜色体系：

```vue
<!-- 1. 页面层：使用纯净的 bg-background -->
<div class="min-h-screen bg-background">

<!-- 2. 卡片层：使用纯净的 bg-card，不使用透明度 -->
<Card class="bg-card border-border">

  <!-- 3. 表单/内容层：使用 bg-muted 作为 subtly 区分，不使用透明度 -->
  <div class="bg-muted/10 rounded-lg p-4">
```

**具体修改**:

```diff
- <Card class="gap-3 rounded-3xl border-border/70 bg-card/85 shadow-sm shadow-black/5">
+ <Card class="rounded-2xl border border-border bg-card">

- <Card class="rounded-2xl border-border/60 bg-background/40 shadow-none">
+ <Card class="rounded-xl border border-border/50 bg-muted/5">

- <div class="... bg-muted/15 ...">
+ <div class="... bg-muted/5 ...">
```

**优先级**: 🔴 高

---

### 1.2 边框颜色不统一

**位置**: `SettingsPage.vue` (全局)

**现状**:
边框透明度混用：`border-border/70`, `border-border/60`, `border-transparent`

**问题**:
1. 相邻元素使用不同的边框透明度，产生视觉"颤动"
2. 深色模式下低透明度边框几乎看不见，导致区域边界模糊

**改进建议**:

统一使用两种边框强度：

```css
/* 主要边界 - 用于卡片、大区域 */
.border-border { /* 100% 透明度 */ }

/* 次要边界 - 用于内部分组、分割线 */
.border-border/50 { /* 50% 透明度 */ }
```

**具体修改**:

```diff
- class="rounded-2xl border-border/70 bg-background/50"
+ class="rounded-xl border border-border bg-muted/5"

- class="border border-border/60"
+ class="border border-border/50"
```

**优先级**: 🔴 高

---

### 1.3 圆角层级不统一

**位置**: `SettingsPage.vue` (多处)

**现状**:
- 外层卡片: `rounded-3xl`
- 中层卡片: `rounded-2xl`
- 内层元素: `rounded-xl` 或 `rounded-lg`
- 按钮/输入框: `rounded-xl` 或 `rounded-2xl`

**问题**:
圆角层级没有明确的视觉逻辑，同一层级使用不同圆角值

**改进建议**:

建立圆角层级系统：

| 层级 | 圆角值 | 使用场景 |
|------|--------|----------|
| 页面级 | `rounded-none` | 全屏容器 |
| 卡片级 | `rounded-xl` (16px) | 设置卡片 |
| 组件级 | `rounded-lg` (12px) | 按钮、输入框 |
| 元素级 | `rounded-md` (8px) | 标签、小按钮 |

```diff
- <Card class="rounded-3xl ...">
+ <Card class="rounded-xl ...">

- <Button class="rounded-2xl ...">
+ <Button class="rounded-lg ...">
```

**优先级**: 🟡 中

---

## 二、交互体验问题

### 2.1 侧边栏与内容区视觉不连贯

**位置**: `SettingsPage.vue:1401-1443`

**现状**:
```vue
<nav class="... lg:border-r lg:border-border/60 lg:pr-4">
  <TabsList class="... bg-transparent ...">
```

**问题**:
1. 侧边栏使用透明背景，与右侧内容区没有视觉分割
2. 选中态使用 `border-transparent` 实际上没有边框，但非选中态 hover 时有背景
3. 移动端使用 Select 下拉，桌面端使用 Tabs，两套交互模型

**改进建议**:

统一侧边栏视觉：

```vue
<nav class="bg-card border-r border-border p-4">
  <!-- 侧边栏也使用卡片背景，与内容区形成明确分割 -->

  <div class="space-y-1">
    <button
      v-for="item in items"
      :key="item.slug"
      :class="cn(
        'w-full rounded-lg px-3 py-2 text-left text-sm transition-colors',
        activeSlug === item.slug
          ? 'bg-primary/10 text-primary font-medium'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
      )"
    >
      {{ item.label }}
    </button>
  </div>
</nav>
```

**优先级**: 🔴 高

---

### 2.2 设置项分组混乱

**位置**: `SettingsPage.vue` (概览、通用、库等 Tab)

**现状**:
- 概览页使用 Dashboard 卡片网格展示统计
- 通用页混合了语言、外观、日志等多个主题
- 同一主题的内容被分散在不同卡片中

**问题**:
1. 用户难以快速找到特定设置
2. 相关设置被物理分隔
3. "概览" Tab 没有实际设置功能，只是展示统计

**改进建议**:

重新组织信息架构：

```
概览 (Overview)
├── 库统计 (Movies, Actors, Storage)
└── 最近活动 (最近扫描、最近添加)

通用 (General)
├── 语言与地区
└── 外观 (主题)

库 (Library)
├── 库路径管理
├── 扫描设置 (自动扫描、整理入库)
└── 元数据刮削

网络 (Network) - 新增
├── 代理设置
└── 连接测试

系统 (System) - 新增
├── 日志设置 (前端 + 后端)
└── 存储管理
```

**优先级**: 🟡 中

---

### 2.3 表单保存交互不一致

**位置**: `SettingsPage.vue` (代理、日志设置)

**现状**:
- 代理设置：需要手动点击保存按钮
- 语言/外观：切换后立即生效
- 日志设置：需要手动点击保存按钮

**问题**:
用户不清楚哪些设置会自动保存，哪些需要手动保存

**改进建议**:

统一为两种模式之一：

**方案 A: 全部自动保存** (推荐)
```vue
<Select v-model="proxyEnabled" @update:model-value="saveProxy">
  <!-- 切换后显示保存状态指示器 -->
</Select>

<!-- 右上角显示轻量指示器 -->
<span v-if="saving" class="text-xs text-muted-foreground">
  <Loader2 class="size-3 animate-spin inline" /> 保存中...
</span>
<span v-else-if="saved" class="text-xs text-green-500">已保存</span>
```

**方案 B: 明确的保存按钮分组**
```vue
<!-- 每个卡片底部统一放置操作栏 -->
<CardFooter class="flex justify-end gap-2 border-t border-border/50 pt-4">
  <Button variant="ghost" @click="reset">重置</Button>
  <Button @click="save" :loading="saving">保存更改</Button>
</CardFooter>
```

**优先级**: 🔴 高

---

### 2.4 图标与标题过于抢眼

**位置**: `SettingsPage.vue` 各 CardHeader

**现状**:
```vue
<CardTitle class="flex items-center gap-3 text-xl font-semibold tracking-tight">
  <span class="flex size-10 shrink-0 items-center justify-center
               rounded-2xl border border-border/60 bg-muted/35
               text-primary shadow-sm">
    <Languages class="size-[1.15rem]" />
  </span>
  {{ t("settings.navGeneral") }}
</CardTitle>
```

**问题**:
1. 每个卡片都有带边框、阴影的图标容器，视觉上过于沉重
2. 图标使用 `text-primary` (粉色)，与页面其他强调色竞争注意力
3. 图标大小与标题不协调

**改进建议**:

简化图标呈现：

```diff
- <CardTitle class="flex items-center gap-3 text-xl font-semibold tracking-tight">
-   <span class="flex size-10 shrink-0 items-center justify-center
-                rounded-2xl border border-border/60 bg-muted/35
-                text-primary shadow-sm">
-     <Languages class="size-[1.15rem]" />
-   </span>
-   {{ title }}
- </CardTitle>

+ <CardHeader class="pb-4">
+   <div class="flex items-center gap-2 text-muted-foreground mb-1">
+     <Languages class="size-4" />
+     <span class="text-xs font-medium uppercase tracking-wider">{{ category }}</span>
+   </div>
+   <CardTitle class="text-lg font-semibold">{{ title }}</CardTitle>
+ </CardHeader>
```

**效果**:
- 图标变为辅助性装饰，不抢眼
- 标题更加清晰突出
- 整体视觉更轻盈

**优先级**: 🟡 中

---

### 2.5 移动端导航体验不佳

**位置**: `SettingsPage.vue:1405-1428`

**现状**:
移动端使用 Select 下拉选择设置分类

**问题**:
1. 下拉选择需要两次点击才能切换
2. 用户无法一眼看到所有可用分类
3. 与桌面端的视觉差异过大

**改进建议**:

使用横向滚动 Tab 栏：

```vue
<!-- 移动端：横向滚动 -->
<div class="lg:hidden overflow-x-auto pb-2 -mx-4 px-4">
  <div class="flex gap-2 min-w-max">
    <button
      v-for="item in navItems"
      :key="item.slug"
      :class="cn(
        'px-4 py-2 rounded-full text-sm whitespace-nowrap transition-colors',
        active === item.slug
          ? 'bg-primary text-primary-foreground'
          : 'bg-muted text-muted-foreground'
      )"
      @click="active = item.slug"
    >
      {{ item.label }}
    </button>
  </div>
</div>
```

**优先级**: 🟡 中

---

## 三、布局与间距问题

### 3.1 卡片间距不一致

**位置**: `SettingsPage.vue` 全局

**现状**:
- 外层容器: `gap-3`
- 卡片内部: `gap-3`
- 表单元素: `gap-3`
- 所有间距都是 12px

**问题**:
没有间距层级，所有元素之间距离相同，缺乏视觉节奏

**改进建议**:

建立间距层级：

| 层级 | 间距 | 使用场景 |
|------|------|----------|
| 章节间距 | `gap-8` (32px) | 不同设置组之间 |
| 卡片间距 | `gap-6` (24px) | 相关卡片之间 |
| 内容间距 | `gap-4` (16px) | 卡片内部段落 |
| 元素间距 | `gap-3` (12px) | 表单元素之间 |
| 紧凑间距 | `gap-2` (8px) | 图标+文字等内联元素 |

```diff
- <div class="flex w-full flex-col gap-3">
+ <div class="flex w-full flex-col gap-6">

- <CardContent class="flex flex-col gap-3 pt-2">
+ <CardContent class="flex flex-col gap-4">
```

**优先级**: 🟢 低

---

### 3.2 内边距与内容密度

**位置**: `SettingsPage.vue` 各 Card

**现状**:
- CardHeader: 默认内边距
- CardContent: 默认内边距
- 额外的容器 `p-4`

**问题**:
多层内边距叠加导致内容区域实际可用空间变小

**改进建议**:

精简内边距：

```diff
- <Card class="...">
-   <CardHeader class="space-y-3 pb-2">
-   <CardContent class="flex flex-col gap-3 pt-2">
-     <div class="... p-4">

+ <Card class="...">
+   <CardHeader class="pb-4">
+   <CardContent class="space-y-4">
+     <div class="... p-4">
```

**优先级**: 🟢 低

---

## 四、具体代码改进示例

### 4.1 改进后的卡片组件结构

```vue
<template>
  <!-- 页面容器 -->
  <div class="mx-auto max-w-5xl px-4 py-6 space-y-8">

    <!-- 设置组标题 -->
    <section class="space-y-4">
      <h2 class="text-sm font-medium text-muted-foreground uppercase tracking-wider">
        {{ t('settings.groupGeneral') }}
      </h2>

      <!-- 设置卡片 -->
      <Card class="rounded-xl border border-border bg-card">
        <CardHeader class="pb-4">
          <div class="flex items-center gap-2 text-muted-foreground mb-1">
            <Languages class="size-4" />
            <span class="text-xs font-medium uppercase tracking-wider">
              {{ t('settings.categoryLanguage') }}
            </span>
          </div>
          <CardTitle class="text-lg font-semibold">
            {{ t('settings.languageTitle') }}
          </CardTitle>
          <CardDescription>
            {{ t('settings.languageDesc') }}
          </CardDescription>
        </CardHeader>

        <CardContent class="space-y-4">
          <!-- 设置项 -->
          <div class="flex items-center justify-between gap-4 p-4 rounded-lg bg-muted/5">
            <div class="space-y-1">
              <Label class="text-sm font-medium">
                {{ t('settings.language') }}
              </Label>
              <p class="text-xs text-muted-foreground">
                {{ t('settings.languageHint') }}
              </p>
            </div>
            <Select v-model="locale" class="w-44">
              <!-- ... -->
            </Select>
          </div>
        </CardContent>
      </Card>
    </section>

  </div>
</template>
```

### 4.2 改进后的侧边栏

```vue
<template>
  <aside class="w-64 bg-card border-r border-border flex flex-col">
    <!-- 侧边栏头部 -->
    <div class="p-4 border-b border-border">
      <h1 class="text-lg font-semibold">{{ t('settings.title') }}</h1>
    </div>

    <!-- 导航 -->
    <nav class="flex-1 p-3 space-y-1 overflow-y-auto">
      <button
        v-for="item in navItems"
        :key="item.slug"
        :class="cn(
          'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-all',
          activeSlug === item.slug
            ? 'bg-primary/10 text-primary font-medium'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground'
        )"
        @click="activeSlug = item.slug"
      >
        <component :is="item.icon" class="size-4" />
        {{ item.label }}
      </button>
    </nav>
  </aside>
</template>
```

---

## 五、优先级总结

| 优先级 | 问题 | 影响 |
|--------|------|------|
| 🔴 高 | 背景色层级混乱 | 视觉混乱，品牌感弱 |
| 🔴 高 | 表单保存交互不一致 | 用户困惑，易丢失设置 |
| 🔴 高 | 边框颜色不统一 | 边界模糊，可读性差 |
| 🟡 中 | 侧边栏视觉不连贯 | 导航体验差 |
| 🟡 中 | 图标过于抢眼 | 视觉重心错误 |
| 🟡 中 | 设置项分组混乱 | 信息架构混乱 |
| 🟡 中 | 圆角层级不统一 | 细节不精致 |
| 🟡 中 | 移动端导航体验 | 移动端可用性差 |
| 🟢 低 | 卡片间距不一致 | 缺乏视觉节奏 |
| 🟢 低 | 内边距叠加 | 空间利用率低 |

---

## 六、实施建议

1. **第一阶段** (1-2天)：修复颜色统一性问题
   - 统一背景色使用，移除透明度叠加
   - 统一边框颜色和圆角

2. **第二阶段** (1-2天)：改进交互体验
   - 统一保存交互模式（建议全部采用自动保存）
   - 简化侧边栏视觉

3. **第三阶段** (1天)：调整布局和间距
   - 重新组织设置项分组
   - 优化间距层级

4. **第四阶段** (可选)：移动端优化
   - 改进移动端导航体验
   - 测试触摸交互
