# application-api-versioning Specification

## Purpose
让 Cellapp 托管应用显式选择可追溯的运行时 API 契约，使已发布的前端在服务端持续演进时保持预期行为，并让 Agent 能定位版本差异。

## Requirements

### Requirement: Explicit application runtime API version
应用运行时 API SHALL 接受等价的 `v=major.minor` 与 `v=yymmdd` 别名选择同一固定契约，两个别名 MUST 一对一映射，且一个日期别名 MUST NOT 指向多个契约。相同契约的响应字段、含义、支持操作和错误语义 MUST 保持兼容；无 `v` 的请求 SHALL 使用服务端最新实现。未知、格式错误或已停用的别名 MUST 返回可识别的版本错误，不得静默回退。

#### Scenario: Pinned and unversioned requests
- **WHEN** 同一应用分别请求 `/_cellapp/me?v=1.1`、`/_cellapp/me?v=261022` 和不带 `v` 的 `/_cellapp/me`
- **THEN** 前两者使用同一固定契约，后者使用当前最新实现，且三者均执行当前访问检查

#### Scenario: Unsupported version
- **WHEN** 请求携带未知、无效或已停用的 `v`
- **THEN** 接口明确返回版本错误及所请求的版本，不把请求当作最新版处理

### Requirement: Configurable stable default for new builds
平台 SHALL 公布当前默认稳定的应用 API 契约及其两个别名，并只允许已支持的固定契约成为默认值。新构建未显式指定版本时 SHALL 固化当时默认稳定契约的 `major.minor` 别名；默认值后续变化 MUST NOT 改变已生成的静态产物的版本选择。

#### Scenario: Default advances after build
- **WHEN** 应用使用默认稳定版构建完成后，平台将默认值改为新版本
- **THEN** 该应用既有静态产物仍请求构建时选择的版本，新构建才采用新默认值

### Requirement: API selection follows static rollback
应用前端的固定 API 版本 SHALL 由本次静态产物携带，而不从可变的应用级设置或部署数据库记录在请求时推断。恢复旧静态产物 MUST 同时恢复其 API 版本选择；已经加载的页面 SHALL 继续使用自身固化的版本。

#### Scenario: Restore previous static release
- **WHEN** 应用从使用新 API 版本的静态发布恢复为旧发布
- **THEN** 新加载的旧页面请求旧发布固化的版本，已打开页面不因发布切换而自动改用另一版本

### Requirement: Agent-readable version history
平台 SHALL 提供无需登录即可读取的版本说明与 changelog，逐个固定契约列出等价的 `major.minor` 和 `yymmdd` 别名、状态、适用接口、行为及错误变化、迁移方法和停用信息，并区分 API 契约版本、SDK 包版本、`/_cellapp/db` 路径和数据表清单版本。

#### Scenario: Diagnose an app error
- **WHEN** Agent 获得应用请求的 API 版本及错误码
- **THEN** Agent 可从公开文档定位该版本的支持范围、相关变化和可执行的迁移说明
