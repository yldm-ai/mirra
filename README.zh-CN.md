<p align="center">
  <img src="apps/mirra/assets/Icon.png" alt="" width="96">
</p>

<h1 align="center">Mirra 镜衣</h1>

<p align="center">
  <a href="README.md">English</a> |
  <strong>简体中文</strong>
</p>

<p align="center">
  从三个参考图库里挑一张脸、一套服装和一个姿势，交给图像模型画出它们描述的那个人。用 Go 和 <a href="https://github.com/nsgo-dev/nsgo">NSGo</a> 写成，界面由各平台自己的原生控件绘制；同一份代码构建 macOS、iOS、Android 和 Windows 版本。
</p>

<p align="center">
  <a href="https://mirra.yldm.ai/">官网</a>
</p>

<p align="center">
  <a href="screenshots/promo.mp4"><img src="screenshots/promo.webp" alt="27 秒介绍：同一个造型从街道换到薰衣草田，在 App 里选脸、选场景，再换场景、风格和服装" width="800"></a>
</p>

<p align="center">
  <img src="screenshots/phone-home-zh-Hans.png" alt="手机上的工作室：今日造型和三个要素" width="260">
  <img src="screenshots/phone-me-zh-Hans.png" alt="我的：App 信息和 AI 服务" width="260">
</p>

## 功能

- **人脸、服装、姿势。** 选一张脸（AI 生成，CC0）、一套服装（没有人穿的衣服）和一个姿势（无脸的灰色人偶），App 让图像模型生成一张具备这些的成年人照片；双人姿势生成两位成年人，人脸和服装用在主要的那位身上。只有姿势是必选的。
- **可选的背景。** 默认由风格决定场景；也可以从 App 自带的 130 张背景里选一张，以自然风景和城市街景为主，另有少量室内，都是没有人物的公有领域照片，人物会被放进这个场景。
- **两种布局。** 窗口窄于 900pt 时是手机布局：底部有工作室、衣橱、作品、我的四个标签，造型分三步选好。更宽时是桌面工作室：左边是素材库，中间是可缩放的画布（本次造型或生成结果），右边是汇总和「生成造型」，AI 服务设置在弹窗里。
- **贴合图库的筛选。** 人脸按发长，服装按季节和场合，姿势按机位。内衣、性感服装和暴露或双人姿势收在 18+ 开关后面，第一次打开前先确认年龄。
- **结果由你决定去留。** 新生成的图片要点「保存到作品」才会入库。可以再生成，也可以一键给同一张脸、同一个姿势换一套衣服；可以收藏、导出、删除。
- **衣橱。** 用 ♡ 收藏喜欢的服装，上传自己的衣服照片（平铺或挂拍），随时看到当前正在穿的那套。
- **造型册。** 保存的作品带编号，可以收藏，再次打开时会带回生成它的那套造型。
- **主流服务商，或你自己的服务。** 预置了 OpenAI、Google Gemini、火山方舟、xAI、OpenRouter、EveryAPI、阿里云百炼和 Azure OpenAI，每家都带 logo 和获取 Key 的链接；也可以用 stable-diffusion.cpp 或任何 OpenAI 兼容的地址，Key 可以不填。每个服务商各自保存 Key 和设置。生成前可以先测试连接；Key 存在系统钥匙串里。
- **补充要求。** 写一句光线、背景之类的要求，随 prompt 一起发送。
- **不用人像照片。** App 不读取任何人的照片：姿势库和服装库的条款禁止用它们的图片表现真实的人。上传的只能是衣服照片。
- 界面支持英文、简体中文、繁体中文、日语、韩语、西班牙语、法语、德语、巴西葡萄牙语、俄语和意大利语，可在设置里选择或跟随系统。

## 素材来源

