# local-buckets

本地資料倉庫 (管理資料, 備份資料)

- 完全彻底只考虑自用
- 一个專案可以包含一个或多个仓库
- 仓库内不可包含子仓库 (子資料夹)
- 每个仓库可单独设置檔案數量上限, 是否加密等参数
- 把文档上传到加密仓库, 会自动加密文档
- 有导出导入功能
  - 导出(export): 下载文档时, 附带同名 toml 文档, 内含备注/关键词等属性
  - 导入(import): 上传文档时, 如果带有同名 toml 文档, 则自动把相关属性写入数据库

## 初始化一个專案 (initialize a project)

1. 新建一个空資料夹
2. 复制 local-buckets.exe 和 public 資料夹, 黏贴到该資料夹中

此时, 该資料夹就是一个新的專案, 启动该資料夹内的 local-buckets.exe, 即可使用这个新專案.

每个專案都有自己的 local-buckets.exe 和 public 資料夹, 这种方式的优点是可以保持每个專案的独立性, 避免專案互相干扰, 缺点是更新升级也需要各个專案分别操作. 由于预估更新升级很少, 專案数量也很少, 因此这个缺点可以容忍.

## 新建仓库

- 通过网页表单新建仓库
- 新建仓库后, 提示用户仓库資料夹的地址, 可点击复制, 同时显示默认设置 (檔案体积上限, 是否加密等)
- 仓库資料夹名只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)
- 仓库可以有标题, 可使用任何语言任何字符

## 上传檔案

- 把想要添加到数据库中的新檔案放进 waiting 資料夹
- 在网页界面刷新 waiting.html 页面, 根据页面提示执行添加檔案的操作
- 在数据库中, 檔案名不分大小写
- 在一个專案内, 檔案名必须是唯一的 (即, 不允许有两个同名檔案, 不同仓库内的檔案也不允许同名)
- 在一个專案内, 檔案内容必须是唯一的 (即, 不允许有重复檔案)
- 把文档上传到加密仓库, 会自动加密文档

### 关于檔案名的唯一性

- 本来我打算设计为一个仓库内的檔案不可同名
  - 举个例子, bucketABC 内不允许有两个 file123.txt,
    但允许 bucketABC 内有一个 file123.txt, 同时 bucketCDE 内也有一个 file123.txt
- 但是为了方便修改檔案, 如果一个檔案名对应多个仓库, 处理起来就比较复杂,
  因此为了偷懒, 就设计为整个專案内都不允许同名, 也就是说,
  - 如果 bucketABC 内有一个 file123.txt, 那么在 bucketCDE 内也不允许有 file123.txt

不允许跨仓库檔案重名的好处:

- **好处1**: 上传檔案时发现同名檔案, 可以提示用户改檔案名或覆盖檔案,
  此时如果选择覆盖檔案, 就能直接覆盖, 不需要选择仓库.
- **好处2**: 跨仓库移动檔案时, 不用处理檔案重名的问题.

### waiting

- 网页 waiting 页面只列出等待上传的檔案信息, 不提供修改信息的功能
- 网页 waiting 页面可以选择上传到哪个仓库 (Bucket),
  但全部等待檔案只能统一选择一个仓库,
  如果想上传到不同仓库, 需要手动分批上传 (分批放进 waiting 資料夹).
- 用户如果想修改檔案信息 (备注, 关键词等), 只能在上传后再修改

在技术上, 上传檔案时, 前端不会把檔案列表传给后端, 只会把 Bucket ID 传给后端,
实际上上传哪些檔案, 完全取决于 waiting 資料夹里有什么檔案.

### 更新同名檔案

- 发现 waiting 資料夹内有檔案与数据库中的现有檔案同名时, 提示用户处理.
- 用户可选择覆盖或更改檔案名.
- 更新同名檔案时, 不批量处理, 而是逐一处理.

## 下载檔案

- 请勿直接修改檔案内容
- 如果要修改檔案内容, 必须先下载, 修改后再上传
- 下载檔案默认下载到 waiting 資料夹

## 导出/导入

- project.toml 中的 DownloadExport 默認為 false, 正常下載檔案,
  - 如果設定為 true 則下載時會導出與所下載檔案同名的 toml.
  - 例如, 如果下载 "abc.txt", 就会同时得到 "abc.txt.toml"
