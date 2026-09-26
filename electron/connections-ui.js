/* Local Desktop connection surface; no remote scripts or HTML interpolation. */
const api = window.curatedConnections
const list = document.querySelector('#servers')
const status = document.querySelector('#status')
const form = document.querySelector('#server-form')
const nameInput = document.querySelector('#name')
const urlInput = document.querySelector('#url')
const cancelEdit = document.querySelector('#cancel-edit')
let editingId
let busy = false
let lastSnapshot = ''

function message(text, error = false) {
  status.textContent = text
  status.classList.toggle('error', error)
}
function resetForm() {
  editingId = undefined
  form.reset()
  document.querySelector('#form-title').textContent = '添加服务器'
  cancelEdit.hidden = true
}
function button(label, action, disabled = false) {
  const el = document.createElement('button')
  el.type = 'button'
  el.textContent = label
  el.disabled = disabled
  el.addEventListener('click', action)
  return el
}
async function run(operation, pending, success) {
  if (busy) return
  busy = true
  message(pending)
  document.querySelectorAll('button, input').forEach(el => { el.disabled = true })
  try {
    const result = await operation()
    if (!result.ok) throw new Error(result.error)
    message(success)
    return true
  } catch (error) {
    message(error.message || '操作失败，请重试。', true)
    return false
  } finally {
    busy = false
    document.querySelectorAll('button, input').forEach(el => { el.disabled = false })
    lastSnapshot = ''
    await refresh()
  }
}
async function refresh() {
  if (busy) return
  try {
    const result = await api.list()
    const snapshot = JSON.stringify(result)
    if (snapshot === lastSnapshot) return
    lastSnapshot = snapshot
    if (result.error) message(result.error, true)
    else if (result.connecting) message('正在连接服务器…')
    document.querySelectorAll('input, form button').forEach(el => { el.disabled = result.connecting })
    list.replaceChildren()
    document.querySelector('#empty').hidden = result.state.servers.length > 0
    for (const server of result.state.servers) {
      const current = server.url === result.currentServerUrl
      const item = document.createElement('li')
      const copy = document.createElement('div')
      copy.className = 'server-copy'
      const name = document.createElement('span')
      name.className = 'server-name'
      name.textContent = server.name
      if (current) {
        const badge = document.createElement('span')
        badge.className = 'badge'; badge.textContent = '当前服务器'; name.append(badge)
      }
      const url = document.createElement('span')
      url.className = 'server-url'; url.textContent = server.url
      copy.append(name, url)
      const actions = document.createElement('div')
      actions.className = 'actions'
      actions.append(
        button(current ? '重新连接' : '连接', () => run(() => api.connect(server.id), `正在连接 ${server.name}…`, '连接完成。'), result.connecting),
        button('编辑', () => {
          editingId = server.id; nameInput.value = server.name; urlInput.value = server.url
          document.querySelector('#form-title').textContent = '编辑服务器'; cancelEdit.hidden = false; nameInput.focus()
        }, result.connecting),
        button('删除', async () => {
          const dialog = document.querySelector('#delete-dialog')
          dialog.returnValue = 'cancel'
          dialog.showModal()
          dialog.addEventListener('close', async () => {
            if (dialog.returnValue !== 'delete') return
            if (await run(() => api.remove(server.id), '正在删除…', '连接已删除。')) {
              if (editingId === server.id) resetForm()
            }
          }, { once: true })
        }, current || result.connecting),
      )
      item.append(copy, actions); list.append(item)
    }
  } catch { message('无法读取服务器列表，请重新打开窗口。', true) }
}
form.addEventListener('submit', async event => {
  event.preventDefault()
  const input = { name: nameInput.value, url: urlInput.value, ...(editingId ? { id: editingId } : {}) }
  if (await run(() => api.save(input), '正在保存…', '服务器已保存。')) resetForm()
})
cancelEdit.addEventListener('click', resetForm)
void refresh()
setInterval(refresh, 1000)
