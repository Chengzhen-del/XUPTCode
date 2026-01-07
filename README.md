西邮代码库（XUPTCode）
基于 Go/Gin + MySQL + Tailwind CSS 的前后端分离代码分享与管理平台
项目介绍
XUPTCode 是面向开发者的代码分享平台，支持用户注册登录、代码资源发布、点赞 / 浏览 / 评论互动、账户充值消费等核心功能。采用前后端分离架构，后端基于 Go 语言 Gin 框架构建高性能 API，前端使用 Tailwind CSS 实现响应式布局，内置 Swagger 接口文档便于调试。
技术栈
分类	技术选型
后端框架	Go 1.21+、Gin Web Framework
ORM 工具	GORM（数据库交互）
数据库	MySQL 8.0+（关系型数据库存储）
前端框架	HTML5、Tailwind CSS、Font Awesome
接口文档	Swagger/OpenAPI（自动生成 RESTful API 文档）
依赖管理	Go Modules
部署工具	Git、Docker（可选）
快速启动
前置条件
安装 Go 1.21+：官方下载地址
安装 MySQL 8.0+：官方下载地址
克隆仓库到本地
bash
运行
git clone https://github.com/Chengzhen-del/XUPTCode.git
cd XUPTCode
后端启动步骤
安装依赖
bash
运行
go mod tidy
配置数据库
复制 config/example.yaml 为 config/app.yaml
修改配置文件中的数据库连接信息：
yaml
database:
  driver: mysql
  dsn: root:你的密码@tcp(127.0.0.1:3306)/xupt_code?charset=utf8mb4&parseTime=True&loc=Local
  max_open_conns: 100
  max_idle_conns: 20
server:
  port: 8080
初始化数据库
执行项目根目录下的 SQL 脚本 sql/init.sql，创建所需表结构
或调用 model 包的自动迁移功能：
go
运行
// 在 cmd/main.go 中添加
db.AutoMigrate(&model.User{}, &model.Resource{}, &model.UserAccount{})
启动后端服务
bash
运行
go run cmd/main.go
服务启动后访问：http://localhost:8080
接口文档访问：http://localhost:8080/swagger/index.html
前端启动步骤
进入前端目录
bash
运行
cd web
安装依赖（需提前安装 Node.js）
bash
运行
npm install
启动开发服务
bash
运行
npm run dev
前端页面访问：http://localhost:3000
项目结构
plaintext
XUPTCode/
├── cmd/                  # 程序入口
│   └── main.go           # 启动文件，初始化路由、数据库
├── internal/             # 内部业务代码（不对外暴露）
│   ├── handler/          # 接口处理器，处理 HTTP 请求
│   ├── dto/              # 数据传输对象，定义请求/响应格式
│   ├── model/            # 数据库模型，映射 MySQL 表结构
│   ├── service/          # 业务逻辑层，处理核心业务
│   └── middleware/       # 中间件，如 JWT 认证、跨域处理
├── config/               # 配置文件目录
│   ├── example.yaml      # 配置示例
│   └── app.yaml          # 生产环境配置（需创建，不上传 Git）
├── web/                  # 前端代码目录
│   ├── src/              # 前端源码
│   └── tailwind.config.js # Tailwind CSS 配置
├── sql/                  # 数据库初始化脚本
├── .gitignore            # Git 忽略文件配置
├── go.mod                # Go 模块依赖
└── README.md             # 项目说明文档
核心功能模块
1. 用户管理模块
注册 / 登录：支持用户名 / 手机号 / 邮箱 + 密码登录，返回 JWT Token
信息更新：修改用户昵称、手机号、头像等信息
邮箱验证：支持邮箱验证码登录
Token 校验：验证登录凭证有效性
2. 资源管理模块
资源发布：发布文本 / 代码资源，自动关联发布者信息
资源查询：分页查询资源列表，支持关键词搜索
互动功能：资源点赞、浏览量统计、评论提交
3. 账户管理模块
充值消费：用户账户充值、余额扣减
账单查询：查看累计充值 / 消费金额
接口文档
启动后端服务后，访问 http://localhost:8080/swagger/index.html 查看完整接口文档，包含：
所有 API 的请求参数、响应格式
状态码说明（200 成功、400 参数错误、401 未登录、500 服务器错误）
支持在线调试接口（直接在网页端发送请求）
部署说明
开发环境
后端：本地启动 Go 服务，连接本地 MySQL
前端：本地启动 Vite 开发服务
生产环境（Docker 部署，可选）
编写 Dockerfile 和 docker-compose.yml
构建镜像并启动容器
bash
运行
docker-compose up -d
注意事项
敏感信息保护
不要将 config/app.yaml、.env 等配置文件提交到 Git
数据库密码、JWT 密钥等敏感信息通过环境变量注入
Swagger 文档更新
修改 handler 或 dto 后，执行 swag init 更新接口文档
确保所有结构体和接口都添加了规范的 Swagger 注释
常见问题
推送代码到 GitHub 失败：先执行 git add . && git commit -m "xxx" 再推送
数据库连接失败：检查 dsn 格式和 MySQL 服务是否启动
贡献指南
Fork 本仓库到你的 GitHub 账号
创建功能分支：git checkout -b feature/xxx
提交代码：git commit -m "feat: 新增 xxx 功能"
推送分支：git push origin feature/xxx
发起 Pull Request
许可证
本项目基于 MIT 许可证开源 - 详见 LICENSE 文件
