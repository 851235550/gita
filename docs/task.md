# Gita MVP 任务拆解（面向 AI Agent 执行）

> 基于《Gita 需求文档 v1.0》拆解。每个任务包含：目标、依赖、交付物、验收标准，供 AI Agent（如 Claude Code）逐条领取执行。
> 建议按 Phase 顺序执行，同一 Phase 内标注"可并行"的任务无依赖关系，可并行处理。

---

## 依赖关系总览

```text
Phase 0: 项目初始化
    ↓
Phase 1: git 模块 ──┐
Phase 1: prompt 模块 ─┤ (可并行)
Phase 1: llm 模块 ───┘
    ↓
Phase 2: context 模块（依赖 git 模块）
    ↓
Phase 3: cmd/gita commit 主流程（依赖 context + prompt + llm）
    ↓
Phase 4: 用户交互细化（e/r/n、--hint 等）
    ↓
Phase 5: 异常处理
    ↓
Phase 6: 测试与验收
```

---

## Phase 0：项目初始化

### T0.1 初始化项目骨架

- **目标**：创建符合技术架构的目录结构与基础工程文件
- **依赖**：无
- **交付物**：
  ```text
  gita/
  ├── cmd/gita/main.go
  ├── internal/git/
  ├── internal/context/
  ├── internal/prompt/
  ├── internal/llm/
  ├── go.mod
  └── README.md
  ```
- **验收标准**：`go build ./...` 成功执行，`gita --help` 有基础输出
- **备注**：语言未强制指定 Go，若使用其他语言（Node/Python/Rust）按对应工程规范调整目录命名，模块划分保持一致

### T0.2 配置文件加载器

- **目标**：实现 `~/.gita/config.yaml` 的读取、解析、默认值填充
- **依赖**：T0.1
- **交付物**：`internal/config/config.go`（或对应语言路径），支持解析需求文档 7.2 节的完整字段结构
- **验收标准**：
  - 配置文件不存在时，使用内置默认值且不报错
  - `providers` / `default_provider` / `language` / `commit_style` / `max_diff_lines` 均可正确解析
  - 单元测试覆盖：空配置、部分字段缺失、非法 yaml 格式

---

## Phase 1：基础模块（三者互不依赖，可并行）

### T1.1【git 模块】封装 Git 命令调用

- **目标**：封装 `git diff --staged`、`git diff --staged --stat`、`git diff --staged --name-only`、`git status` 的调用与结果解析
- **依赖**：T0.1
- **交付物**：`internal/git/diff.go`
  - `GetStagedDiff() (string, error)`
  - `GetStagedStat() (string, error)`
  - `GetStagedFileNames() ([]string, error)`
  - `IsGitRepo() bool`
  - `HasStagedChanges() bool`
- **验收标准**：
  - 在非 Git 仓库目录调用返回明确错误，不 panic
  - 无 staged 变更时 `HasStagedChanges()` 返回 false
  - 单元测试：使用临时 Git 仓库 fixture 验证各函数输出

### T1.2【git 模块】Hunk Header 解析（函数上下文提取）

- **目标**：从 `git diff` 输出中提取每个 hunk header（`@@ ... @@` 后的函数/类上下文文本）
- **依赖**：T1.1
- **交付物**：`internal/git/hunkparser.go`
  - `ExtractHunkContexts(diffText string) map[string][]string`（key 为文件路径，value 为该文件所有 hunk 的上下文文本，已去重）
- **验收标准**：
  - 正则 `^@@ .*@@ ?(.*)$` 正确匹配并提取末尾上下文
  - 上下文为空时返回空字符串而非报错（对应需求文档 6.5 节的降级要求）
  - 单元测试覆盖：Go / Python / JS 三种语言的 diff 样例

### T1.3【prompt 模块】模板加载与变量渲染

