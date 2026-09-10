import ts from 'typescript'
import type { Plugin } from 'vite'

const DICTIONARY_NAMES = new Set(['map', 'DICT2', 'DICT3', 'DICT4', 'DICT5'])

/** 仅转换词典中的字符串、数组和普通对象，遇到可执行表达式即拒绝构建。 */
function readLiteral(node: ts.Expression): unknown {
  if (ts.isStringLiteral(node)) return node.text
  if (ts.isArrayLiteralExpression(node)) return node.elements.map(readLiteral)
  if (ts.isObjectLiteralExpression(node)) {
    const result: Record<string, unknown> = {}
    for (const property of node.properties) {
      if (!ts.isPropertyAssignment(property) || !property.name ||
          (!ts.isIdentifier(property.name) && !ts.isStringLiteral(property.name))) {
        throw new Error('Unsupported pinyin dictionary property')
      }
      result[property.name.text] = readLiteral(property.initializer)
    }
    return result
  }
  throw new Error('Unsupported pinyin dictionary value')
}

/** 从锁定版本的拼音模块提取纯词典；不改变匹配算法或词条。 */
export function extractPinyinDictionaries(source: string) {
  const file = ts.createSourceFile('pinyin.mjs', source, ts.ScriptTarget.Latest, true, ts.ScriptKind.JS)
  const dictionaries: Record<string, unknown> = {}
  const replacements: { start: number; end: number; text: string }[] = []
  for (const statement of file.statements) {
    if (!ts.isVariableStatement(statement)) continue
    for (const declaration of statement.declarationList.declarations) {
      if (!ts.isIdentifier(declaration.name) || !DICTIONARY_NAMES.has(declaration.name.text)) continue
      if (!declaration.initializer) throw new Error('Missing pinyin dictionary initializer')
      const name = declaration.name.text
      dictionaries[name] = readLiteral(declaration.initializer)
      replacements.push({ start: declaration.initializer.getStart(file), end: declaration.initializer.end, text: `__curatedPinyinData.${name}` })
    }
  }
  if (Object.keys(dictionaries).length !== DICTIONARY_NAMES.size) {
    throw new Error('Pinyin dictionary format changed; review the data extraction before upgrading')
  }
  let code = source
  for (const replacement of replacements.reverse()) {
    code = code.slice(0, replacement.start) + replacement.text + code.slice(replacement.end)
  }
  return { code, dictionaries }
}

/** 将可选拼音检索的大型纯词典作为内容哈希 JSON 加载，减少可执行 JS 解析。 */
export function pinyinDataPlugin(): Plugin {
  return {
    name: 'curated-pinyin-data',
    apply: 'build',
    /** 仅处理 pinyin-pro 的 ESM 入口，开发与单测保留原包行为。 */
    transform(source, id) {
      if (!id.replaceAll('\\', '/').endsWith('/pinyin-pro/dist/index.mjs')) return
      const { code, dictionaries } = extractPinyinDictionaries(source)
      const reference = this.emitFile({ type: 'asset', name: 'pinyin-dictionaries.json', source: JSON.stringify(dictionaries) })
      return {
        code: `const __curatedPinyinResponse = await fetch(import.meta.ROLLUP_FILE_URL_${reference});
if (!__curatedPinyinResponse.ok) throw new Error('Unable to load pinyin dictionary');
const __curatedPinyinData = await __curatedPinyinResponse.json();
${code}`,
        map: null,
      }
    },
  }
}
