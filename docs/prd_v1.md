# Gita 需求文档

| 项目     | 内容                   |
| -------- | ---------------------- |
| 文档版本 | v1.0                   |
| 状态     | MVP 需求已确认，待开发 |
| 更新日期 | 2026-07-08             |
| 负责人   | -                      |

---

## 1. 背景

开发者在完成代码修改后，编写高质量的 Commit Message、PR 描述、Release Notes 等文档性内容，普遍存在以下问题：

- 写 commit message 时容易敷衍（`fix bug`、`update`），事后难以追溯改动意图
- PR 描述、Release Notes 需要人工从多个 commit 中归纳总结，耗时且容易遗漏
- 现有 AI 辅助工具多数直接分析源码目录，缺乏对 Git 语义（变更范围、diff、历史）的针对性利用

Gita 希望通过 **Git 数据 + LLM** 的组合，成为开发者日常 Git 工作流中的辅助工具，而非独立的代码分析平台。

---

## 2. 产品目标

Gita 是一个基于 Git 与大语言模型（LLM）的开发辅助 CLI 工具，通过分析 Git 仓库中的变更内容，自动生成：

- Commit Message（MVP）
- Pull Request Description（v1）
- Release Notes（v2）
- Code Review Summary（v3）

**产品定位**：Gita 的目标不是替代 Git，而是成为 Git 工作流中的 AI 助手。

---

## 3. 目标用户与使用场景

| 用户画像           | 场景                                                                |
| ------------------ | ------------------------------------------------------------------- |
| 日常写代码的开发者 | `git add` 后不想手写 commit message，希望一键生成规范、准确的描述   |
| 提交 PR 前的开发者 | 需要快速整理本次分支相对 main 的变更说明，用于 Code Review          |
| Release 负责人     | 需要将一段时间内的多个 commit 归纳为对外发布的 Release Notes        |
| 开源维护者         | 希望自带 LLM Key 使用，不依赖 Gita 官方服务，成本与数据完全自主可控 |

---

## 4. 设计原则

### 4.1 Git First

所有能力均基于 Git 数据（`git diff` / `git status` / `git log` / `git show`），而非直接分析源码目录，保证语义聚焦在"变更"本身。

### 4.2 LLM Provider 无关

通过统一接口接入 DeepSeek、OpenAI、Claude、Gemini 等，用户自行配置 Key，Gita 不代理任何 LLM 调用成本。

### 4.3 Prompt 可配置

Prompt 不写死在代码中，支持用户在 `~/.gita/prompts/` 下自定义模板（`commit.md`、`pr.md`、`review.md`、`release.md`），无自定义时使用内置默认模板。

### 4.4 CLI First

优先做好命令行体验，后续再考虑 VSCode Extension、JetBrains Plugin、Git Hook、GitHub Action 等形态。

---

## 5. 范围定义

### 5.1 MVP 范围（本文档详细设计部分）

```text
gita commit
```

生成 Commit Message 的完整闭环：读取 staged 变更 → 构造上下文 → 调用 LLM → 用户确认/编辑/重新生成 → 执行 git commit。

### 5.2 后续版本（仅列目标，不在本次详细设计范围内）

| 版本 | 命令           | 目标                                            |
| ---- | -------------- | ----------------------------------------------- |
| v1   | `gita pr`      | 基于 `git diff origin/main...HEAD` 生成 PR 描述 |
| v2   | `gita release` | 基于 `git log` 生成 Release Notes               |
| v3   | `gita review`  | 基于分支 diff 生成 AI Code Review 摘要          |

### 5.3 Non-goals（MVP 明确不做）

- 不做多语言仓库自动检测（自动识别项目技术栈并调整 prompt 策略）
- 不做基于 AST 的深度语义理解或自动 commit 拆分建议
- 不做 commit message 历史学习 / 个性化风格记忆
- 不做 Git Hook / IDE 插件等非 CLI 形态
- 不做多 commit 批量生成（一次只处理一次 staged 快照）
- 不限制 LLM 重新生成次数（开源工具，用户自备 Key，成本自理）

---

## 6. 功能需求（MVP：`gita commit`）

### 6.1 整体流程

```text
读取 staged changes
    ↓
构造上下文（含 Diff 分级处理 + 可选 --hint）
    ↓
调用 LLM
    ↓
生成 Commit Message
    ↓
用户确认 / 编辑 / 重新生成 / 取消
    ↓
执行 git commit
```

### 6.2 命令与参数

```bash
gita commit [flags]
```

| 参数                            | 说明                                                              | 是否 MVP |
| ------------------------------- | ----------------------------------------------------------------- | -------- |
| `--hint <text>`                 | 传入额外上下文说明（如"这是个 hotfix"），拼入 prompt 提升生成质量 | ✅       |
| `--lang <zh-CN\|en>`            | 覆盖 config.yaml 中的默认输出语言                                 | ✅       |
| `--style <conventional\|plain>` | 覆盖默认输出风格                                                  | ✅       |
| `--provider <name>`             | 覆盖默认 LLM Provider                                             | ✅       |
| `--api-key <key>`               | 临时指定 Key，不落盘，仅本次生效                                  | ✅       |
| `--force`                       | 超大 diff 时跳过确认提示，强制生成                                | ✅       |