- **目标**：实现模板加载优先级（用户自定义 > 内置默认）与 `{{variable}}` 语法渲染
- **依赖**：T0.1
- **交付物**：`internal/prompt/loader.go` + `internal/prompt/templates/commit.md`（内置默认模板）
  - `LoadTemplate(name string) (string, error)`
  - `Render(template string, vars map[string]string) string`
- **验收标准**：
  - `~/.gita/prompts/commit.md` 存在时优先加载，不存在时使用内置模板
  - 支持需求文档 7.4 节列出的全部变量：`diff` / `stat` / `file_list` / `hint` / `language` / `style`
  - 未提供的变量渲染为空字符串而非报错或保留 `{{var}}` 原样
  - 单元测试覆盖模板缺失变量、多变量嵌套场景

### T1.4【llm 模块】统一 Provider 接口定义

- **目标**：定义与具体 LLM 无关的统一调用接口
- **依赖**：T0.1
- **交付物**：`internal/llm/provider.go`
  ```text
  interface Provider {
    Generate(ctx, prompt string) (string, error)
  }
  ```
- **验收标准**：接口定义完成，附带 mock 实现供后续单元测试使用，不依赖任何真实网络请求

### T1.5【llm 模块】DeepSeek Provider 实现

- **目标**：实现 T1.4 接口的 DeepSeek 适配器，作为 MVP 默认打通的 Provider
- **依赖**：T1.4
- **交付物**：`internal/llm/deepseek.go`
- **验收标准**：
  - 从环境变量读取 Key（变量名由 config 中 `api_key_env` 指定，不硬编码）
  - 请求超时阈值 30s，超时返回明确错误类型（供上层判断是否提示重试）
  - Key 缺失时返回的错误信息包含"应设置的环境变量名"

### T1.6【llm 模块】OpenAI / Claude Provider 实现

- **目标**：实现另外两个 Provider 适配器
- **依赖**：T1.4（可与 T1.5 并行）
- **交付物**：`internal/llm/openai.go`、`internal/llm/claude.go`
- **验收标准**：与 T1.5 相同的验收标准，分别针对各自 API 格式

---

## Phase 2：context 模块（依赖 Phase 1 的 git 模块）

### T2.1 Diff 行数分级判断

- **目标**：根据 diff 行数返回分级结果（小型/中型/大型/超大型）
- **依赖**：T1.1
- **交付物**：`internal/context/classifier.go`
  - `ClassifyDiff(lineCount int) DiffLevel`（枚举：Small / Medium / Large / ExtraLarge）
- **验收标准**：边界值测试（999/1000/5000/5001/15000/15001 行），阈值需从 config 的 `max_diff_lines` 读取而非硬编码

### T2.2 摘要压缩实现（大型变更）

- **目标**：实现需求文档 6.5 节"抓大放小"的摘要压缩逻辑
- **依赖**：T1.1、T1.2、T2.1
- **交付物**：`internal/context/summarizer.go`
  - `BuildSummary(files []FileChange) string`
  - 单文件 < 50 行改动 → 全量 diff；否则 → 文件路径 + 变更类型 + 增删统计 + hunk 上下文列表
- **验收标准**：混合场景测试（同一次 commit 里既有小文件又有大文件），验证输出中小文件为全量、大文件为摘要

### T2.3 Context Builder 主入口

- **目标**：整合 T2.1、T2.2，根据分级结果拼装最终发送给 LLM 的上下文文本
- **依赖**：T2.1、T2.2
- **交付物**：`internal/context/builder.go`
  - `BuildContext(diff, stat, fileNames string, level DiffLevel) string`
- **验收标准**：四个分级档位均有对应单元测试，输出内容符合需求文档 6.4 节表格定义

---

## Phase 3：主流程（依赖 Phase 1 + Phase 2 全部完成）

### T3.1 `gita commit` 命令骨架

