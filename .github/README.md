# GitHub Actions 自动编译说明

## 功能说明

本项目配置了 GitHub Actions 自动编译工作流，可以自动构建前后端并生成多平台可执行文件。

## 触发条件

工作流会在以下情况自动触发：

1. **推送到 main 分支** - 自动构建并上传 artifacts
2. **创建 tag（如 v1.0.0）** - 自动构建并创建 GitHub Release
3. **Pull Request** - 自动构建测试
4. **手动触发** - 在 Actions 页面手动运行

## 构建产物

### 应用程序二进制文件

- `new-api-linux-amd64` - Linux x64
- `new-api-linux-arm64` - Linux ARM64
- `new-api-windows-amd64.exe` - Windows x64
- `new-api-darwin-amd64` - macOS Intel
- `new-api-darwin-arm64` - macOS Apple Silicon

### Docker 镜像

- Docker Hub: `your-username/new-api:latest`
- GitHub Container Registry: `ghcr.io/ketion/ketionapi:latest`

## 配置 Docker Hub（可选）

如果需要推送到 Docker Hub，请在仓库设置中添加以下 Secrets：

1. 进入仓库 Settings → Secrets and variables → Actions
2. 添加以下 secrets：
   - `DOCKERHUB_USERNAME` - 你的 Docker Hub 用户名
   - `DOCKERHUB_TOKEN` - 你的 Docker Hub Access Token

## 发布新版本

1. 更新 `VERSION` 文件中的版本号
2. 提交并推送到 main 分支
3. 创建并推送 tag：

```bash
git tag v1.0.0
git push origin v1.0.0
```

4. GitHub Actions 会自动构建并创建 Release

## 下载构建产物

- **Artifacts**: 在 Actions 页面的每次运行中下载（保留 7 天）
- **Release**: 在 Releases 页面下载（永久保存）

## 本地测试

如果需要在本地测试构建流程：

```bash
# 前端构建
cd web
bun install
DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build

# 后端构建
cd ..
go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api
```
