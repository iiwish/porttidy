# Cross-Platform Testing Guide

> 如何在 macOS 上测试 Linux 和 Windows 版本的 porttidy

---

## Linux 测试

### 方案 1：Docker（推荐 ⭐）

最轻量、最快、最推荐。Go 编译成静态二进制文件，在容器里直接跑。

```bash
# 1. 编译 Linux 版二进制（macOS 上交叉编译）
GOOS=linux GOARCH=amd64 go build -o porttidy-linux ./cmd/porttidy

# 2. 启动一个带开发环境的容器
docker run -it --rm \
  --name porttidy-test \
  -v $(pwd):/workspace \
  -w /workspace \
  node:20-alpine sh

# 在容器里：
# 安装 Go（如果需要源码编译）
apk add --no-cache go

# 运行测试
./porttidy-linux scan --json

# 启动一些 dev server 来测试扫描
npx serve -l 8080 &
npx http-server -p 8765 &
npx vite --port 5173 &

# 再扫描
./porttidy-linux scan
./porttidy-linux kill --all --dry-run
```

**多发行版矩阵**：

```bash
# Ubuntu
docker run -it --rm -v $(pwd):/workspace -w /workspace ubuntu:22.04 bash
# 容器内: apt-get update && apt-get install -y golang-go procps lsof

# Alpine（musl libc，更严格）
docker run -it --rm -v $(pwd):/workspace -w /workspace alpine sh
# 容器内: apk add go procps lsof

# Fedora
docker run -it --rm -v $(pwd):/workspace -w /workspace fedora bash
# 容器内: dnf install -y golang procps-ng lsof
```

### 方案 2：OrbStack（macOS 推荐）

如果你用 OrbStack 代替 Docker Desktop，体验更好：

```bash
# OrbStack 支持直接运行 Linux VM
orb create ubuntu porttidy-ubuntu
orb ssh porttidy-ubuntu
# 然后像普通 Linux 一样操作
```

### 方案 3：GitHub Actions CI（自动化）

`.github/workflows/test.yml`：

```yaml
name: Cross-Platform Test

on: [push, pull_request]

jobs:
  test-linux:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        distro: [ubuntu-latest, ubuntu-22.04]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./...
      - run: go build ./cmd/porttidy
      # 启动测试进程
      - run: |
          python3 -m http.server 8080 &
          python3 -m http.server 8765 &
          sleep 2
      - run: ./porttidy scan --json
      - run: ./porttidy kill --all --force

  test-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./...
      - run: go build ./cmd/porttidy
```

---

## Windows 测试

### 方案 1：GitHub Actions CI（最实际）

```yaml
  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./...
      - run: go build ./cmd/porttidy
      - name: Start test servers
        shell: pwsh
        run: |
          Start-Process python -ArgumentList "-m http.server 8080" -WindowStyle Hidden
          Start-Process python -ArgumentList "-m http.server 8765" -WindowStyle Hidden
          Start-Sleep 5
      - run: .\porttidy.exe scan --json
```

**优点**：免费、自动化、每次 push 都测  
**缺点**：无法交互式调试，输出只能通过日志看

### 方案 2：本地 Windows VM（Parallels / UTM / VMware）

如果你用 Apple Silicon Mac：

| 工具 | 费用 | 体验 | 推荐 |
|------|------|------|------|
| **UTM** | 免费 | 够用，配置稍复杂 | ⭐ 推荐 |
| **Parallels Desktop** | 付费（~500/年） | 最好，无缝集成 | 土豪选 |
| **VMware Fusion** | 个人免费 | 好 | 备选 |

**UTM 安装 Windows 步骤**：

```bash
# 1. 下载 Windows 11 ARM64 ISO
# https://www.microsoft.com/software-download/windows11

# 2. UTM 新建 VM
# - 架构: ARM64 (Virtualization)
# - 内存: 4GB+
# - 磁盘: 40GB+

# 3. 安装后启用 WSL2（用于 Go 开发环境）
wsl --install
# 重启后在 WSL Ubuntu 里: sudo apt install golang-go

# 4. 共享文件夹（macOS ↔ Windows）
# UTM 设置 → Shared Directory → 选 ~/self/porttidy
```

### 方案 3：云 Windows VM（按需使用）

不需要长期维护 VM，测试时才开：

**Azure**（新用户有 $200 免费额度）：
```bash
# Azure CLI 创建 Windows VM（Standard_B2s，~0.08 美元/小时）
az vm create \
  --resource-group porttidy-test \
  --name windows-test \
  --image MicrosoftWindowsDesktop:Windows-11:win11-23h2-pro:latest \
  --size Standard_B2s \
  --admin-username iiwish \
  --admin-password <password>

# RDP 连接
az vm show -d -g porttidy-test -n windows-test --query publicIps -o tsv
# 用 Microsoft Remote Desktop App 连接

# 用完删除
az vm delete -g porttidy-test -n windows-test --yes
```