**使用示例：**

```bash
git add .
gita commit --hint "为兼容旧版 API 保留了废弃字段"
```

### 6.3 Diff 获取

执行 `git diff --staged` 作为变更数据的唯一来源。

### 6.4 Context Builder（Diff 分级策略）

根据 staged diff 行数自动分级，避免超大 diff 直接打满 LLM 上下文或产生高额调用成本：

| 级别   | 行数范围        | 发送内容                                                            |
| ------ | --------------- | ------------------------------------------------------------------- |
| 小型   | < 1000 行       | 完整 `git diff --staged`                                            |
| 中型   | 1000 ~ 5000 行  | 完整 diff + `--stat`                                                |
| 大型   | 5000 ~ 15000 行 | **按文件摘要压缩**（见 6.5）                                        |
| 超大型 | > 15000 行      | `--stat` + `--name-only`，并提示用户拆分 commit（`--force` 可跳过） |

**超大型变更提示示例：**

```text
本次暂存变更约 18000 行，已超出建议范围。
Gita 将仅基于文件列表生成粗略的 commit message，建议：
1. 使用 git add -p 分批暂存后再执行 gita commit
2. 或使用 --force 强制生成（质量可能下降）
是否继续？[y/N]
```

### 6.5 摘要压缩规则（大型变更适用）

对每个变更文件，提取以下信息而非全量 diff：

- 文件路径 + 变更类型（新增 / 修改 / 删除 / 重命名）
- 增删行数统计
- **改动位置摘要**：解析 `git diff` 输出中每个 hunk header（形如 `@@ -10,7 +10,8 @@ func doSomething(...)`）末尾 Git 自带的函数/类上下文文本，无需额外的 AST 解析或正则匹配源码。Git 对主流语言（C/C++、Python、Java、Go、Rust、JS/TS、Ruby、PHP、C#、CSS、HTML 等）内置了识别规则，按文件扩展名自动生效。
- 对未被 Git 内置规则覆盖的语言，该字段允许为空，自动降级为"仅文件名 + 行数统计"，不额外兜底

**抓大放小原则**：单文件改动 < 50 行时，即便整体处于"大型"分级，仍对该文件发送完整 diff；仅对改动量大的文件做摘要压缩。

### 6.6 用户交互（结果确认环节）

```text
Suggested Commit Message
─────────────────────────
feat(deployment): 初始化部署记录管理服务项目基础架构
─────────────────────────
[Y] 确认提交   [e] 编辑后提交   [r] 重新生成   [n] 取消
```

| 操作        | 行为                                                                          |
| ----------- | ----------------------------------------------------------------------------- |
| `Y`（默认） | 执行 `git commit -m "..."`                                                    |
| `e`         | 打开 `$EDITOR`（或内置输入框）供用户修改文本后提交                            |
| `r`         | 重新调用 LLM 生成，不限次数；可选携带上一次输出作为"需要规避的结果"提升多样性 |
| `n`         | 取消，不执行 commit                                                           |

### 6.7 输出格式

- 默认风格：`feat(scope): 描述`（Conventional Commit）
- 支持中文 / 英文输出，由 `config.yaml` 的 `language` 或 `--lang` 决定
- 支持 `plain`（无 type/scope 前缀）风格，由 `commit_style` 或 `--style` 决定

---

## 7. 配置设计

### 7.1 配置目录

```text
~/.gita/
├── config.yaml
└── prompts/
    ├── commit.md
    ├── pr.md
    ├── review.md
    └── release.md
```

### 7.2 config.yaml

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
  claude:
    model: claude-sonnet-4-6
    base_url: https://api.anthropic.com/v1
    api_key_env: GITA_ANTHROPIC_API_KEY

