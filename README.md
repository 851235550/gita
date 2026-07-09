# Gita —— AI 驱动的 Git 提交助手

Gita 是一个基于 Git 与大语言模型（LLM）的 CLI 工具，自动分析 `git diff --staged` 内容，生成高质量的 Commit Message。

---

## 快速开始

### 方式一：从源码编译（推荐）

```bash
# 克隆仓库
git clone https://github.com/your-org/gita.git
cd gita

# 编译当前平台
make build
# 产物在 build/gita
```

### 方式二：下载预编译二进制

从 [Releases](https://github.com/851235550/gita/releases) 页面下载对应平台的二进制文件。

### 方式三：使用 go install

```bash
go install github.com/851235550/gita/cmd/gita@latest
```

---

## 安装与验证

### macOS / Linux

```bash
# 方式 A：使用 Makefile 一键安装（需要 sudo）
sudo make install

# 方式 B：手动复制到 PATH 目录
# macOS arm64 (Apple Silicon M1/M2/M3/M4)
sudo cp build/gita-darwin-arm64 /usr/local/bin/gita
sudo chmod +x /usr/local/bin/gita

# macOS amd64 (Intel)
sudo cp build/gita-darwin-amd64 /usr/local/bin/gita
sudo chmod +x /usr/local/bin/gita

# Linux amd64
sudo cp build/gita-linux-amd64 /usr/local/bin/gita
sudo chmod +x /usr/local/bin/gita

# Linux arm64 (树莓派 64 位 / AWS Graviton)
sudo cp build/gita-linux-arm64 /usr/local/bin/gita
sudo chmod +x /usr/local/bin/gita

# 方式 C：添加到用户 PATH（无需 sudo，推荐 ~/.local/bin）
mkdir -p ~/.local/bin
cp build/gita-* ~/.local/bin/gita   # 选择对应平台的文件
chmod +x ~/.local/bin/gita
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc  # 或 ~/.zshrc
source ~/.bashrc
```

### 验证安装

```bash
# 确认 gita 可用
gita --help

# 查看版本信息（仅在通过 Makefile 编译时注入）
gita --version
```

---

## 全平台编译

项目提供 Makefile 支持交叉编译到 Linux / macOS 的 amd64 和 arm64 架构：

```bash
# 查看所有可用目标
make help

# 全平台编译（生成 4 个平台的二进制文件）
make build-all

# 产物：
#   build/gita-linux-amd64      # Linux x86_64
#   build/gita-linux-arm64      # Linux ARM64
#   build/gita-darwin-amd64     # macOS Intel
#   build/gita-darwin-arm64     # macOS Apple Silicon

# 单平台编译
make build-linux      # Linux amd64
make build-linux-arm  # Linux arm64
make build-mac        # macOS amd64 (Intel)
make build-mac-arm    # macOS arm64 (Apple Silicon)
```

---

## 配置

创建 `~/.gita/config.yaml`（可选，无配置文件时使用内置默认值）：

```yaml
default_provider: deepseek

providers:
  deepseek:
    model: deepseek-chat
    base_url: https://api.deepseek.com/v1
    api_key_env: GITA_DEEPSEEK_API_KEY
  openai:
    model: gpt-4o
    base_url: https://api.openai.com/v1
    api_key_env: GITA_OPENAI_API_KEY

language: zh-CN # zh-CN | en
commit_style: conventional # conventional | plain
max_diff_lines: 5000 # Medium/Large 分级阈值
```

### 设置 API Key

```bash
# 根据你使用的 Provider 设置对应的环境变量
export GITA_DEEPSEEK_API_KEY="sk-your-key-here"

# 建议写入 shell 配置文件持久化
echo 'export GITA_DEEPSEEK_API_KEY="sk-your-key-here"' >> ~/.zshrc
source ~/.zshrc
```

### 使用

```bash
# 暂存变更
git add .

# 一键生成 Commit Message
gita commit

# 带额外上下文
gita commit --hint "为兼容旧版 API 保留了废弃字段"

# 指定语言和风格
gita commit --lang en --style plain

# 切换 LLM Provider
gita commit --provider openai

# 临时指定 API Key（不落盘）
gita commit --api-key sk-xxx

# 超大 diff 强制生成
gita commit --force
```

生成后进入交互确认：

```text
Suggested Commit Message
─────────────────────────
feat(deployment): 初始化部署记录管理服务项目基础架构
─────────────────────────
[Y] 确认提交  [e] 编辑后提交  [r] 重新生成  [n] 取消
```

---

## 工作原理

```text
git diff --staged
    ↓
根据行数分级（Small / Medium / Large / ExtraLarge）
    ↓
构造 Prompt 上下文（含"抓大放小"摘要压缩）
    ↓
调用 LLM（DeepSeek / OpenAI / Claude）
    ↓
用户确认 / 编辑 / 重新生成 / 取消
    ↓
git commit -m "..."
```

---

## 分级策略

| 级别       | 行数         | 发送内容                                              |
| ---------- | ------------ | ----------------------------------------------------- |
| Small      | < 1000       | 完整 diff                                             |
| Medium     | 1000 ~ 4999  | 完整 diff + --stat                                    |
| Large      | 5000 ~ 14999 | 按文件摘要压缩（< 50 行文件保留完整 diff）            |
| ExtraLarge | >= 15000     | --stat + --name-only，提示拆分（可用 `--force` 跳过） |

---

## 自定义 Prompt 模板

在 `~/.gita/prompts/commit.md` 中编写自定义模板（不存在则使用内置模板）：

```markdown
你是专业代码审查者，根据以下变更生成 Commit Message。

## 变更内容

{{diff}}

## 额外说明

{{hint}}

请用 {{language}} 输出 {{style}} 风格的 Commit Message。
```

支持的变量：`{{diff}}` `{{stat}}` `{{file_list}}` `{{hint}}` `{{language}}` `{{style}}`

---

## 支持的环境变量

| 变量                     | 说明                                                 |
| ------------------------ | ---------------------------------------------------- |
| `GITA_DEEPSEEK_API_KEY`  | DeepSeek API Key                                     |
| `GITA_OPENAI_API_KEY`    | OpenAI API Key                                       |
| `GITA_ANTHROPIC_API_KEY` | Claude API Key                                       |
| `EDITOR`                 | 编辑模式使用的编辑器（可选，未设置时降级为内置输入） |

---

## 开发者文档

> 以下内容仅供 AI Agent 和开发者参考。

| 文档         | 文件                   | 说明                   |
| ------------ | ---------------------- | ---------------------- |
| 需求文档     | `docs/prd_v1.md`       | 产品需求与设计决策     |
| 任务拆解     | `docs/task.md`         | 任务列表与依赖关系     |
| 测试用例说明 | `docs/test_case.md`    | 测试用例编号与预期结果 |
| 开发规范     | `docs/develop_rule.md` | 测试与注释强制要求     |

### 项目结构

```text
gita/
├── cmd/gita/          # CLI 入口
│   ├── main.go        # 主流程
│   ├── flags.go       # 参数解析
│   └── interact.go    # 用户交互
├── internal/
│   ├── config/        # 配置加载
│   ├── git/           # Git 命令封装
│   ├── context/       # Diff 分级与摘要压缩
│   ├── prompt/        # 模板加载与渲染
│   ├── llm/           # LLM Provider 适配
│   └── errors/        # 异常处理
├── docs/              # 项目文档
└── go.mod
```

### 运行测试

```bash
# 全部单元测试
go test ./...

# 端到端测试
go test ./cmd/gita/ -v -run "E2E"
```
