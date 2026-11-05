// gofmt -l 自身总是返回 0，不能直接当检查用：这里有输出时打印文件列表并以退出码 1 结束。
import { spawnSync } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const backend = join(dirname(fileURLToPath(import.meta.url)), '..', 'backend')
const r = spawnSync('gofmt', ['-l', '.'], { cwd: backend, encoding: 'utf8' })

if (r.error) {
  console.error(`无法运行 gofmt：${r.error.message}（需要安装 Go 并加入 PATH）`)
  process.exit(1)
}
if (r.status !== 0) {
  console.error(r.stderr)
  process.exit(r.status ?? 1)
}

const files = r.stdout.split(/\r?\n/).filter(Boolean)
if (files.length > 0) {
  console.error('以下文件未经 gofmt 格式化：')
  for (const f of files) console.error(`  backend/${f}`)
  process.exit(1)
}
console.log('gofmt：全部文件格式正确')
