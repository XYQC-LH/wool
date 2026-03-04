@echo off
setlocal EnableExtensions EnableDelayedExpansion
set "ORIG_DIR=%CD%"
cd /d "%~dp0"
chcp 65001 >nul
title Nexus API Gateway - 开发环境启动 (后端 Docker + 前端 Windows)

set "VERIFY_MODE=0"
if /I "%~1"=="--verify" set "VERIFY_MODE=1"
if /I "%~1"=="/verify" set "VERIFY_MODE=1"

set "EXIT_CODE=0"

echo ============================================
echo    Nexus API Gateway - 开发环境启动
echo    后端: Docker Compose (postgres/redis/api/job-worker/resource-pool/prometheus/grafana)
echo    前端: Windows 本地 (Next.js dev)
echo ============================================
echo.

echo [步骤 1/6] 环境检查...

docker --version >nul 2>&1
if errorlevel 1 (
  echo [错误] Docker 未安装或未运行！
  echo 请先安装/启动 Docker Desktop: https://www.docker.com/products/docker-desktop
  set "EXIT_CODE=1"
  goto :die
)

docker compose version >nul 2>&1
if errorlevel 1 (
  echo [错误] Docker Compose 不可用！
  set "EXIT_CODE=1"
  goto :die
)

docker info >nul 2>&1
if errorlevel 1 (
  echo [信息] Docker 引擎未就绪，正在尝试启动 Docker Desktop...
  call :startDockerDesktop
  if errorlevel 1 (
    echo [错误] Docker 引擎仍不可用，请手动启动 Docker Desktop 后重试
    set "EXIT_CODE=1"
    goto :die
  )
)

node --version >nul 2>&1
if errorlevel 1 (
  echo [错误] Node.js 未安装！
  echo 请先安装 Node.js: https://nodejs.org/
  set "EXIT_CODE=1"
  goto :die
)

call npm.cmd --version >nul 2>&1
if errorlevel 1 (
  echo [错误] npm 不可用或无法运行！
  echo 请确认已安装 Node.js 并已将 npm 加入 PATH
  set "EXIT_CODE=1"
  goto :die
)

if not exist ".env" (
  echo [信息] 正在创建默认配置文件...
  if exist ".env.example" (
    copy ".env.example" ".env" >nul 2>&1
    if errorlevel 1 (
      echo [错误] 创建 ".env" 失败
      set "EXIT_CODE=1"
      goto :die
    )
  ) else (
    echo [错误] 未找到 ".env.example"，无法自动创建 ".env"
    set "EXIT_CODE=1"
    goto :die
  )
)

call :loadEnvPorts

echo [步骤 2/6] 清理占用端口...
echo [信息] 检查并清理端口 %FRONTEND_USER_PORT% (用户端前端)...
call :killPort %FRONTEND_USER_PORT%
echo [信息] 检查并清理端口 %FRONTEND_ADMIN_PORT% (管理员端前端)...
call :killPort %FRONTEND_ADMIN_PORT%
echo [信息] 检查并清理端口 %API_PORT% (API 服务)...
call :killPort %API_PORT%
echo [信息] 检查并清理端口 %RESOURCE_POOL_PORT% (资源池)...
call :killPort %RESOURCE_POOL_PORT%
echo [信息] 检查并清理端口 %PROMETHEUS_PORT% (Prometheus)...
call :killPort %PROMETHEUS_PORT%
echo [信息] 检查并清理端口 %GRAFANA_PORT% (Grafana)...
call :killPort %GRAFANA_PORT%
echo [信息] 端口清理完成！
echo.

echo [步骤 3/6] 选择可用端口（自动避开系统保留/排除端口）...
set "REQUESTED_FRONTEND_USER_PORT=%FRONTEND_USER_PORT%"
set "REQUESTED_FRONTEND_ADMIN_PORT=%FRONTEND_ADMIN_PORT%"
set "REQUESTED_API_PORT=%API_PORT%"
set "REQUESTED_RESOURCE_POOL_PORT=%RESOURCE_POOL_PORT%"

set "FRONTEND_USER_PORT_SELECTED="
set "FRONTEND_ADMIN_PORT_SELECTED="
set "API_PORT_SELECTED="
set "RESOURCE_POOL_PORT_SELECTED="