**AWS EC2**（t3.medium，~0.04 美元/小时）：
```bash
aws ec2 run-instances \
  --image-id ami-xxxxx  # Windows Server 2022 AMI
  --instance-type t3.medium \
  --key-name your-key
```

### 方案 4：朋友的 Windows 电脑（最接地气）

```bash
# 1. 在 macOS 上交叉编译 Windows 版
GOOS=windows GOARCH=amd64 go build -o porttidy.exe ./cmd/porttidy

# 2. 发 exe 给朋友
# 3. 朋友双击运行（Windows Defender 可能会报毒，需要加签名或信任）
```

---

## 推荐测试策略（实际可行的）

### 日常开发（macOS）

```bash
# 主开发环境
make dev        # go run ./cmd/porttidy
make test       # go test ./...
```

### 交叉编译验证（每次 push 前）

```bash
# Makefile 目标
make build-all:
	GOOS=darwin  GOARCH=arm64 go build -o dist/porttidy-darwin-arm64 ./cmd/porttidy
	GOOS=darwin  GOARCH=amd64 go build -o dist/porttidy-darwin-amd64 ./cmd/porttidy
	GOOS=linux   GOARCH=amd64 go build -o dist/porttidy-linux-amd64 ./cmd/porttidy
	GOOS=linux   GOARCH=arm64 go build -o dist/porttidy-linux-arm64 ./cmd/porttidy
	GOOS=windows GOARCH=amd64 go build -o dist/porttidy-windows-amd64.exe ./cmd/porttidy
```

### 自动化 CI（GitHub Actions）

每次 push 自动在 3 个平台运行测试：

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [macos-latest, ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go test ./...
      - run: go build ./cmd/porttidy
      - name: Integration test
        run: |
          # 各平台启动测试进程
          # 运行 porttidy scan
          # 验证输出
```

### 手动深度测试（每周/发布前）

| 平台 | 方式 | 频率 |
|------|------|------|
| macOS | 本机 | 每次开发 |
| Linux | Docker 容器 | 每次 push |
| Linux | GitHub Actions | 每次 PR |
| Windows | GitHub Actions | 每次 PR |
| Windows | 云 VM / 本地 VM | 发版前 |

---

## 测试脚本模板

```bash
#!/bin/bash
# scripts/integration-test.sh

set -e

echo "=== Porttidy Integration Test ==="

# 编译
go build -o porttidy-test ./cmd/porttidy

# 启动测试进程
echo "Starting test servers..."
python3 -m http.server 8080 &
PID1=$!
python3 -m http.server 8765 &
PID2=$!
sleep 2

# 扫描
echo "Scanning..."
./porttidy-test scan --json | jq '.summary'

# 关闭（dry-run）
echo "Dry-run kill..."
./porttidy-test kill --all --dry-run

# 实际关闭
echo "Killing..."
./porttidy-test kill --all --force

# 验证
echo "Verifying..."
if lsof -i:8080 >/dev/null 2>&1; then
  echo "FAIL: port 8080 still occupied"
  exit 1
fi
if lsof -i:8765 >/dev/null 2>&1; then
  echo "FAIL: port 8765 still occupied"
  exit 1
fi

echo "PASS"

# 清理
rm -f porttidy-test
```

---

## 平台差异速查

| 功能 | macOS | Linux | Windows |
|------|-------|-------|---------|
| 进程列表 | `ps aux` + `lsof` | `/proc/*/status`, `/proc/*/cmdline` | WMI (`wmic process list`) + psapi |
| 工作目录 | `lsof -a -d cwd -p PID` | `/proc/PID/cwd` (readlink) | `GetProcessId` + `QueryFullProcessImageName` |
| 端口占用 | `lsof -i` | `/proc/PID/fd` + `readlink` → socket inode → `/proc/net/tcp` | `netstat -ano` + WMI |
| 关闭进程 | `kill -9 PID` | `kill -9 PID` | `taskkill /F /PID PID` |
| PPID | `ps -o ppid= -p PID` | `/proc/PID/status` (PPid) | WMI `ParentProcessId` |
| 命令行 | `ps -o command=` | `/proc/PID/cmdline` | WMI `CommandLine` |
| 启动时间 | `ps -o lstart=` | `/proc/PID/stat` (field 22) | WMI `CreationDate` |

---

## 最小可行测试方案

如果你只想**最快落地**而不追求全面：

1. **macOS**：本机日常开发测试 ✅
2. **Linux**：GitHub Actions CI（免费，每次 push 自动测）✅
3. **Windows**：GitHub Actions CI（免费）+ 发版前用云 VM 手动验证一次 ✅

这样你不需要维护任何本地 VM，零额外成本。
