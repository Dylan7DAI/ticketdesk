// 首次安装时生成 .env：不存在才从 .env.example 复制，已存在则不覆盖。
// 用 Node 而不是 cp，保证 cmd、PowerShell、bash 下行为一致。
import { copyFileSync, existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const target = join(root, '.env')

if (existsSync(target)) {
  console.log('.env 已存在，保持不变')
} else {
  copyFileSync(join(root, '.env.example'), target)
  console.log('已从 .env.example 生成 .env')
}