| 槽位 | 图库 | 条款 |
| --- | --- | --- |
| 姿势 | [POSE-DB-01](https://ai-pose-gallery.pages.dev) | 免费使用，包括商用；禁止再分发，禁止批量或用工具自动下载 |
| 服装 | [COSTUME-DB-01](https://ai-costume-gallery.pages.dev) | 同上 |
| 人脸 | [FACE REFERENCE](https://ai-portrait-gallery.pages.dev) | CC0 |

图片归各图库作者所有。本仓库不包含任何一张，App 也遵守图库的条款：

- 每次都在线读取图库的目录。浏览时缩略图从图库加载，和用浏览器看一样；除了系统的图片缓存，App 自己不保存副本。
- 只有按下「生成」时才下载原图，而且只下载你选中的那几张，每张一次。

姿势库和服装库要求不要用它们的图片表现真实的人或未成年人，也不要用于违反公序良俗的用途；人脸库要求不要用来冒充真实的人。用这个 App 做出来的东西同样受这些条款约束，详见 [NOTICE.md](NOTICE.md)。

## 工作原理

```
图库目录（HTML / JS）  ──►  条目：id、分类、标题、缩略图、原图
                                │
    选人脸、服装（或你自己的衣服）和姿势
                                │
     原图（各下载一次）+ prompt
                                │
            服务商的生图接口（图片按顺序内联发送）
                                │
       图片或链接  ──►  先展示；点「保存到作品」才入库
```

图片按姿势、服装、人脸的顺序发送，prompt 按这个顺序给它们编号，并说明每张的用途：人偶的姿势和机位、要穿上的服装、要用的脸和发型。

代码地图以及代码遵守的边界见 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 运行

[最新发布](https://github.com/yldm-ai/mirra/releases/latest)里有 macOS 磁盘映像、Windows 安装程序、Android APK（发布了的话）和网页版；网页版也可以直接在 [mirra.yldm.ai/app/](https://mirra.yldm.ai/app/) 打开。想自己构建：

仓库里每个 App 放在 `apps/` 下；Mirra 的代码、Go 模块和打包配置（`nsgo.yaml`）都在 `apps/mirra`，Go 和 NSGo 命令在那里运行。

需要 Go 1.27.4 或更高版本。在 Mac 上（也能构建 iOS 和 Android 版本）：

```sh
git clone https://github.com/yldm-ai/mirra
cd mirra/apps/mirra
go tool nsgo dev                     # 本机
go tool nsgo dev -lang zh-Hans       # 中文界面
go tool nsgo dev -target ios_simulator
go tool nsgo dev -target android     # 已连接的设备或模拟器
go tool nsgo build                   # 打包 nsgo.yaml 里列出的所有平台，输出到 dist/
```

`go tool nsgo doctor` 会告诉你这台机器能构建、运行哪些平台。

### 配置生图接口

手机上打开「我的」，桌面上点右上角的设置按钮，选一个服务商，填好 Key 后点「测试连接」。模型在 Key 下面选，选项是这把 Key 能用的图像模型，填好 Key 后自动从服务商获取，默认用服务商自己的。接口地址和尺寸也默认用服务商自己的，收在「高级设置」里；只有要换地区或网关时才需要改，「恢复默认」可以一键改回。阿里云百炼和 Azure OpenAI 没法用 Key 列出模型，仍需手动填写。Azure OpenAI 和自定义服务的接口地址直接显示在 Key 旁边。

| 服务商 | 接口 | 默认模型 |
| --- | --- | --- |
| OpenAI | `POST /images/edits`，multipart | `gpt-image-2` |
| Google Gemini | `POST /models/{model}:generateContent`，图片内联 | `gemini-nano-banana-2.1` |
| 火山方舟 | `POST /images/generations`，图片用 data URI；每次调用出一张 | `doubao-seedream-5-5-261028` |
| xAI | `POST /images/edits`，JSON | `grok-imagine-image-2.0` |
| OpenRouter | `POST /images`，input references | `bytedance-seed/seedream-4.5` |
| EveryAPI | OpenAI 的 `POST /images/edits`，multipart；中国大陆可改用 `https://api-cn.everyapi.ai/v1` | `gpt-image-2` |
| 阿里云百炼 | 千问图像编辑的多模态生成接口；最多 3 张图 | `qwen-image-2.0-pro` |
| Azure OpenAI | 在你资源的 `/openai/v1` 上调用 OpenAI 的编辑接口，`api-key` 请求头 | 你的部署名 |
| stable-diffusion.cpp | 它的服务端提供的 OpenAI 兼容 `/images/edits` | `sd-cpp-local` |
| Magpie | 本机网关提供的 OpenAI 兼容 `/images/edits`，可用你已登录账号的图像模型 | `gpt-image-2` |
| 自定义 | 任何 OpenAI 兼容的 `/images/edits` | `gpt-image-2` |

Key 存在 macOS / iOS 钥匙串、Android Keystore 或 Windows 凭据管理器里。尺寸可选竖版（`1024x1536`）、方形（`1024x1024`）或横版（`1536x1024`）；只接受宽高比的服务商会用对应的比例，比如 `2:3`。

测试连接会请求服务商的模型列表；火山方舟和阿里云百炼没有这个接口，就发一个空请求，Key 有效时会返回 400。每按一次「生成」是一次调用（火山方舟和 Gemini 是每张图一次），云端服务商会按次计费。各家请求都按官方 API 文档实现，并在测试里的模拟接口上验证过；还没有逐一对真实服务测试，欢迎反馈各服务商的实际表现。

## 平台

| 平台 | 状态 |
| --- | --- |
| macOS | 可以运行；在真实窗口里测试过。发布里的磁盘映像用 Developer ID 签名并经过公证；App 通过 Sparkle 自动更新 |
| Windows | 发布里的安装程序用代码签名证书签过名，App 通过 WinSparkle 自动更新；还没有在设备上运行过 |
| Android | 发布里带有用正式密钥签名的 APK，App 启动时会提示安装更新的版本；还没有在设备上运行过 |
| iOS | 从源码构建（`go tool nsgo dev -target ios_simulator`）；还没有在设备上运行过 |
| Linux | 还不在构建目标里；NSGo 支持，这里没测过 |
| 网页 | 在 [mirra.yldm.ai/app/](https://mirra.yldm.ai/app/) 打开，和官网一起构建发布。浏览器里没有安全存储和文件系统，所以 Key、作品和上传的衣服只保存在当前页面，想保留的图片要导出 |

## 隐私

- 只有按下「生成」或「测试连接」时，App 才会访问你配置的服务，发送的只有你选的图片（图库图片或你的衣服照片）和 prompt。
- 你自己运行的网关（如 Magpie）会把请求转发给背后的服务，费用计入那个服务的套餐。
- 你上传的衣服照片只存在 App 的数据目录里。
- 图库会看到你打开的缩略图和原图的请求，和用浏览器访问一样。
- 新生成的图片在保存前单独存放；保存到作品后一直留在 App 的数据目录里，直到你删除。

## 开发

在 `apps/mirra` 目录下：

```sh
go tool nsgo test .                                    # headless 测试，图库和接口都用本地模拟
go tool nsgo test -target macos .                      # 同一套测试在真实窗口里跑
go tool nsgo check -target macos,ios,android,windows,web   # 对每个平台做 vet
go run ./tools/appicon                                 # 重新绘制图标
```

测试不访问网络：`apps/mirra/fixtures_test.go` 按图库的格式提供虚构的页面数据，以及一个模拟的 `/images/edits`。

如果哪个图库改了页面格式，选择页会显示「图库加载失败」和解析器的报错；解析代码在 `apps/mirra/gallery.go`。

### 官网

[官网](https://mirra.yldm.ai/)在 `apps/website`：React 上的 TanStack Start 和 Router，加 Tailwind 和 react-i18next，用 Vite+ 和 Bun 构建，预渲染成中文（`/`）、英文（`/en/`）和日文（`/ja/`）三个静态页面。文案在 `src/i18n/copy.ts`；版本号、Star 数、贡献者和最新发布在构建时从 `apps/mirra/nsgo.yaml` 和 GitHub API 读取。每个页面都带 canonical 地址、语言版本链接、Open Graph 和 Twitter 卡片以及结构化数据，构建时还会生成 `sitemap.xml`。在 `apps/website` 目录下：

```sh
bun install
bun run dev         # http://localhost:5175/
bun run typecheck
bun run build       # 输出到 dist/client
bun run og          # 从 dev server 的 /og/<语言>/ 卡片重新截出 public/og-<语言>.png
```

推送到 `main` 且改动了官网、App 或截图时，`.github/workflows/website.yml` 会把它发布到 GitHub Pages，App 的网页版（在 `apps/mirra` 里 `go tool nsgo build -target web`）放在 `app/` 下。

### 发布

`.github/workflows/ci.yml` 在 PR 和 `main` 上跑测试、对每个平台做 vet、构建所有安装包和官网。`main` 上通过后，`.github/workflows/release.yml` 按上一个 tag 以来的 Conventional Commits 决定版本：`feat` 升次版本号，`fix` 和 `perf` 升修订号，`docs`、`chore` 和 `website` scope 不发版。它把版本号写进 `apps/mirra/nsgo.yaml`，把 `CHANGELOG.md` 的 Unreleased 一节改成这个版本的一节（Unreleased 为空时按提交生成），提交并打 `vX.Y.Z` tag，构建安装包（`.github/workflows/package.yml`），连同校验和与来源证明一起发布，再重新构建官网，让下载卡片指向这些文件。在 `main` 上手动运行 Release 可以强制升版本。macOS 安装包用组织的证书签名并公证，Windows 安装包也用组织的证书签名；Android APK 用 `ANDROID_KEYSTORE`、`NSGO_KEYSTORE_PASSWORD`、`NSGO_KEY_PASSWORD` 这几个 secret 里的正式密钥签名，因为已安装的 App 只能升级到同一个密钥签名的 APK。

已安装的 App 会从最新发布自动更新。发布构建用 `NSGO_UPDATES_KEY` secret 里的私钥给每个安装包签名，并上传各桌面平台和 Android 的 `appcast-<target>.xml`；`apps/mirra/nsgo.yaml` 的 `updates.key` 是对应的公钥，App 只安装这把密钥签过名的更新。macOS 上的 Sparkle 和 Windows 上的 WinSparkle 每天检查一次，Android 在 App 启动时检查，「关于」里的「检查更新」可以立即检查。私钥务必保管好：丢了它，已安装的 App 就再也无法更新。0.7.0 及以前的版本没有更新器，需要手动重装一次。

## 许可

代码使用 [MIT](LICENSE) 许可。图库的图片不属于本仓库，仍受其所有者的条款约束。
