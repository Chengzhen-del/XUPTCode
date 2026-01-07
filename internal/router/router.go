package router

import (
	"CMS/internal/handler"
	"CMS/internal/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化路由
func SetupRouter(staffHandler *handler.StaffHandler) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Cors())

	// 静态资源（存放前端CSS/JS/图片）
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "favicon.ico")
	r.LoadHTMLGlob("templates/*") // 加载前端页面模板
	// 根路由
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "公式面试网站接口服务正常",
		})
	})
	page := r.Group("/page")
	{
		page.GET("/login", func(c *gin.Context) {
			c.HTML(http.StatusOK, "login.html", nil)
		})
		page.GET("/register", func(c *gin.Context) {
			c.HTML(http.StatusOK, "register.html", nil)
		})
		page.GET("/person", func(c *gin.Context) {
			c.HTML(http.StatusOK, "person.html", nil)
		})
		page.GET("/index", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", nil)
		})
		page.GET("/word", func(c *gin.Context) {
			c.HTML(http.StatusOK, "word.html", nil)
		})
		page.GET("/first", func(c *gin.Context) {
			c.HTML(http.StatusOK, "first.html", nil)
		})
		page.GET("/resource-list", func(c *gin.Context) {
			c.HTML(http.StatusOK, "resource-list.html", nil)
		})
		page.GET("/upload", func(c *gin.Context) {
			c.HTML(http.StatusOK, "upload.html", nil)
		})
		page.GET("/ai", func(c *gin.Context) {
			c.HTML(http.StatusOK, "AI.html", nil)
		})
		page.GET("/resource-detail", func(c *gin.Context) {
			c.HTML(http.StatusOK, "resource-detail.html", nil)
		})
	}

	// 员工接口分组
	staffGroup := r.Group("/staff")
	{
		staffGroup.POST("/register", staffHandler.Register)
		staffGroup.POST("/login", staffHandler.Login)
		staffGroup.POST("/elogin", staffHandler.Elogin)
		staffGroup.POST("/elogin/res", staffHandler.Eres)
		staffGroup.POST("/logout", staffHandler.Logout)
		staffGroup.POST("/update", middleware.JWTMiddleware(), staffHandler.UpdateUserHandler)
		staffGroup.POST("/update-avatar", middleware.JWTMiddleware(), staffHandler.UpdateAvatarHandler) // 更新头像
		staffGroup.GET("/get-info", middleware.JWTMiddleware(), staffHandler.GetUserByUuid)

	}
	accountGroup := r.Group("/account")
	{
		accountGroup.GET("/get-account", middleware.JWTMiddleware(), staffHandler.GetAccountByUserUUID)
		accountGroup.POST("/recharge", middleware.JWTMiddleware(), staffHandler.Recharge)
		accountGroup.POST("/deduct", middleware.JWTMiddleware(), staffHandler.Deduct)
	}
	resourceGroup := r.Group("/resource")
	{
		resourceGroup.POST("/create", middleware.JWTMiddleware(), staffHandler.CreateResourceHandler)
		resourceGroup.POST("/list", middleware.JWTMiddleware(), staffHandler.ResourceListHandler)
		resourceGroup.GET("/detail", staffHandler.ResourceDetailHandler)
		resourceGroup.POST("/incr-view-count", staffHandler.IncrViewCountHandler)
		resourceGroup.POST("/like", staffHandler.IncrLikeCountHandler)
		resourceGroup.POST("/comment", staffHandler.CreateCommentHandler)
	}
	r.GET("/api/auth/verify-token", middleware.JWTMiddleware(), staffHandler.Checktoken) //检验token有效性
	r.GET("get-letter", middleware.JWTMiddleware(), middleware.JWTMiddleware(), staffHandler.GetWordText)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "success",
			"data":    "This is a test",
		})
	})
	return r
}