call :findUsablePort %REQUESTED_FRONTEND_USER_PORT% 50 FRONTEND_USER_PORT_SELECTED
call :findUsablePort %REQUESTED_FRONTEND_ADMIN_PORT% 50 FRONTEND_ADMIN_PORT_SELECTED
call :findUsablePort %REQUESTED_API_PORT% 50 API_PORT_SELECTED
call :findUsablePort %REQUESTED_RESOURCE_POOL_PORT% 50 RESOURCE_POOL_PORT_SELECTED

if defined FRONTEND_USER_PORT_SELECTED (
  if defined FRONTEND_ADMIN_PORT_SELECTED (
    if "!FRONTEND_ADMIN_PORT_SELECTED!"=="!FRONTEND_USER_PORT_SELECTED!" (
      set "FRONTEND_USER_PORT_SELECTED="
      set /a "USER_BASE=!FRONTEND_ADMIN_PORT_SELECTED!+1"
      call :findUsablePort !USER_BASE! 200 FRONTEND_USER_PORT_SELECTED
    )
  )
)
if defined API_PORT_SELECTED (
  if defined RESOURCE_POOL_PORT_SELECTED (
    if "!RESOURCE_POOL_PORT_SELECTED!"=="!API_PORT_SELECTED!" (
      set "RESOURCE_POOL_PORT_SELECTED="
      set /a "RESOURCE_POOL_BASE=!API_PORT_SELECTED!+1"
      call :findUsablePort !RESOURCE_POOL_BASE! 200 RESOURCE_POOL_PORT_SELECTED
    )
  )
)

if not defined FRONTEND_USER_PORT_SELECTED call :findUsablePort 3000 200 FRONTEND_USER_PORT_SELECTED
if not defined FRONTEND_ADMIN_PORT_SELECTED call :findUsablePort 3001 200 FRONTEND_ADMIN_PORT_SELECTED
if not defined API_PORT_SELECTED call :findUsablePort 18080 200 API_PORT_SELECTED
if not defined RESOURCE_POOL_PORT_SELECTED call :findUsablePort 18090 200 RESOURCE_POOL_PORT_SELECTED

if defined FRONTEND_USER_PORT_SELECTED (
  if defined FRONTEND_ADMIN_PORT_SELECTED (
    if "!FRONTEND_ADMIN_PORT_SELECTED!"=="!FRONTEND_USER_PORT_SELECTED!" (
      set "FRONTEND_USER_PORT_SELECTED="
      set /a "USER_BASE=!FRONTEND_ADMIN_PORT_SELECTED!+1"
      call :findUsablePort !USER_BASE! 200 FRONTEND_USER_PORT_SELECTED
      if not defined FRONTEND_USER_PORT_SELECTED call :findUsablePort 3000 200 FRONTEND_USER_PORT_SELECTED
    )
  )
)
if defined API_PORT_SELECTED (
  if defined RESOURCE_POOL_PORT_SELECTED (
    if "!RESOURCE_POOL_PORT_SELECTED!"=="!API_PORT_SELECTED!" (
      set "RESOURCE_POOL_PORT_SELECTED="
      set /a "RESOURCE_POOL_BASE=!API_PORT_SELECTED!+1"
      call :findUsablePort !RESOURCE_POOL_BASE! 200 RESOURCE_POOL_PORT_SELECTED
      if not defined RESOURCE_POOL_PORT_SELECTED call :findUsablePort 18090 200 RESOURCE_POOL_PORT_SELECTED
    )
  )
)

if not defined FRONTEND_USER_PORT_SELECTED (
  echo [错误] 无法找到可用的用户端前端端口
  set "EXIT_CODE=1"
  goto :die
)
if not defined FRONTEND_ADMIN_PORT_SELECTED (
  echo [错误] 无法找到可用的管理员端前端端口
  set "EXIT_CODE=1"
  goto :die
)
if not defined API_PORT_SELECTED (
  echo [错误] 无法找到可用的 API 端口
  set "EXIT_CODE=1"
  goto :die
)
if not defined RESOURCE_POOL_PORT_SELECTED (
  echo [错误] 无法找到可用的资源池端口
  set "EXIT_CODE=1"
  goto :die
)

set "FRONTEND_USER_PORT=%FRONTEND_USER_PORT_SELECTED%"
set "FRONTEND_ADMIN_PORT=%FRONTEND_ADMIN_PORT_SELECTED%"
set "API_PORT=%API_PORT_SELECTED%"
set "RESOURCE_POOL_PORT=%RESOURCE_POOL_PORT_SELECTED%"
set "NEXT_PUBLIC_API_URL=http://localhost:%API_PORT%"

