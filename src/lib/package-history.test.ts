/// <reference types="node" />

import { describe, expect, it } from "vitest"

import {
  buildPackageHistoryChangeSummary,
  extractPreviousPackageHistoryCommit,
} from "../../scripts/release/package-history.mjs"

describe("package history helper", () => {
  it("extracts the previous commit from the latest history row", () => {
    const markdown = [
      "# 整包打包历史",
      "",
      "| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |",
      "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
      "| 2026-04-15 | 1.2.3 | `89131f08` / `master` | release:publish | release/portable/x.zip | 成功 | wujiahui | 历史记录补齐前未采集 | note |",
      "| 2026-04-16 | 1.2.4 | `7a7b2981` / `master` | release:publish | release/portable/y.zip | 成功 | wujiahui | 历史记录补齐前未采集 | note |",
    ].join("\n")

    expect(extractPreviousPackageHistoryCommit(markdown)).toBe("7a7b2981")
  })

  it("extracts the previous commit from a latest history row without backticks", () => {
    const markdown = [
      "# 整包打包历史",
      "",
      "| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |",
      "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
      "| 2026-04-16 | 1.2.4 | 7a7b2981 / master | release:publish | release/portable/y.zip | 成功 | wujiahui | 历史记录补齐前未采集 | note |",
    ].join("\n")

    expect(extractPreviousPackageHistoryCommit(markdown)).toBe("7a7b2981")
  })

  it("returns a first-record message when the history has no prior rows", async () => {
    const markdown = [
      "# 整包打包历史",
      "",
      "| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |",
      "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
    ].join("\n")

    const summary = await buildPackageHistoryChangeSummary({
      historyMarkdown: markdown,
      currentCommit: "abc1234",
      resolveCommit: async () => "abc1234",
      loadGitLog: async () => {
        throw new Error("should not call git log when no previous row exists")
      },
    })

    expect(summary).toBe("首条打包记录，无上一包可比对")
  })

  it("returns a no-diff message when the previous and current commits match", async () => {
    const markdown = [
      "# 整包打包历史",
      "",
      "| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |",
      "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
      "| 2026-04-16 | 1.2.4 | `7a7b2981` / `master` | release:publish | release/portable/y.zip | 成功 | wujiahui | 历史记录补齐前未采集 | note |",
    ].join("\n")

    const summary = await buildPackageHistoryChangeSummary({
      historyMarkdown: markdown,
      currentCommit: "7a7b2981",
      resolveCommit: async (commit) => commit,
      loadGitLog: async () => "",
    })

    expect(summary).toBe("无代码差异（同一提交重复打包）")
  })

  it("returns a current-commit parse message when the current commit cannot be resolved", async () => {
    const markdown = [
      "# 整包打包历史",
      "",
      "| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |",
      "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
      "| 2026-04-16 | 1.2.4 | `7a7b2981` / `master` | release:publish | release/portable/y.zip | 成功 | wujiahui | 历史记录补齐前未采集 | note |",
    ].join("\n")

    const summary = await buildPackageHistoryChangeSummary({
      historyMarkdown: markdown,
      currentCommit: "missing",
      resolveCommit: async (commit) => {
        if (commit === "missing") {
          return ""
        }
        return commit
      },
      loadGitLog: async () => {
        throw new Error("should not call git log when current commit cannot resolve")
      },
    })

    expect(summary).toBe("无法解析当前打包记录对应 commit")
  })

  it("joins raw git log lines with html breaks", async () => {
    const markdown = [
      "# 整包打包历史",
      "",
      "| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |",
      "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
      "| 2026-04-15 | 1.2.3 | `89131f08` / `master` | release:publish | release/portable/x.zip | 成功 | wujiahui | 历史记录补齐前未采集 | note |",
    ].join("\n")

    const summary = await buildPackageHistoryChangeSummary({
      historyMarkdown: markdown,
      currentCommit: "7a7b2981",
      resolveCommit: async (commit) => commit,
      loadGitLog: async () => "abc1234 feat: add package history column\ndef5678 fix: escape markdown cells",
    })

    expect(summary).toBe(
      "abc1234 feat: add package history column<br>def5678 fix: escape markdown cells",
    )
  })
})
