# fastpve
One click to run vm in PVE. 对应论坛帖子：https://www.koolcenter.com/t/topic/7777

可以在 PVE 上面一键下载并安装 Windows，iStoreOS，Docker，DD 镜像等等系统。

### This script is meant for quick & easy install:
#### via curl
```Bash
bash -c "$(curl -sSL https://www.linkease.com/rd/fastpve/)"
```
#### via wget
```Bash
bash -c "$(wget --no-check-certificate -qO- https://www.linkease.com/rd/fastpve/)"
```
##### Or
```Bash
cd /tmp; wget --no-check-certificate -O fastpve-install.sh https://www.linkease.com/rd/fastpve/; bash ./fastpve-install.sh
```

### DD 镜像安装

支持从 URL 下载任意 raw 磁盘镜像（.img / .img.gz / .img.xz / .img.zst），直接导入为虚拟机磁盘。

**交互式安装（fastpve 菜单）**：

1. 启动 `fastpve`，选择 `6、安装DD镜像`
2. 选择已有 .img 文件，或输入 URL 下载新镜像
3. 选择 BIOS 模式：UEFI (OVMF，推荐) 或 SeaBIOS (传统)
4. 配置 CPU 核数、内存、磁盘大小
5. 选择「下载并安装」或「仅下载」

**命令行下载（fastpve-download）**：

```bash
# 下载 DD 镜像
fastpve-download dd --url https://example.com/windows11.img.gz

# 指定路径和 BIOS 模式
fastpve-download dd \
  --url https://example.com/system.img.xz \
  --iso-path /var/lib/vz/template/iso \
  --cache-path /var/lib/vz/template/cache \
  --bios uefi

# 断点续传
fastpve-download dd --resume
```

**支持的压缩格式**：`.gz` (gzip)、`.xz`、`.zst`/`.zstd`，以及未压缩的 `.img` 文件。

### 镜像来源

本项目默认会从尽量尝试从官方地址下载 ISO，如果下载失败，则回退到：https://github.com/orgs/kspeeder/packages 这里下载。
不会对任何镜像进行任何的修改，也欢迎监督。

#### win7x64

* https://files.dog/MSDN/Windows%207/en_windows_7_ultimate_with_sp1_x64_dvd_u_677332.iso
* https://archive.org/details/Win7UltimateSP1CHS

## 编译代码

* make build

## ⚠️ 免责声明

- **合法合规使用**：本存储库旨在为大家方便安装虚拟机。使用者应严格遵守所在司法辖区法律法规及相关平台服务条款，任何非法用途的法律责任由使用者自行承担
- **非关联性与独立责任**：本存储库与各第三方平台不存在任何隶属、代理或合作关系。任何基于本存储库的 fork、二次开发、再分发或衍生版本均由其维护者独立承担全部责任；作者、维护者及贡献者不对衍生存储库的任何行为或后果承担法律或连带责任
- **无担保与免责条款**：在适用法律允许的最大范围内，本存储库按“现状（AS IS）”提供，不提供任何明示或暗示担保（包括但不限于适销性、特定用途适用性、非侵权等）。对因使用本存储库而造成的任何直接或间接损失（包括但不限于数据丢失、业务中断、利润损失等），作者、维护者及贡献者不承担任何责任
- **风险自担原则**：使用者应自行评估使用风险，确保其使用行为合法合规，不侵犯第三方权益，不得将本存储库用于任何违法、侵权、恶意或不当用途
- **第三方平台合规**：使用者应遵守相关平台的服务条款、API 使用政策、速率限制及版权要求，避免对源平台造成过载或干扰。各平台对其内容、服务及政策拥有最终解释权
- **知识产权保护**：通过本存储库获取的内容受相应版权法保护。使用者应遵守相关许可协议、版权声明及使用条款，不得从事任何侵犯知识产权的行为
- **安全防护建议**：虽然本存储库采用无日志架构，不存储用户请求数据，但基于互联网传输的固有风险，建议使用者对下载内容进行安全扫描，尤其对可执行文件、脚本等保持谨慎
- **开源性质声明**：本存储库为开源项目，作者与贡献者不承担提供技术支持、错误修复或持续维护的义务。外部贡献的合并不代表对特定用途或效果的承诺与背书
- **名称使用规范**：严禁任何可能暗示作者或贡献者提供商业合作、技术支持、担保或背书的表述。涉及存储库名称或作者标识的使用应遵循相关法律法规及通用规范
- **免责声明更新**：本免责声明可能随存储库发展或法律环境变化进行更新修订。使用者继续使用、复制、分发或修改本存储库即视为接受最新版本的免责声明
