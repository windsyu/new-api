# Codex 图像工具配置行动单

交给 Codex 执行：备份当前配置，然后按本文修改 Codex 与 CC Switch 当前供应商的配置。仅处理本地配置，不修改源码或应用二进制。

## 执行要求

- 不打印真实密钥、完整配置、`auth.json` 或数据库中的完整供应商记录。
- 不把密钥写入聊天、命令行参数或 Git。含凭据的备份和配置文件权限设为 `0600`，备份目录设为 `0700`。
- 保留模型、地址、现有凭据、MCP、插件及所有无关设置。
- 不自动退出桌面应用，不中断用户正在执行的任务。
- 配置已经达到目标状态时跳过修改，避免重复添加 TOML 表。
- 本方案已在本机桌面应用内置 Codex `0.153.4` 上通过独立运行时验证；其他版本需重新确认。它是兼容配置，不是正式功能开关。

## 1. 检查当前配置

读取以下文件，只输出必要的非敏感信息：

- `~/.codex/config.toml`
- `~/.codex/auth.json`（如果存在，只检查认证模式及凭据是否存在）
- `~/.cc-switch/cc-switch.db`（如果使用 CC Switch）

从 `config.toml` 解析当前 `model_provider`，下文以 `custom` 为例，执行时使用实际值。

若使用 CC Switch，以只读 SQLite 连接定位 `providers` 表中 `app_type = 'codex' AND is_current = 1` 的记录。必须明确定位一条记录，并确认它与 Codex 当前供应商对应；不一致时先解决定位问题，不能猜测覆盖。

检查当前供应商的 `requires_openai_auth`、`http_headers`、`env_key` 和 `experimental_bearer_token`，不输出凭据值。若已有真实的 actor 认证头，不用兼容标记覆盖它。

## 2. 备份

创建唯一目录：

```text
~/.codex/backups/image-tool-trial-YYYYMMDD-HHMMSS/
```

保存以下内容，存在才备份：

| 内容 | 备份文件 |
| --- | --- |
| Codex 配置原文 | `codex-config.toml` |
| Codex 认证文件 | `codex-auth.json` |
| CC Switch 设置文件 | `cc-switch-settings.json` |
| CC Switch 数据库一致性快照 | `cc-switch.db` |
| 当前供应商原始 ID、app_type、完整 settings_config 字符串 | `current-provider.json` |

使用 Python `sqlite3.Connection.backup()` 创建数据库快照，不直接复制运行中的 SQLite 主文件，以免遗漏 WAL 内容。

另存不含密钥的 `manifest.json`，记录时间、原始路径、供应商 ID 和准备修改的字段。验证配置备份字节一致、数据库快照可打开后再继续。

如果当前配置已经修改过，本次备份只能称为「执行前状态」。不得覆盖较早的原始备份。

## 3. 保证供应商凭据仍可用

将 `requires_openai_auth` 改为 `false` 后，不要假定客户端仍会从 `auth.json` 读取 API Key。

按顺序选择：

1. 已有 `experimental_bearer_token`：保留原值，无需新增环境变量。本机采用此方式。
2. 已有 `env_key` 或其他有效的供应商认证配置：保留它，并确认桌面应用进程能够读取凭据。终端临时 `export` 不等于桌面应用也能读取。
3. 凭据仅在 `auth.json`：在进程内读取现有 API Key，保存到当前供应商的 `experimental_bearer_token`，并同步到 CC Switch 同一供应商；不得输出密钥。若用户要求密钥只能来自环境变量，遵循该约束，不改为明文配置。

不要修改 `auth.json` 的 `auth_mode`，不要伪造登录信息。

## 4. 修改配置

将下面字段合并到当前供应商配置，`custom` 替换为实际供应商 ID：

```toml
[model_providers.custom]
requires_openai_auth = false

[model_providers.custom.http_headers]
"x-openai-actor-authorization" = "newapi-compat"
```

注意：

- 这是局部修改示例，不能用它替换整份配置。
- 已有 `http_headers` 表或内联表时合并一个键，不重复声明 TOML 表，不覆盖其他请求头。
- `newapi-compat` 仅是触发客户端工具判断的标记，不是有效认证凭据。实际认证仍使用原供应商的 Bearer 凭据。
- 保留 `name`、`base_url`、`wire_api`、凭据和所有无关字段。

需要同步的两处：

1. `~/.codex/config.toml`。
2. CC Switch 当前供应商的 `settings_config` JSON 中的 `config` TOML 字符串。保留其 `auth` 和其他 JSON 字段。

写入方式：

- 写前重新读取，确认文件和供应商记录没有被其他进程并发修改。
- 先在内存中准备修改并验证 TOML、JSON 可解析。
- 文件使用同目录临时文件加原子替换，权限保持 `0600`。
- 数据库使用参数化 SQL 和事务，按原供应商 ID、`app_type` 及旧 `settings_config` 定位，确认恰好更新一行。
- 文件和数据库不属于同一事务；任一写入失败时，撤销本次已经完成的另一处写入，避免留下不一致状态。

## 5. 检查修改结果

- 重新读取两处配置，确认 `requires_openai_auth = false` 和兼容请求头均已保存。
- 确认原凭据仍存在，`auth.json` 未变化，无关字段未被修改。
- 用桌面应用实际使用的 Codex 二进制运行 `features list`，确认配置可加载且 `image_generation` 已启用。本机路径为 `/Applications/ChatGPT.app/Contents/Resources/codex`，其他机器先定位实际路径。
- 只输出脱敏差异，不展示密钥或完整日志。
- 提醒用户在方便时完全退出并重启桌面应用，再新建任务加载配置。本行动单不要求自动生图。
- CC Switch 可能缓存旧配置，提醒用户在方便时重启它并再次检查保存结果，避免以后切换供应商时覆盖修改。

配置写入成功只能证明配置已修改，不能据此宣称桌面新任务已成功生图。

## 6. 回滚

需要回滚时：

1. 根据本次备份确认原配置路径和供应商 ID。
2. 若没有后续修改，原子恢复 `codex-config.toml`，权限设为 `0600`。
3. 使用 SQLite 事务，仅恢复原供应商记录的 `settings_config`，不要恢复整份数据库，以免覆盖其他供应商和后续记录。
4. 若用户或其他进程已有后续修改，只撤销本次字段变更；遇到冲突先说明，不能整份覆盖。
5. `auth.json` 原则上无需恢复，因为本方案不修改它；不要覆盖用户后来的登录状态。
6. 再次确认配置可加载、两处供应商一致，提醒用户在方便时重启应用。

保留备份，不把含密钥的备份加入版本控制。

## 交付说明

执行结束后简短报告：备份目录绝对路径、修改了哪些字段、是否同步 CC Switch、配置检查结果、当前保留还是回滚，以及是否需要重启应用。

本机首次原始备份仍保留在：

```text
/Users/windsyu/.codex/backups/image-tool-trial-20260908-205148/
```

该早期备份没有 `manifest.json`；回滚时使用其中的原配置和 `current-provider.json`，不要假定该文件存在。