if not "%FRONTEND_USER_PORT%"=="%REQUESTED_FRONTEND_USER_PORT%" (
  echo [警告] 用户端前端端口已自动从 %REQUESTED_FRONTEND_USER_PORT% 切换为 %FRONTEND_USER_PORT%
)
if not "%FRONTEND_ADMIN_PORT%"=="%REQUESTED_FRONTEND_ADMIN_PORT%" (
  echo [警告] 管理员端前端端口已自动从 %REQUESTED_FRONTEND_ADMIN_PORT% 切换为 %FRONTEND_ADMIN_PORT%
)
if not "%API_PORT%"=="%REQUESTED_API_PORT%" (
  echo [警告] API 端口已自动从 %REQUESTED_API_PORT% 切换为 %API_PORT%
)
if not "%RESOURCE_POOL_PORT%"=="%REQUESTED_RESOURCE_POOL_PORT%" (
  echo [警告] 资源池端口已自动从 %REQUESTED_RESOURCE_POOL_PORT% 切换为 %RESOURCE_POOL_PORT%
)
echo [信息] NEXT_PUBLIC_API_URL=%NEXT_PUBLIC_API_URL%
echo.

echo [步骤 4/6] 启动 Docker 后端服务...
docker compose -f "docker-compose.yml" up -d --remove-orphans
if errorlevel 1 (
  echo [错误] Docker 服务启动失败！
  docker compose -f "docker-compose.yml" ps
  set "EXIT_CODE=1"
  goto :die
)

echo [信息] 等待后端服务就绪...
call :waitHttp "http://localhost:%API_PORT%/health" 60
if errorlevel 1 (
  echo [错误] API 健康检查失败: http://localhost:%API_PORT%/health
  docker compose -f "docker-compose.yml" logs --tail=200 api
  set "EXIT_CODE=1"
  goto :die
)
call :waitHttp "http://localhost:%RESOURCE_POOL_PORT%/health" 60
if errorlevel 1 (
  echo [错误] 资源池健康检查失败: http://localhost:%RESOURCE_POOL_PORT%/health
  docker compose -f "docker-compose.yml" logs --tail=200 resource-pool
  set "EXIT_CODE=1"
  goto :die
)
echo [信息] 后端服务已就绪
echo.

echo [步骤 5/6] 检查前端依赖...
call :ensureFrontendDeps
if errorlevel 1 (
  set "EXIT_CODE=1"
  goto :die
)

if "%VERIFY_MODE%"=="1" (
  echo.
  echo [验证模式] 执行前端构建校验（不会启动 dev server，不会打开浏览器）...
  call :buildFrontends
  if errorlevel 1 (
    echo [错误] 前端构建校验失败
    set "EXIT_CODE=1"
    goto :die
  )

  echo [验证模式] 停止 Docker 服务...
  docker compose -f "docker-compose.yml" down >nul 2>&1

  echo.
  echo [验证模式] 校验通过
  echo   API 服务:   http://localhost:%API_PORT%
  echo   资源池:     http://localhost:%RESOURCE_POOL_PORT%
  echo.
  set "EXIT_CODE=0"
  goto :cleanup
)

echo.
echo [步骤 6/6] 启动前端服务并打开浏览器...
echo [信息] 启动用户端前端 (端口 %FRONTEND_USER_PORT%)...
start "Nexus User Frontend" /D "frontend-user" cmd /k "set \"NEXT_PUBLIC_API_URL=%NEXT_PUBLIC_API_URL%\" && node_modules\\.bin\\next.cmd dev -p %FRONTEND_USER_PORT%"
echo [信息] 启动管理员端前端 (端口 %FRONTEND_ADMIN_PORT%)...
start "Nexus Admin Frontend" /D "frontend-admin" cmd /k "set \"NEXT_PUBLIC_API_URL=%NEXT_PUBLIC_API_URL%\" && node_modules\\.bin\\next.cmd dev -p %FRONTEND_ADMIN_PORT%"

call :sleep 5
echo [信息] 等待用户端前端就绪...
call :waitHttp "http://localhost:%FRONTEND_USER_PORT%" 120
if errorlevel 1 (
  echo [错误] 用户端前端就绪检查失败: http://localhost:%FRONTEND_USER_PORT%
) else (
  start "" "http://localhost:%FRONTEND_USER_PORT%"
)
call :sleep 1
echo [信息] 等待管理员端前端就绪...
call :waitHttp "http://localhost:%FRONTEND_ADMIN_PORT%" 120
if errorlevel 1 (
  echo [错误] 管理员端前端就绪检查失败: http://localhost:%FRONTEND_ADMIN_PORT%
) else (
  start "" "http://localhost:%FRONTEND_ADMIN_PORT%"
)

