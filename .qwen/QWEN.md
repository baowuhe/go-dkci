# go-dkci项目

## 项目介绍
管理docker镜像的工具，包括：
- 导出docker镜像（tar文件）并保存到百度网盘
- 从百度网盘下载镜像（tar文件）并导入到docker

## 约束和规则
### 必须遵守的约束
- 从docker导出镜像时，命名为：`<image_name>_<tag>_<os>_<arch>.tar`，image_name如果包含`/`，则替换为`·`
- 缓存目录（临时目录）必须为/tmp/go-dkci，缓存目录必须为/tmp/go-dkci ！！！
- 打印成功的提示，必须以"[√] "作为前缀，打印错误或者失败的提示，必须以"[x] "作为前缀，程序退出码为1

### 必须遵循的三方库使用规则
- 命令行交互库（如实现多选列表），必须使用：github.com/AlecAivazis/survey/v2
- 百度云盘登录、上传、下载、文件列表查询、文件删除、创建目录等操作，必须使用：github.com/baowuhe/go-bdfs/pan包，版本v0.1.2，AIP文档：https://github.com/baowuhe/go-bdfs/blob/master/API.md


