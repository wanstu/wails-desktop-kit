# 配置目录与安全配置

[返回首页](../README.md)

Kit v0.4.0 起提供统一的应用配置目录与 Secure Config。

## 配置目录

Kit 统一采用：

~~~text
$XDG_CONFIG_HOME/<app-id>
~~~

当 `XDG_CONFIG_HOME` 未设置时：

~~~text
~/.config/<app-id>
~~~

这个约定在 Windows、Linux、macOS 上一致。Windows 不再因为 `os.UserConfigDir()` 默认落到 AppData 而产生另一套路径。

Go API：

~~~go
import kitpaths "github.com/wanstu/wails-desktop-kit/paths"

dir, err := kitpaths.ConfigDir("ssh-client")
if err != nil {
    return err
}

// 需要实际创建时：
dir, err = kitpaths.EnsureConfigDir("ssh-client")
~~~

App ID 必须是稳定的 ASCII 路径安全标识，只允许字母、数字、`-`、`_`、`.`，不能以 `.` 开头，也不能包含 `..`。App ID 一旦用于 Secure Config，不应随产品显示名称变化。

这是一项**显式使用的路径约定**。仅仅升级 Kit 不会扫描或搬迁已有应用的 AppData、旧 `os.UserConfigDir()`、工作目录配置等位置。消费者应在自己的版本迁移中决定何时迁入 `~/.config/<app-id>`，并在成功后再清理旧数据。

## 为什么普通配置和 Secret 要分开

普通设置可以继续保存为可读 JSON，例如：

~~~text
~/.config/ssh-client/settings.json
~~~

里面适合保存：

- Connection Profile 名称、Host、Port、Username。
- Theme mode / Theme Pack。
- 分组、标签、窗口与交互设置。
- 私钥文件**路径引用**。

不要直接保存：

- SSH 密码。
- 私钥口令。
- API Token。
- Bearer Token。
- Cookie / Session Secret。
- FRP 或其他服务的敏感认证值。
- 应用自己管理的私钥原文。

这些值应由 `secureconfig` 保存，普通 JSON 只保留逻辑引用，例如：

~~~json
{
  "auth": {
    "mode": "password",
    "credential_ref": "profile_abc/password"
  }
}
~~~

## Secure Config

~~~go
import "github.com/wanstu/wails-desktop-kit/secureconfig"

store, err := secureconfig.New("ssh-client")
if err != nil {
    return err
}

err = store.Put("profile_abc/password", []byte(password))

password, err := store.Get("profile_abc/password")

err = store.Delete("profile_abc/password")
~~~

也支持加密 JSON：

~~~go
type Credential struct {
    Token string `json:"token"`
    Key   string `json:"key"`
}

err := store.SaveJSON("api/main", Credential{
    Token: token,
    Key: key,
})

var saved Credential
err = store.LoadJSON("api/main", &saved)
~~~

## 存储结构

应用目录示例：

~~~text
~/.config/ssh-client/
├── settings.json
├── known_hosts.json
└── secure/
    ├── <sha256-name>.json.enc
    └── <sha256-name>.json.enc
~~~

Secure Config 使用：

- 随机 256-bit 主密钥。
- AES-256-GCM。
- 每次写入使用随机 nonce。
- App ID + logical secret name 作为 authenticated additional data。
- 逻辑 secret name 使用 SHA-256 作为磁盘文件名，避免直接泄露键名。
- 密文采用同目录临时文件 + 原子替换。
- secure 目录尝试使用 0700、密文文件尝试使用 0600。

主密钥**不写入配置目录**，而是保存在操作系统安全凭据库：

| 平台 | 系统存储 |
| --- | --- |
| Windows | Windows Credential Manager |
| macOS | Keychain |
| Linux | Secret Service |

Kit 当前通过 `github.com/zalando/go-keyring` 访问这些平台能力。

如果系统凭据库不可用，Secure Config 返回错误，不会自动退化为明文文件、固定内置密钥或和密文同目录的 key 文件。

## 威胁模型

Secure Config 主要解决：

> 攻击者只复制走 `~/.config/<app>` 时，不能直接读取密码、Token 或其他 Secret。

它**不**承诺防御已经控制当前用户会话的恶意程序。若攻击者能以同一用户身份调用系统凭据库、注入应用进程或读取进程内存，应用本身能访问的 Secret 也可能被访问。

另外：

- Go 无法保证 Secret 从进程内存中被可靠、安全地清零。
- 删除密文文件不等价于物理介质上的“安全擦除”，尤其是 SSD。
- 仅保存私钥路径不会加密原私钥文件；如果应用选择托管私钥原文，应把原文交给 Secure Config，或依赖用户自己的 OpenSSH 文件权限/硬件密钥。
- 系统凭据库中的主密钥被删除后，已有密文不会被 Kit 静默使用新 key 覆盖；读取会返回 master-key-missing 错误，避免造成不可逆的误判和数据覆盖。

## 应用建议

对大多数桌面应用建议：

~~~text
settings.json
  ↓
只放普通配置和 credential_ref

secureconfig
  ↓
放真正的 Secret

系统 Credential Store
  ↓
只放随机 master key
~~~

这样一套机制可以同时用于 SSH Client、FRP Client、ADM Secret Header 等消费者，而不需要每个应用重新设计自己的加密格式。