echo.
echo ============================================
echo    所有服务启动完成！
echo ============================================
echo.
echo   用户端前端: http://localhost:%FRONTEND_USER_PORT%
echo   管理员前端: http://localhost:%FRONTEND_ADMIN_PORT%
echo   API 服务:   http://localhost:%API_PORT%
echo   资源池:     http://localhost:%RESOURCE_POOL_PORT%
echo.
echo   停止 Docker 服务: docker compose -f "docker-compose.yml" down
echo   前端服务请手动关闭对应的命令行窗口
echo.
pause
set "EXIT_CODE=0"
goto :cleanup

:: 统一退出（显示 pause 信息后退出）
:die
if "%VERIFY_MODE%"=="1" (
  docker compose -f "docker-compose.yml" down >nul 2>&1
  goto :cleanup
)
echo.
pause
goto :cleanup

:: 清理并退出（恢复工作目录、结束 setlocal）
:cleanup
cd /d "%ORIG_DIR%" >nul 2>&1
endlocal & exit /b %EXIT_CODE%

:: 从 .env 读取常用端口（不存在则使用默认值）
:loadEnvPorts
set "FRONTEND_USER_PORT=3000"
set "FRONTEND_ADMIN_PORT=3001"
set "API_PORT=8080"
set "RESOURCE_POOL_PORT=8090"
set "PROMETHEUS_PORT=9090"
set "GRAFANA_PORT=3002"
if not exist ".env" goto :eof
for /f "usebackq eol=# tokens=1,* delims==" %%A in (".env") do (
  if /I "%%A"=="FRONTEND_USER_PORT" set "FRONTEND_USER_PORT=%%B"
  if /I "%%A"=="FRONTEND_ADMIN_PORT" set "FRONTEND_ADMIN_PORT=%%B"
  if /I "%%A"=="API_PORT" set "API_PORT=%%B"
  if /I "%%A"=="RESOURCE_POOL_PORT" set "RESOURCE_POOL_PORT=%%B"
  if /I "%%A"=="PROMETHEUS_PORT" set "PROMETHEUS_PORT=%%B"
  if /I "%%A"=="GRAFANA_PORT" set "GRAFANA_PORT=%%B"
)
goto :eof

:: 找到一个可绑定端口（优先从 basePort 起向上探测）
:findUsablePort
setlocal EnableExtensions EnableDelayedExpansion
set "BASE=%~1"
set "MAX=%~2"
set "OUT_VAR=%~3"
if "%BASE%"=="" goto :findUsablePort_end
if "%MAX%"=="" set "MAX=30"
for /l %%P in (0,1,%MAX%) do (
  set /a "CANDIDATE=BASE+%%P"
  call :canBindPort !CANDIDATE!
  if not errorlevel 1 (
    for /f %%Q in ("!CANDIDATE!") do (
      endlocal & set "%OUT_VAR%=%%Q"
      goto :eof
    )
  )
)
:findUsablePort_end
endlocal
goto :eof

:: 判断端口是否可绑定（同时覆盖：占用/保留/排除端口等情况）
:canBindPort
setlocal
set "P=%~1"
if "%P%"=="" (
  endlocal & exit /b 1
)
powershell -NoProfile -Command "try { $port=%P%; $l6=[System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::IPv6Any, $port); try { $l6.Server.DualMode=$true } catch {} ; $l6.Start(); $l6.Stop(); exit 0 } catch { $ex=$_.Exception; if ($ex -is [System.Net.Sockets.SocketException] -and $ex.SocketErrorCode -eq [System.Net.Sockets.SocketError]::AddressFamilyNotSupported) { try { $l4=[System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Any, %P%); $l4.Start(); $l4.Stop(); exit 0 } catch { exit 1 } } exit 1 }" >nul 2>&1
endlocal & exit /b %errorlevel%

:: 睡眠 N 秒（避免 timeout 在重定向输入时出错）
:sleep
setlocal EnableExtensions
set "SECONDS=%~1"
if "%SECONDS%"=="" set "SECONDS=1"
set /a "COUNT=%SECONDS%+1"
ping -n %COUNT% 127.0.0.1 >nul 2>&1
endlocal & exit /b 0

