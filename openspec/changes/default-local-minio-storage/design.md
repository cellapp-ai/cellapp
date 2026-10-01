# Design

## Context

动机见 [proposal.md](proposal.md)。`LoadConfig` 的非生产回退仍是七牛 Endpoint、`cn-east-1` 和不可用的 Bucket/凭证占位符；生产不使用这些回退。`NewStorage` 已支持 HTTP/HTTPS、path-style、Signature V4 和指定 Bucket 健康检查，并要求生产 HTTPS，继续复用即可。

`infra/compose.yaml` 仅有 PostgreSQL 与 Caddy；服务在宿主运行，因此新存储宿主入口使用 loopback。`TestLiveS3` 已提供随机对象读写、匿名 GET 拒绝、删除和清理测试，但跳过提示仍限定七牛。`.env.production.example` 使用通用 S3 密钥名，与代码不一致。

主规格没有独立 S3 配置契约，新增 `s3-storage`；仓库启动文档规则在既有 `repository-workflow` 中补充。历史归档明确选择过七牛，属于历史事实，不能改写为本轮设计。

## Goals / Non-Goals

**Goals:**
- 让新的本地环境拥有真实、持久且私有的 S3 依赖，默认值、容器初始化和文档一致。
- 统一通用凭证命名，保持既有外部配置的可读性，明确失败及回滚行为。
- 清理当前七牛默认定位并留下可审查的残留引用分类。

**Non-Goals:**
- 不迁移云对象、不创建云资源、不改个人 `.env`、不删除数据卷。
- 不改变上传/发布/网关 HTTP 协议、Storage 接口、数据库结构或客户端部署算法。
- 不在本轮实现生产 MinIO 编排，也不调整与本变更无关的旧数据库名称。

## Decisions

### 1. 本地依赖由 Compose 提供，应用只探测指定 Bucket

Compose 新增 `minio` 与一次性 `minio-init` 服务及命名数据卷。固定经过验证的 MinIO/初始化客户端镜像版本，实施时验证镜像实际可获取，不使用浮动 latest。MinIO API 绑定 `127.0.0.1:9000`，管理端口绑定 `127.0.0.1:9001`；容器内部初始化访问 `http://minio:9000`。开发默认 Region `us-east-1`、Bucket `cellapp`、Access Key `cellapp-local`、Secret Key `cellapp-local-development-only`，只用于本地依赖，服务与容器共用一致值。

初始化等待就绪采用有界重试，凭证错误或超时非零退出；创建 Bucket 使用幂等方式并关闭匿名访问，任何步骤失败都传递错误。重复运行保留对象，停止/重启不删命名卷。README 的启动步骤须等待依赖健康并检查初始化服务成功，之后启动 Go；仅执行后台 up 不能当作 Bucket 已就绪。

替代方案：由 Go 自动创建 Bucket 会扩大生产权限与职责；让开发者手工建 Bucket 会保留首次启动阻碍。两者都不采用。

### 2. 通用配置与成对兼容回退

Config 和适配器统一使用供应商无关字段。通用凭证组完整则使用该组；仅一项非空则立即报缺失变量。两项均空才检查旧 `QINIU_*` 组，规则相同；两组均空且非生产时才使用本地开发凭证，生产报缺失通用变量。不得逐字段混合新旧密钥。报错仅包含变量名，不含值。

Endpoint/Region/Bucket 仍按现有生产必填、开发回退模式读取，开发回退改成本地值。所有显式值保留。生产保持 HTTPS 限制，不增加本地回退，不自动创建 Bucket。生产示例继续注入通用密钥；本地例子只包含公开开发值，真实外部密钥仍由进程环境注入。

替代方案：立即删除旧变量会破坏既有配置；继续使用七牛字段会使通用示例不可用。因此只保留明确的环境变量兼容入口，内部字段、默认值与提示统一通用命名。

### 3. 当前说明切到通用 S3，历史引用有明确用途

以 `docs/s3-storage.md` 作为当前配置、MinIO 和真实存储验收入口。旧 `docs/qiniu-storage.md` 保留简短历史链接/迁移说明，指向新文档，避免历史验收记录断链；不再是当前推荐入口。图表 JSON 的节点 ID 与边引用同步从 `qiniu` 改为通用存储标识，HTML 对应更新，使用“私有 S3 / 本地 MinIO”描述实际边界。

子仓库 `public-boundary.json` 将存储文档排除项泛化为 `docs/`，已有允许清单不包含 docs，扩大排除不减少现有允许文件；公开包继续只有客户端材料。该单处修改需要遵循子仓库分支、检查和交付流程。

替代方案：全仓机械删除七牛字样会改写历史证据及破坏旧配置；只更新 Endpoint 会留下不一致凭证、启动流程和图表。采用按用途清理。

