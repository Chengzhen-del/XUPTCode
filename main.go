package main

import (
	"CMS/internal/handler"
	"CMS/internal/pkg" // 统一导入pkg包
	"CMS/internal/repository"
	"CMS/internal/router"
	"CMS/internal/service"
	"regexp"

	// 必须引入生成的docs包（swag init后自动创建，替换为你的项目实际模块路径）

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-sql-driver/mysql"
)

// 自定义手机号验证函数（保持不变）
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" { // 空值不校验（omitempty）
		return true
	}
	// 国内手机号正则
	reg := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return reg.MatchString(phone)
}

// @title CMS系统API文档
// @version 1.0
// @description CMS系统核心接口文档，包含用户注册、登录、信息更新、头像修改、邮箱验证等功能，基于Go+Gin+原生MySQL实现
// @termsOfService http://example.com/terms/
// @contact.name CMS技术支持
// @contact.url http://example.com/support
// @contact.email support@cms-example.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// ========== 保留：注册自定义验证器 ==========
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册自定义标签：phone（替代regex）
		_ = v.RegisterValidation("phone", validatePhone)
	}

	// ========== 核心修改：初始化原生MySQL连接（替换GORM） ==========
	dsn := "root:123456@tcp(127.0.0.1:3306)/go_project?charset=utf8mb4&parseTime=True&loc=Local"
	// 注意：gormDB 改为 db（原生*sql.DB）
	db, err := pkg.InitMySQL(dsn)
	if err != nil {
		panic("数据库连接失败：" + err.Error())
	}
	defer db.Close() // 程序退出时关闭连接（关键，防止连接泄露）

	// ========== 以下逻辑保持不变（接口兼容） ==========
	// 初始化仓储层（传入原生*sql.DB）
	userRepo := repository.NewUserRepo(db)
	useraccRepo := repository.NewAccountRepo(db)
	resourceRepo := repository.NewResourceRepo(db)

	// 初始化业务层
	staffSvc := service.NewStaffService(userRepo, useraccRepo)
	accSvc := service.NewAccountService(useraccRepo, userRepo)
	resourceSvc := service.NewResourceService(resourceRepo, userRepo, useraccRepo)
	// 初始化处理器
	staffHandler := handler.NewStaffHandler(staffSvc, accSvc, resourceSvc)

	// 初始化路由
	r := router.SetupRouter(staffHandler)

	// 启动服务
	if err := r.Run(":8080"); err != nil {
		panic("服务启动失败：" + err.Error())
	}
}