- 另外, 不管 DownloadExport 是真是假, 只要上传时在 waiting 资料夹内的文档  
  有同名的 toml, 则会执行导入 (import)
  - 例如, 同时有 "abc.txt" 和 "abc.txt.toml", 就會執行導入.
- 所谓导入 (import), 就是在上传时把 toml 文档中的 Notes, Keywords 等信息写入数据库
- 如果希望当作一个新文档来上传, 请删除同名的 toml 文档

### 特殊情況

- 导入时, 会根据同名 toml 文档中的 BucketName 上传到指定的仓库中,  
  但如果找不到 BucketName 的仓库, 则上传到网页表单指定的仓库中.
- 如果導入的檔案與數據庫中的檔案同名, 並且用戶選擇了 overwrite,  
  那麼就忽略 toml 中的 BucketName, 以數據庫中的 bucket 為準.  
  (因爲 overwrite 不能移動檔案)

## 只读保护

檔案(File) 自动设为只读权限,
建议用户不要直接修改檔案, 而是通过网页界面进行操作.

本来我打算设计为让用户能自由修改檔案,
但这样一来就不得不经常扫描全部檔案, 计算每一个檔案的 checksum,
以便发现哪些檔案被修改过, 这会消耗大量计算资源, 会消耗硬盘.

因此, 现在设计为禁止用户直接修改.

## 刷新数据库 (取消该设计)

网页界面有一个很重要的按钮 **Update Database**.

该按钮的主要功能是检查真实檔案与数据库中的信息是否一致.

例如, 如果添加了新檔案, 然后点击 **Update Database**,  
就能发现真实檔案与数据库中的信息不一致, 因为此时数据库中还没有新檔案的信息.

另外, 有时我们可能更改檔案名, 修改檔案内容, 删除檔案等等,  
如果不是通过网页界面操作, 而是直接修改真实檔案, 就会造成檔案与数据库信息不一致.  
此时就需要点击 **Update Database** 按钮刷新数据库.

发现信息不一致后, 提示用户进行下一步处理.

## 删除檔案

- 通过网页按钮删除檔案 (请勿通过其他途径删除檔案)
- 第一次删除只是把檔案标记为 "deleted"
- 第二次删除才是真正删除
- 也可以进入資料夹内手动删除檔案, 详见下面的 "批量删除" 章节

## 更改檔案名

- 在数据库中, 檔案名不分大小写
- 通过网页表单更改檔案名  (请勿通过其他途径更改檔案名)

### 跨仓库移动檔案

- 在同一專案内, 可跨仓库移动檔案.
- 通过网页表单移动檔案  (请勿通过其他途径移动檔案)

## 备份专案

- 因为双向同步备份很容易出错, 程序复杂, 使用时也要非常小心.  
- 因此, local-buckets 采用 **单向同步备份** 方式
- 永远不用担心发生旧文档覆盖新文档的情况,
- 也不需要处理冲突
  - 备份专案专门用于备份, 不可进行添加/修改/删除等操作, 只能查阅.
  - 原专案里的文档必然是最新的, 备份专案里的文档必然是旧的, 直接用新的覆盖旧的即可.

### 新建一个备份专案

可通过网页表单新建备份专案.

建议把备份专案建在 USB 硬盘中 (鸡蛋不要全部放在同一个篮子里).

1. 必须先有一个普通专案 (以下称为 "原专案")
2. 新建一个空资料夹 (以下称为 "备份专案根目录")
3. 把原专案的 project.toml, local-buckets.exe 和 public 资料夹复制粘贴到备份专案根目录内
4. 把备份专案的 project.toml 中的 IsBackup 设为 True

### 删除备份专案, 恢复备份专案

- 可通过网页按钮删除备份专案, 这只是从列表中删除, 不会删除硬盘中的文件.
- 由于添加备份专案必须指定一个空文件夹, 因此一旦删除, 就无法通过网页表单把备份专案加回去
- 但可以直接编辑 project.toml 文件, 例如在文件中修改 BackupProjects 的内容: `BackupProjects = ['D:\temp\temp-bk-project']`

### 对比, 同步

- 选择框, 动作, 方向, 檔案名
- 新增檔案 (左→右)
- 新增檔案 (右→左)
- 删除檔案 (左)
- 删除檔案 (右)
- 更新檔案 (根据日期初步确定方向, 可点击改变方向)