:: 启动并等待 Docker Desktop 引擎就绪
:startDockerDesktop
setlocal EnableExtensions EnableDelayedExpansion
set "DOCKER_DESKTOP_EXE=%ProgramFiles%/Docker/Docker/Docker Desktop.exe"
if not exist "%DOCKER_DESKTOP_EXE%" set "DOCKER_DESKTOP_EXE=%ProgramFiles(x86)%/Docker/Docker/Docker Desktop.exe"
if not exist "%DOCKER_DESKTOP_EXE%" (
  endlocal & exit /b 1
)
start "" "%DOCKER_DESKTOP_EXE%" >nul 2>&1
for /l %%I in (1,1,150) do (
  docker info >nul 2>&1
  if not errorlevel 1 (
    endlocal & exit /b 0
  )
  call :sleep 2
)
endlocal & exit /b 1

:: 等待 HTTP 端点返回 2xx
:waitHttp
setlocal EnableExtensions EnableDelayedExpansion
set "URL=%~1"
set "SECONDS=%~2"
if "%SECONDS%"=="" set "SECONDS=60"
for /l %%I in (1,1,%SECONDS%) do (
  powershell -NoProfile -Command "try { $r=Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 -Uri '%URL%'; if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) { exit 0 } exit 1 } catch { exit 1 }" >nul 2>&1
  if not errorlevel 1 (
    endlocal & exit /b 0
  )
  call :sleep 1
)
endlocal & exit /b 1

:: 检查并安装前端依赖（如缺失）
:ensureFrontendDeps
setlocal
if not exist "frontend-user\\node_modules" (
  echo [信息] 安装 frontend-user 依赖...
  pushd "frontend-user" >nul 2>&1
  if errorlevel 1 (
    echo [错误] 未找到目录: "frontend-user"
    popd >nul 2>&1
    endlocal & exit /b 1
  )
  call npm.cmd install
  if errorlevel 1 (
    popd >nul 2>&1
    echo [错误] frontend-user 依赖安装失败！
    endlocal & exit /b 1
  )
  popd >nul 2>&1
)

if not exist "frontend-admin\\node_modules" (
  echo [信息] 安装 frontend-admin 依赖...
  pushd "frontend-admin" >nul 2>&1
  if errorlevel 1 (
    echo [错误] 未找到目录: "frontend-admin"
    popd >nul 2>&1
    endlocal & exit /b 1
  )
  call npm.cmd install
  if errorlevel 1 (
    popd >nul 2>&1
    echo [错误] frontend-admin 依赖安装失败！
    endlocal & exit /b 1
  )
  popd >nul 2>&1
)

endlocal & exit /b 0

:: 前端构建校验（用于 --verify）
:buildFrontends
setlocal
pushd "frontend-user" >nul 2>&1
set "NEXT_PUBLIC_API_URL=%NEXT_PUBLIC_API_URL%"
call npm.cmd run build
if errorlevel 1 (
  popd >nul 2>&1
  endlocal & exit /b 1
)
popd >nul 2>&1

pushd "frontend-admin" >nul 2>&1
set "NEXT_PUBLIC_API_URL=%NEXT_PUBLIC_API_URL%"
call npm.cmd run build
if errorlevel 1 (
  popd >nul 2>&1
  endlocal & exit /b 1
)
popd >nul 2>&1

endlocal & exit /b 0

:: 函数：杀死指定端口的进程
:killPort
setlocal
set PORT=%~1
if "%PORT%"=="" (
    endlocal
    goto :eof
)
powershell -NoProfile -Command "$port=%PORT%; $processIds = netstat -ano | Select-String (':'+$port+'\\s') | Select-String 'LISTENING' | ForEach-Object { ($_.Line -split '\\s+')[-1] } | Sort-Object -Unique; foreach ($processId in $processIds) { if (-not $processId) { continue }; if ($processId -eq 4) { Write-Host ('[警告] 端口 {0} 被 System (PID:4) 占用/保留，已跳过。' -f $port); continue }; $p = Get-Process -Id $processId -ErrorAction SilentlyContinue; if ($p -and @('com.docker.backend','wslrelay') -contains $p.ProcessName) { Write-Host ('[警告] 端口 {0} 被 {1} (PID:{2}) 占用，疑似 Docker/WSL 端口转发，已跳过杀进程。' -f $port, $p.ProcessName, $processId); continue }; Write-Host ('[信息] 正在终止占用端口 {0} 的进程 (PID:{1})...' -f $port, $processId); Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue }" >nul 2>&1
endlocal
goto :eof
