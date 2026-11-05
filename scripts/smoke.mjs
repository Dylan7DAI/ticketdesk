// 接口演示脚本：按 I3 流程走一遍，再演示一次 403 和 409。
// 用 Node 20 内置的 fetch，无需安装依赖。需要后端已启动（npm run dev 或 npm run dev:api）。
// 用法：npm run smoke            （默认 http://127.0.0.1:8080）
//       BASE=http://127.0.0.1:5173 npm run smoke   （经 Vite 代理）
const BASE = process.env.BASE ?? 'http://127.0.0.1:8080'

// 演示用户（WF§1）：1 主管王敏，2 李雷，3 韩梅
const SUP = 1
const LILEI = 2
const HANMEI = 3

async function call(user, method, path, body) {
  const res = await fetch(BASE + path, {
    method,
    headers: { 'X-User-ID': String(user), 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  return { status: res.status, json: await res.json() }
}

function show(label, r) {
  if (r.json.error) {
    console.log(`  ${label.padEnd(26)} -> ${r.status} ${r.json.error.code}：${r.json.error.message}`)
  } else if (r.json.status) {
    const who = r.json.assignee ? r.json.assignee.name : '无'
    console.log(`  ${label.padEnd(26)} -> ${r.status} 状态=${r.json.status.padEnd(11)} 处理人=${who}`)
  } else {
    console.log(`  ${label.padEnd(26)} -> ${r.status}`)
  }
  return r
}

function expect(r, status, code) {
  const got = r.json.error?.code
  if (r.status !== status || (code && got !== code)) {
    console.error(`\n期望 ${status} ${code ?? ''}，实际 ${r.status} ${got ?? ''}`)
    process.exit(1)
  }
  return r.json
}

async function main() {
  const health = await fetch(BASE + '/api/health').catch(() => null)
  if (!health?.ok) {
    console.error(`连不上 ${BASE}：先运行 npm run dev`)
    process.exit(1)
  }

  console.log('== 生命周期（I3）==')
  const created = expect(
    show('主管 创建工单', await call(SUP, 'POST', '/api/tickets', {
      title: '冒烟测试：支付失败', description: '用户无法下单', customer_name: '测试客户',
      customer_contact: 'smoke@example.com', priority: 'high',
    })), 201)
  const p = `/api/tickets/${created.id}`
  expect(show('李雷 领取', await call(LILEI, 'POST', `${p}/claim`)), 200)
  expect(show('李雷 评论', await call(LILEI, 'POST', `${p}/comments`, { body: '已联系客户' })), 201)
  expect(show('李雷 解决', await call(LILEI, 'POST', `${p}/resolve`, { note: '已处理' })), 200)
  expect(show('李雷 重开', await call(LILEI, 'POST', `${p}/reopen`)), 200)
  expect(show('主管 改派给韩梅', await call(SUP, 'POST', `${p}/reassign`, { assignee_id: HANMEI })), 200)
  expect(show('韩梅 解决', await call(HANMEI, 'POST', `${p}/resolve`)), 200)
  expect(show('主管 关闭', await call(SUP, 'POST', `${p}/close`)), 200)

  console.log('\n== 权限拒绝（403）==')
  expect(show('李雷 分配（agent 无权）', await call(LILEI, 'POST', `${p}/assign`, { assignee_id: LILEI })), 403, 'FORBIDDEN')
  // 详情对所有客服只读可见，但不能操作他人的工单。
  expect(show('韩梅 读李雷的工单', await call(HANMEI, 'GET', '/api/tickets/3')), 200)
  expect(show('韩梅 解决李雷的工单', await call(HANMEI, 'POST', '/api/tickets/3/resolve')), 403, 'NOT_ASSIGNEE')

  console.log('\n== 非法流转（409）==')
  expect(show('主管 重开已关闭工单', await call(SUP, 'POST', `${p}/reopen`)), 409, 'INVALID_TRANSITION')
  expect(show('主管 评论已关闭工单', await call(SUP, 'POST', `${p}/comments`, { body: 'x' })), 409, 'INVALID_STATE')

  const detail = expect(await call(SUP, 'GET', p), 200)
  console.log(`\n历史（${detail.events.length} 条）：`)
  for (const e of detail.events) console.log(`  - ${e.summary}`)
  console.log('\n冒烟测试通过')
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