### 对比同步方案

- 选择右边 (对比目标)
- 选择对比哪些仓库

## 检查文档完整性

- 可自动检查文档完整性
- 可手动检查文档完整性
- 可设置检查间隔 (比如每 30 天检查一次)

### 检查间隔

假设设定为每 30 天检查一次 (默认设定), 那么, 每次自动检查或手动点击 Check 按钮时,
本软件会先找出超过 30 天未检查的文档, 逐一检查其完整性. 如果全部文档都已经在
30 天内检查过, 就不再重复检查.

### 触发自动检查

- 源专案: 每次上传文档时触发自动检查
- 备份专案: 每次执行同步时触发自动检查

### 自动检查时, 文档体积的累计上限

为了避免一次检查时间太长, 自动检查时, 每次只检查很少文件,
因此日常使用过程中, 用户很可能完全没有感觉, 自动检查就完成了.

- 每次检查文档的体积累计上限是 1 GB （`1 << 30`)
- 但, 每次至少先检查 1 个文件 (避免被超过 1GB 的文件卡住)

另外, 手动检查 (点击 Check 按钮) 的单次上限是 4GB.

## 缩略图

图片的缩略图以 base64 形式保存.

## 加密

- <https://cryptography.io/en/latest/hazmat/primitives/aead/>
- 每个仓库可单独选择是否加密
- 把文档上传到加密仓库, 会自动加密文件
- 每个專案统一一个密码, 换言之, 一个專案内的各个加密仓库的密码相同
- 未输入密码不会显示加密仓库
- 初始密码是 "abc123", 请自行更改密码.

### 备份密钥

- 必须备份 project.toml 檔案, 否则即使输入正确密码也无法解密.  
- 更具体而言, 必须拥有正确密码以及 project.toml 里的 CipherKey 才能解密.
- 并且要注意, 更改密码会改变 CipherKey, 因此每次更改密码后都要重新备份 CipherKey.
- 建议使用密码管理器记住密码和 CipherKey.

### 加密强度

密码越短越容易被破解, 本软件的加密方式, 要求密码长度超过 15 位才比较安全.

但是, 本软件的加密目的只是为了稍稍提高保密性而已, 不建议用来保存特别重要的机密.

因此, 反而建议密码短一点, 比如 8 位大小写英文字母与数字的组合, 万一忘记密码, 自己破解起来也比较容易, 没必要追求太高的加密强度.

总之, 根据自己的保密性需求来决定密码长度吧.

### 管理员登入

启动本程式后, 打开首页, 点击 "管理登入 Admin Login" 按钮.
输入正确密码后即可登入.

- 如果未登入 (处于登出状态), 则看不见加密仓库; 登入后才可看见加密仓库及其中的文档.
- 登入后, 不会自动登出, 需要手动点击 Logout 按钮或关闭程序才会登出.
- 本程式不使用 session(cookie), 因此一旦登入, 以任何途径访问 api 均可获得解密文件.
- 简而言之, 本程式的加密, 只是简单保护, 安全等级不高, 请勿用来保存重要机密.

### 加密文档的 checksum

- 加密文档的 checksum, 取加密后的 checksum
- 因此, 上传文档时, 无法发现加密文档的内容是否重复
- 也就是说, 如果待上传文档与 **公开仓库** 里某个文档的内容相同, 无法上传,
  - 但如果待上传文档与 **加密仓库** 里某个文档的内容相同, 可以上传

## 已知小问题

- 上传文档到加密仓库时, 数据库中记录的文档体积是加密前的, 因此会有一点误差.

## TODO

- export/import
- 缩略图
- 搜索
- 自动检查文档
- 删除 File.BucketID
- 提醒仓库文档数量太多
  - "该仓库内已有 n 个档案, 档案数量太多可能影响性能, 建议创建新仓库,使用新仓库."
- 可设定是否模拟网络延迟
- 加密时需要把整个文件读进内存, 因此上传到加密仓库的文件不能太大.

A FAT32 directory can have 65,536 directory entries. Each file and subdirectory takes from 2 to 13 entries, depending on the length of its name.

What is the maximum number of files in a Windows folder?
Maximum number of files in a single folder: 4,294,967,295