### 4. 七牛遗留引用清单（本轮只读盘点）

检索包含大小写不敏感的 `qiniu|qiniucs|七牛`，覆盖主仓库、已初始化子仓库及忽略的项目文本；排除 Git 内部、依赖、生成产物、测试结果、本地秘密目录及个人 `.env`。个人环境配置不读取/输出值，由迁移说明提醒自行检查，不能宣称已清理机器全部配置。

| 命中文件 | 实施处理 |
| --- | --- |
| `apps/server/internal/hosting/config.go`、`storage.go` | 改默认 Endpoint/Region/Bucket 与内部字段，旧环境变量只保留兼容分支 |
| `apps/server/internal/hosting/core_test.go` | 改通用凭证测试，新增明确命名的旧变量兼容场景 |
| `apps/server/internal/hosting/storage_test.go` | 跳过提示改为真实 S3 通用说明 |
| `.env.example` | 默认使用本地 MinIO，凭证说明使用通用变量 |
| `README.md`、`AGENTS.md` | 更新架构、开发启动与真实存储验收描述 |
| `docs/qiniu-storage.md` | 改为旧路径历史入口与迁移说明；当前内容迁至通用文档 |
| `docs/diagrams/cellapp.architecture.json`、`cellapp-deployment.workflow.json` | 更新节点、边与显示名称 |
| `docs/diagrams/cellapp-architecture.html`、`cellapp-deployment-flow.html` | 与图表源同步，核对可见文本及引用 |
| `skills/public-boundary.json` | 泛化存储运维排除项，验证客户端边界 |
| `docs/verification.md` | 保留历史七牛验收事实；旧存储链接通过保留入口继续有效 |
| `docs/console-frontend-verification.md`、`repository-workflow-verification.md`、`cellapp-naming-verification.md`、`web-control-plane-verification.md` | 保留对应轮次未执行/已执行的历史说明 |
| `openspec/changes/archive/2026-09-22-static-app-hosting/design.md`、`tasks.md` | 保留原设计和任务事实，不重写归档 |

本变更规划文件自身会提及七牛清理和兼容用途；实施完成后在本轮验收记录重跑盘点，逐处标明剩余引用的理由。`.env.production.example` 无七牛命中，但其通用变量与代码一致性必须验证。现有 MinIO Go 依赖名称是 SDK 名称，保留。

## Risks / Trade-offs

- [本地默认切换可能让旧环境访问空 Bucket] → 不修改个人配置；升级前保留原 Endpoint/Region/Bucket/密钥，不将默认切换解释为自动迁移。
- [本地端口被占用或容器镜像不可获取] → 明确诊断并允许显式覆盖本地配置；不停止无关进程，记录未完成的真实验收。
- [初始化重复执行改变既有匿名策略] → 初始化只面向受控本地实例，私有是项目约束；不指向或修改生产资源。
- [开发凭证误用于生产] → 生产缺项拒绝启动、只接受 HTTPS，文档说明开发值用途与真实密钥注入边界。
- [供应商文字盘点掩盖历史事实] → 依据清单逐项分类，保留原验收和归档，不用历史结果证明 MinIO 验收。
- [子模块未推送阻碍主仓库交付] → 先完成子仓库提交与获授权推送及远程可获取性确认，再提交最终 gitlink；没有推送授权时保留工作并报告交付缺口。

## Migration Plan

1. 实施前复查两个仓库工作区；仅修改本变更文件，子模块 detached HEAD 时先建开发分支。
2. 完成通用配置和兼容回退、Compose、文档及图表；保持现有生产显式配置逻辑。
3. 用独立 Compose 项目/数据卷验证首次初始化、重复初始化和重启保留对象；真实存储测试仅使用隔离测试 Bucket 和随机对象。
4. 验证生产缺项、HTTP 拒绝、新旧凭证优先级及缺项失败；执行 Go 检查、真实 MinIO 冒烟、必要的隔离数据库 race 和跨服务浏览器回归。现有浏览器测试使用存储替身，另执行真实 MinIO 支撑的 CLI 发布/访客鉴权/更新/删除检查。
5. 更新当前资料及本轮验收记录，核对剩余七牛引用、文档链接与图表；完成客户端独立检查及包边界检查。
6. 必要验收通过后同步规格、归档并按项目规则提交；跨仓库遵循先子仓库且远程可获取、后主仓库 gitlink 的顺序，推送按已有授权执行。

旧环境保留原后端配置即可原位升级，不切换存储数据。回滚服务时恢复旧版本和对应旧凭证变量配置，保留数据库、Bucket 和对象卷；不要使用删除卷操作。若主动将后端切换到 MinIO，需要另行设计对象及数据库引用的一致迁移，不属于本变更。