- **目标**：实现主命令入口，串联 git → context → prompt → llm 全流程（暂不含交互细化）
- **依赖**：T1._、T2._、T0.2
- **交付物**：`cmd/gita/commit.go`
- **验收标准**：
  - `git add . && gita commit` 能输出建议的 commit message（无需交互，先打印即可）
  - 无 staged 变更时按需求文档第 8 节提示，不调用 LLM

### T3.2 CLI 参数解析

- **目标**：实现 `--hint` / `--lang` / `--style` / `--provider` / `--api-key` / `--force` 参数解析
- **依赖**：T3.1
- **交付物**：`cmd/gita/flags.go`
- **验收标准**：各参数均能正确覆盖 config.yaml 默认值，`--api-key` 不写入任何文件（仅存于运行时内存）

---

## Phase 4：用户交互细化

### T4.1 结果确认交互（Y/e/r/n）

- **目标**：实现需求文档 6.6 节的四种交互操作
- **依赖**：T3.1
- **交付物**：`cmd/gita/interact.go`
- **验收标准**：
  - `Y`（默认回车）执行 `git commit -m`
  - `e` 打开 `$EDITOR`（读取 `$EDITOR` 环境变量，未设置时降级为内置简单输入）
  - `r` 重新调用 LLM，不限次数，且不复用旧的 HTTP 连接失败重试逻辑（每次视为新请求）
  - `n` 取消且不产生任何 git 操作
  - 集成测试：mock LLM 返回固定文本，模拟四种按键路径

### T4.2 `--force` 超大 diff 确认流程

- **目标**：实现需求文档 6.4 节超大型变更的用户提示与 `--force` 跳过逻辑
- **依赖**：T2.1、T3.2
- **交付物**：在 `cmd/gita/commit.go` 中补充分支逻辑
- **验收标准**：超大型 diff 且未传 `--force` 时阻塞等待用户确认；传入 `--force` 时跳过提示直接生成

---

## Phase 5：异常处理（覆盖需求文档第 8 节全部场景）

### T5.1 异常处理统一封装

- **目标**：将第 8 节列出的 7 种异常场景实现为统一的错误提示层，不与业务逻辑耦合
- **依赖**：T3.1、T1.5
- **交付物**：`internal/errors/handler.go`
- **验收标准**：逐条对照需求文档第 8 节表格，每种场景均有对应的单元测试断言提示文案包含关键信息（如具体缺失的环境变量名）

---

## Phase 6：测试与验收

### T6.1 端到端集成测试

- **目标**：搭建可重复执行的集成测试，覆盖需求文档第 11 节"验收标准"全部 8 条
- **依赖**：Phase 0-5 全部完成
- **交付物**：`test/e2e/commit_test.go`（或对应语言的集成测试目录）
- **验收标准**：CI 中一键运行，全部用例通过

### T6.2 README 与使用文档

- **目标**：编写面向最终用户的安装、配置、使用说明
- **依赖**：T6.1
- **交付物**：`README.md`
- **验收标准**：新用户按文档操作，从 `git clone` 到成功执行一次 `gita commit` 全程无需查阅额外资料

---

## 给 AI Agent 的执行建议

0. **每个任务在标记完成前，必须同时满足《Gita 开发规范》（单元测试与代码注释要求）**，该文档中的检查清单与本文档各任务的"验收标准"同等强制，不是可选项。
1. **每次只领取一个任务**，完成后运行该任务对应的验收标准自测，通过后再领取下一个，避免因未验证的上游模块导致下游返工。
2. **Phase 1 内的 T1.1/T1.3/T1.4 可分别开新分支并行开发**，但 T1.5/T1.6 依赖 T1.4 接口定义先合并。
3. 涉及真实 API Key 的任务（T1.5、T1.6、T6.1 中的真实 LLM 调用部分），本地开发环境需要在 `.env` 或 shell 中预设对应环境变量，**不要**将真实 Key 写入任何会被提交的文件。
4. 每个任务的"验收标准"即该任务的 Definition of Done，agent 可据此自行编写并运行单元测试，无需等待人工确认后再继续下一任务。
