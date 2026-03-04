#!/usr/bin/env node
/* eslint-disable no-console */

const fs = require('node:fs')
const net = require('node:net')
const path = require('node:path')
const { spawn } = require('node:child_process')

function parseArgs(argv) {
  const args = { app: undefined, preferred: undefined, range: 50, dryRun: false, passthrough: [] }

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index]

    if (token === '--') {
      args.passthrough = argv.slice(index + 1)
      break
    }
    if (token === '--help' || token === '-h') {
      args.help = true
      break
    }
    if (token === '--dry-run') {
      args.dryRun = true
      continue
    }

    const next = argv[index + 1]
    if (token === '--app') {
      args.app = next
      index += 1
      continue
    }
    if (token === '--preferred') {
      args.preferred = next
      index += 1
      continue
    }
    if (token === '--range') {
      args.range = Number(next)
      index += 1
      continue
    }

    throw new Error(`未知参数: ${token}`)
  }

  return args
}

function printHelp() {
  console.log(
    [
      '用法: node ../scripts/next-dev.js --app <user|admin> [--range <n>] [--dry-run] [-- <next 参数...>]',
      '  --app        选择前端类型，用于读取根目录 .env 中的端口变量',
      '  --preferred  显式指定首选端口（优先级最高）',
      '  --range      向上搜索可用端口的范围（默认 50）',
      '  --dry-run    仅输出最终端口，不启动 Next',
      '  --           后面的参数会透传给 next dev',
    ].join('\n')
  )
}

function parseDotEnv(content) {
  const env = {}
  const lines = content.split(/\r?\n/)

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) continue
    const equalIndex = line.indexOf('=')
    if (equalIndex < 0) continue

    const key = line.slice(0, equalIndex).trim()
    let value = line.slice(equalIndex + 1).trim()
    if (!key) continue

    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1)
    }

    env[key] = value
  }

  return env
}

function readRepoRootEnv() {
  const repoRoot = path.resolve(__dirname, '..')
  const envPath = path.join(repoRoot, '.env')
  if (!fs.existsSync(envPath)) return {}

  try {
    const content = fs.readFileSync(envPath, 'utf8')
    return parseDotEnv(content)
  } catch (error) {
    console.warn(`[警告] 读取根目录 .env 失败，将忽略: ${error.message}`)
    return {}
  }
}

function toPortNumber(value) {
  const port = Number(value)
  if (!Number.isInteger(port) || port <= 0 || port > 65535) return undefined
  return port
}

function canListen(port) {
  return new Promise((resolve) => {
    const server = net.createServer()

    server.unref()
    server.once('error', () => resolve(false))
    server.listen(port, () => server.close(() => resolve(true)))
  })
}

async function findAvailablePort(preferredPort, range) {
  for (let port = preferredPort; port <= preferredPort + range; port += 1) {
    // eslint-disable-next-line no-await-in-loop
    if (await canListen(port)) return port
  }
  return undefined
}

function resolveNextCli() {
  try {
    return require.resolve('next/dist/bin/next', { paths: [process.cwd()] })
  } catch (error) {
    throw new Error(`未找到 next，可先在当前目录执行 npm install。原始错误: ${error.message}`)
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  if (args.help) {
    printHelp()
    return
  }

  const rootEnv = readRepoRootEnv()
  const preferredFromArgs = toPortNumber(args.preferred)

  let preferredPort
  if (preferredFromArgs) {
    preferredPort = preferredFromArgs
  } else if (args.app === 'admin') {
    preferredPort = toPortNumber(process.env.FRONTEND_ADMIN_PORT) ?? toPortNumber(rootEnv.FRONTEND_ADMIN_PORT) ?? 3001
  } else if (args.app === 'user') {
    preferredPort = toPortNumber(process.env.FRONTEND_USER_PORT) ?? toPortNumber(rootEnv.FRONTEND_USER_PORT) ?? 3000
  } else {
    throw new Error('必须指定 --app <user|admin> 或 --preferred <port>')
  }

  const range = Number.isFinite(args.range) && args.range >= 0 ? args.range : 50
  const port = await findAvailablePort(preferredPort, range)
  if (!port) {
    throw new Error(`未找到可用端口：起始 ${preferredPort}，范围 ${range}`)
  }

  if (port !== preferredPort) {
    console.warn(`[警告] 端口 ${preferredPort} 已被占用，已自动切换为 ${port}`)
  } else {
    console.log(`[信息] 使用端口 ${port}`)
  }

  if (args.dryRun) return

  const nextCli = resolveNextCli()
  const nextArgs = ['dev', '-p', String(port), ...args.passthrough]
  const child = spawn(process.execPath, [nextCli, ...nextArgs], {
    stdio: 'inherit',
    env: { ...process.env, PORT: String(port) },
  })

  child.on('exit', (code) => process.exit(code ?? 0))
}

main().catch((error) => {
  console.error(`[错误] ${error.message}`)
  process.exit(1)
})
