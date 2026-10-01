# 应用数据访问

Cellapp 仍然只托管静态文件，不运行用户后端，也不转发数据库请求。所有者可以把自有 Supabase 项目绑定到应用，让已经通过分享密钥进入页面的访客在浏览器里共用一份数据。

## 绑定

在控制台应用详情填写 Supabase URL 和 anon/publishable key，或在本机执行：

```sh
cellapp data <app-id> --provider supabase --url https://PROJECT.supabase.co --anon-key <anon-or-publishable-key>
```

查看当前绑定：`cellapp data <app-id>`。清除：`cellapp data <app-id> --clear`。配置保存在控制元数据中，不必重新发布静态文件。不要把 service role 或 `sb_secret` 交给 Cellapp，也不要写入 `cellapp.json`。

## 页面读取

访客解锁应用后，从同一 origin 读取公开配置：

```js
const response = await fetch('/_hosting/data', { credentials: 'same-origin' });
if (!response.ok) throw new Error('data backend is not bound');
const { url, anonKey } = await response.json();
const supabase = createClient(url, anonKey);
```

未绑定返回 `404 data_not_configured`。无访问会话时不会返回配置。

## 边界

- 同一应用的全部分享密钥访客共用该 Supabase 项目中的数据；Cellapp 不创建应用内账号，也不把分享密钥换成 Supabase JWT。
- 分享密钥保护的是页面。anon key 对能打开应用的人可见，也可以被复制后在应用域外调用。敏感行需要由该项目的 RLS 约束；共享数据集通常意味着 anon 角色可读写你打算共享的表。
- 浏览器 localStorage 仍按应用 origin 隔离，不会跨设备同步。
- Cellapp 不代建 Supabase 项目，不代理 Rest/Realtime，不在网关执行 SQL。
