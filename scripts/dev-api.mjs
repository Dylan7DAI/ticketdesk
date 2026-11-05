// 启动后端：先 go build 到 backend/bin/，再运行编译产物，收到中断信号时结束子进程。
// 不直接用 go run：Windows 上 go run 被中断后，编译出的临时 exe 有时仍在运行，
// 下次启动会报端口 8080 被占用。
import { spawn, spawnSync } from 'node:child_process'
import { mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const backend = join(dirname(fileURLToPath(import.meta.url)), '..', 'backend')
const exe = join(backend, 'bin', process.platform === 'win32' ? 'server.exe' : 'server')

mkdirSync(join(backend, 'bin'), { recursive: true })

const build = spawnSync('go', ['build', '-o', exe, './cmd/server'], {
  cwd: backend,
  stdio: 'inherit',
})
if (build.error) {
  console.error(`无法运行 go：${build.error.message}（需要安装 Go 1.22+ 并加入 PATH）`)
  process.exit(1)
}
if (build.status !== 0) process.exit(build.status ?? 1)

// cwd 设为 backend/，与 go -C backend 一致，main.go 会读取 ../.env。
const child = spawn(exe, [], { cwd: backend, stdio: 'inherit' })

let stopping = false
function stop() {
  if (stopping) return
  stopping = true
  child.kill()
}
process.on('SIGINT', stop)
process.on('SIGTERM', stop)
process.on('exit', stop)

child.on('exit', (code, signal) => {
  process.exit(stopping ? 0 : (code ?? (signal ? 1 : 0)))
})
