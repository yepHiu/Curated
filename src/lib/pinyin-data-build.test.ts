import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { describe, expect, it } from 'vitest'
import { extractPinyinDictionaries } from '../../vite.pinyin-data'

/** 为提取测试提供完整、无可执行表达式的最小词典模块。 */
function fixture(map = "{ zhōng: ['中'], guó: ['国'] }") {
  return `const map = ${map}; const DICT2 = { 中国: 'zhōng guó' }; const DICT3 = {}; const DICT4 = {}; const DICT5 = {}; export { map, DICT2 };`
}

describe('pinyin dictionary data extraction', () => {
  /** 词条及数组顺序保持不变，算法主体仍留在原模块。 */
  it('preserves dictionary content and executable exports', () => {
    const result = extractPinyinDictionaries(fixture())
    expect(result.dictionaries).toEqual({ map: { zhōng: ['中'], guó: ['国'] }, DICT2: { 中国: 'zhōng guó' }, DICT3: {}, DICT4: {}, DICT5: {} })
    expect(result.code).toContain('const map = __curatedPinyinData.map;')
    expect(result.code).toContain('export { map, DICT2 };')
  })

  /** 上游格式变化必须显式报错，避免静默丢失词条。 */
  it('rejects executable values and missing dictionaries', () => {
    expect(() => extractPinyinDictionaries(fixture('{ key: run() }'))).toThrow('Unsupported')
    expect(() => extractPinyinDictionaries('const map = {};')).toThrow('format changed')
  })

  /** 对锁文件实际安装的完整词典执行转换，验证常见及多音字数据存在。 */
  it('extracts the installed pinyin-pro dictionaries', () => {
    const require = createRequire(import.meta.url)
    const source = readFileSync(require.resolve('pinyin-pro/dist/index.mjs'), 'utf8')
    const { dictionaries, code } = extractPinyinDictionaries(source)
    expect(Object.keys(dictionaries)).toEqual(['map', 'DICT2', 'DICT3', 'DICT4', 'DICT5'])
    expect((dictionaries.DICT2 as Record<string, string>)['这个']).toBe('zhè ge')
    expect(Buffer.byteLength(source) - Buffer.byteLength(code)).toBeGreaterThan(200_000)
  })
})
