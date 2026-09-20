# Wails Desktop Kit Roadmap

[返回首页](../README.md)

核对日期：**2026-09-20**。本页记录 Kit 的未来方向与设计边界，不等同于已发布能力；实际版本内容以 Release 和对应 tag 文档为准。

## 近期：桌面基础能力继续下沉

### 1. 原生 Clipboard

**v0.8.1 已落地基础能力。** 目标是让 ADM、FRP、SSH、Nginx Manager 等消费者不再各自依赖 WebView 的 `navigator.clipboard` / `document.execCommand('copy')`。

已完成：

- Kit Controller 提供系统剪贴板读写 API，底层复用 Wails 原生 Clipboard runtime。
- `/desktopkit/runtime.js` 提供统一前端调用入口；桌面优先走原生剪贴板，普通 Web 预览才降级到浏览器 API。
- 复制失败以 error/rejected promise 返回，不静默吞掉。

后续：

- 在 Windows / Linux / macOS 消费者中补齐实际桌面 smoke 验收。
- ADM、FRP、SSH、Nginx Manager 逐步迁移到 Kit helper，删除各自重复的 Clipboard 兼容代码。

### 2. Windows Tray Tooltip

已确认问题在 Windows 原生托盘链路，不是业务应用漏传标题。Kit 已有 `Tray.Tooltip -> Config.Title` 默认回退。

调查与修复重点：

- `Shell_NotifyIconW`
- `NOTIFYICON_VERSION_4`
- `NIF_TIP`
- `NIF_SHOWTIP`
- Windows systray 后端对 Tooltip buffer / update 时机的处理

目标是在 Kit 一次修复，消费者升级 Kit 后统一受益，而不是在 ADM/FRP/SSH/Nginx Manager 分别打补丁。

## 中期：securetransport

目标：把桌面管理类应用的远程通信安全基线统一到 Kit，而不是每个产品自行拼装 TLS。

建议模块：`securetransport`。

首阶段能力：

- 远程连接默认要求 HTTPS。
- 明文 HTTP 默认只允许 localhost / loopback。
- 标准 TLS 证书链与主机名严格校验。
- Private CA / 自定义 CA bundle。
- mTLS：客户端证书 + 服务端证书双向认证。
- Certificate Pinning，支持可轮换的 pin 集合而不是单一永久 pin。
- 与 `secureconfig` 集成保存客户端私钥、Token、证书口令等敏感材料。
- 证书、Key、pin 的显式轮换与过渡期策略。
- 明确的连接诊断：TLS 版本、证书主体、issuer、到期时间、pin/mTLS 状态；不泄露私钥。

Nginx Manager 当前“远程必须 HTTPS、HTTP 仅 loopback”的做法可以作为首个消费者基线。

### 非目标

- 不自己实现 TLS。
- 不发明新的密码算法、KDF、签名方案或证书格式。
- 不因为支持 Private CA 就默认跳过主机名校验。
- 不提供“忽略证书错误继续连接”的常规产品模式。

## 后期：应用层端到端加密（E2EE）

E2EE 是独立于 HTTPS/mTLS 的协议级能力。只有在威胁模型明确要求“即使 TLS 在反向代理 / 网关处终止，中间服务器仍不能看到管理载荷”时才启用。

设计原则：

- 先完成 HTTPS + mTLS + pinning；E2EE 不替代 TLS。
- 使用成熟、公开审计的密码原语和协议设计，不自创算法。
- 优先评估基于标准 AEAD 与成熟密钥协商原语的方案。
- 会话必须包含身份认证、重放保护、消息序号 / nonce 管理和协议版本。
- 必须支持密钥轮换、设备撤销和恢复流程。
- 长期身份密钥进入 `secureconfig`，不落明文配置。
- 协议版本不兼容时应 fail closed，并提供明确升级提示。
- 调试日志不得记录明文业务载荷、会话密钥或长期私钥。

### E2EE 进入实现阶段前必须先完成

1. 威胁模型文档：保护谁、信任谁、不信任谁。
2. 是否真的需要隐藏反向代理可见内容。
3. 客户端与服务端身份模型。
4. 首次配对 / 信任建立方式。
5. 轮换、撤销、丢失设备恢复方案。
6. 协议版本升级与兼容策略。
7. 独立安全 Review 与测试向量。

## 建议版本节奏

| 阶段 | 建议范围 |
| --- | --- |
| v0.8.1 | Clipboard Runtime/helper + 安全随机 Secret + Roadmap |
| 后续小版本 | Windows Tray Tooltip 原生修复 + 对应 Windows smoke tests |
| 后续功能版本 | `securetransport`：HTTPS policy、Private CA、mTLS、pinning、secureconfig 集成 |
| 独立大功能版本 | E2EE 协议、配对、轮换、版本协商与安全 Review |

## 消费者迁移顺序

建议先用实际需求驱动 Kit：

1. ADM：Clipboard、Gateway 连接安全和协议兼容提示。
2. Nginx Manager：作为 `securetransport` 首个远程管理消费者。
3. SSH Client：复用 secureconfig / Clipboard，并验证证书与主机身份类 UX。
4. FRP Client / Service Manager：迁移通用远程连接策略。
5. 其他 Wails 应用按需升级，不强制一次迁移全部能力。