language: zh-CN
commit_style: conventional # conventional | plain
max_diff_lines: 5000
```

### 7.3 API Key 读取优先级

1. 命令行参数 `--api-key`（临时覆盖，不落盘）
2. 环境变量（`api_key_env` 指定的变量名）
3. 均未设置则报错，并提示具体缺少的环境变量名及设置示例

> 出于安全考虑，不支持在 `config.yaml` 中明文写死 `api_key`，避免误提交至 Git 仓库。

### 7.4 Prompt 模板变量

内置以下变量供模板中使用（语法：`{{variable}}`）：

| 变量            | 说明                             |
| --------------- | -------------------------------- |
| `{{diff}}`      | 完整或摘要后的 diff 内容         |
| `{{stat}}`      | `git diff --stat` 输出           |
| `{{file_list}}` | 变更文件列表                     |
| `{{hint}}`      | 用户通过 `--hint` 传入的额外说明 |
| `{{language}}`  | 输出语言                         |
| `{{style}}`     | 输出风格                         |

**模板加载优先级**：`~/.gita/prompts/commit.md`（若存在）> 内置默认模板。

---

## 8. 异常场景处理

| 场景                         | 处理方式                                                              |
| ---------------------------- | --------------------------------------------------------------------- |
| 没有 staged 变更             | 提示"没有检测到暂存变更，请先执行 git add"，不调用 LLM                |
| 当前目录不是 Git 仓库        | 提示"当前目录不是有效的 Git 仓库"                                     |
| API Key 缺失/无效            | 提示具体缺少的环境变量，并给出设置示例命令                            |
| LLM 请求超时（建议阈值 30s） | 提示超时，询问是否重试；重试沿用同一份已构建 context                  |
| LLM 返回内容为空/格式异常    | 提示"生成失败，可尝试 `-r` 重新生成"，不阻塞用户手动执行 `git commit` |
| 网络不可达                   | 提示网络错误，并提示可改用 `git commit` 手动提交                      |
| Diff 中包含二进制文件        | 自动排除二进制内容，仅在摘要中标注"包含二进制文件变更"                |

---

## 9. 非功能需求

### 9.1 性能

| 阶段                                 | 目标                                            |
| ------------------------------------ | ----------------------------------------------- |
| Git 数据读取 + Context 构建（本地）  | < 500ms                                         |
| LLM 调用等待期间                     | 显示 spinner / 进度提示，避免用户误以为程序卡死 |
| 端到端总耗时（正常网络 + 中型 diff） | 尽量 < 15 秒，超过 20 秒需明确超时提示          |

### 9.2 成本控制

- 避免直接发送超大 Diff（见 6.4 分级策略）
- 大型变更走摘要压缩（见 6.5），而非简单截断
- 重新生成不设次数限制，成本由用户自行承担（开源工具定位）

### 9.3 可扩展性

架构需保证 `gita pr` / `gita review` / `gita release` / `gita changelog` 等后续命令可复用同一套 `git` / `context` / `prompt` / `llm` 模块，无需重构核心模块。

### 9.4 安全性

- API Key 不允许以明文形式写入配置文件
- 本地不缓存/落盘用户的 diff 内容（除非用户主动开启调试日志）

---

## 10. 技术架构

```text
cmd/gita
    │
    ▼
service
    │
    ├── git       （封装 diff / status / log / show 等 Git 命令调用）
    │
    ├── context   （Diff 分级、摘要压缩、hunk header 解析、上下文拼装）
    │
    ├── prompt    （模板加载、变量渲染、默认模板 fallback）
    │
    └── llm       （统一 Provider 接口，屏蔽各家 API 差异）
```

各模块职责单一、通过接口解耦，为后续 `pr` / `review` / `release` 命令复用 `git`、`context`、`llm` 模块，仅需新增各自的 `prompt` 模板与结果解析逻辑。

---

## 11. 验收标准（MVP）

- [ ] `git add` 后执行 `gita commit`，10 秒内（正常网络）展示建议的 commit message
- [ ] 支持 `Y / e / r / n` 四种确认操作，均按预期行为执行
- [ ] `--hint` 参数生效，能观察到生成内容因 hint 内容不同而变化
- [ ] 小型 / 中型 / 大型 / 超大型四档 diff 均能正常生成，不因超大 diff 导致程序崩溃或超时无提示
- [ ] 切换 `--provider` 能正常调用对应 LLM
- [ ] 未配置任何 API Key 时，报错信息清晰指出应设置的环境变量名
- [ ] 支持 `~/.gita/prompts/commit.md` 自定义模板覆盖默认模板
- [ ] 无 staged 变更时不发起 LLM 调用，直接给出提示

---

## 12. 里程碑（建议）

| 阶段   | 内容                                                                                                      | 预估周期 |
| ------ | --------------------------------------------------------------------------------------------------------- | -------- |
| Week 1 | `git` / `context` / `prompt` / `llm` 四个核心模块打通，支持单一 Provider（如 DeepSeek）跑通小型 diff 场景 | 5 天     |
| Week 2 | 补齐 Diff 分级策略、摘要压缩、用户交互（e/r/n）、异常处理、多 Provider 支持                               | 5 天     |
| 验收   | 按第 11 节验收标准逐项自测                                                                                | 1-2 天   |

---

## 13. 风险与依赖

| 风险                                                        | 影响                       | 应对                                                       |
| ----------------------------------------------------------- | -------------------------- | ---------------------------------------------------------- |
| 不同 LLM Provider 的 API 差异（如流式返回、错误码格式不一） | 增加 `llm` 模块适配成本    | 统一接口层做好抽象，各 Provider 实现细节内聚在各自适配器中 |
| Git 内置 funcname 规则对冷门语言覆盖不足                    | 摘要压缩质量下降           | 已设计降级方案（仅文件名+行数），不阻塞主流程              |
| 用户 LLM 调用成本失控（重新生成不限次数）                   | 用户侧成本，非 Gita 侧风险 | 在 README / CLI 提示中明确告知，用户自行承担               |
