package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"navdesk/handlers"
	"navdesk/middleware"
	"navdesk/models"
	"navdesk/storage"
)

func main() {
	// 获取端口号
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// 创建存储实例
	store := storage.NewStorage()

	// 初始化操作已移除，项目使用预打包的数据文件

	// 创建Gin路由器
	r := gin.Default()

	// CORS设置
	r.Use(cors.Default())

	// Session设置
	secretKey := os.Getenv("SESSION_SECRET")
	if secretKey == "" {
		// 从users.json文件中读取secretKey
		key, err := store.GetSecretKey()
		if err != nil {
			log.Fatalf("读取 users.json 中的 secretKey 失败: %v", err)
		}
		if key == "" {
			log.Fatal("安全错误: secretKey 未配置。\n" +
				"请在 data/users.json 中设置 secretKey 或使用 SESSION_SECRET 环境变量。\n" +
				"示例: \"secretKey\": \"your-secure-random-key-here\"")
		}
		secretKey = key
	}
	cookieStore := cookie.NewStore([]byte(secretKey))
	cookieStore.Options(sessions.Options{
		Path:     "/",
		Domain:   "",
		MaxAge:   30 * 24 * 60 * 60, // 30天持久化 (30天 * 24小时 * 60分钟 * 60秒)
		HttpOnly: true,
		Secure:   false, // 在生产环境中应设为true
		SameSite: http.SameSiteDefaultMode,
	})
	r.Use(sessions.Sessions("navdesk_session", cookieStore))

	// 静态文件服务
	r.Static("/static", "./public")
	r.StaticFS("/uploads", http.Dir("./data/uploads"))

	// Favicon服务
	r.GET("/favicon.ico", func(c *gin.Context) {
		faviconPath := filepath.Join("./data/uploads/favicon/favicon.ico")
		if _, err := os.Stat(faviconPath); err == nil {
			c.File(faviconPath)
		} else {
			c.Status(http.StatusNotFound)
		}
	})

	// 创建处理器
	authHandler := handlers.NewAuthHandler(store)
	categoriesHandler := handlers.NewCategoriesHandler(store)
	bookmarksHandler := handlers.NewBookmarksHandler(store)
	uploadHandler := handlers.NewUploadHandler(store)
	settingsHandler := handlers.NewSettingsHandler(store)

	// API路由组
	api := r.Group("/api")

	// 认证相关路由
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/status", authHandler.Status)
	}

	// 分类相关路由
	categories := api.Group("/categories")
	{
		categories.GET("/", categoriesHandler.GetCategories)
		categories.GET("/:id", categoriesHandler.GetCategory)
		categories.POST("/", middleware.RequireAuth(), categoriesHandler.CreateCategory)
		categories.PUT("/:id", middleware.RequireAuth(), categoriesHandler.UpdateCategory)
		categories.DELETE("/:id", middleware.RequireAuth(), categoriesHandler.DeleteCategory)
	}

	// 书签相关路由
	bookmarks := api.Group("/bookmarks")
	{
		bookmarks.GET("/", bookmarksHandler.GetBookmarks)
		bookmarks.GET("/category/:categoryId", bookmarksHandler.GetBookmarksByCategory)
		bookmarks.GET("/search/:keyword", bookmarksHandler.SearchBookmarksH)
		bookmarks.GET("/:id", bookmarksHandler.GetBookmark)
		bookmarks.POST("/", middleware.RequireAuth(), bookmarksHandler.CreateBookmark)
		bookmarks.PUT("/:id", middleware.RequireAuth(), bookmarksHandler.UpdateBookmark)
		bookmarks.DELETE("/:id", middleware.RequireAuth(), bookmarksHandler.DeleteBookmark)
	}

	// 上传相关路由
	upload := api.Group("/upload", middleware.RequireAuth())
	{
		upload.POST("/icon", uploadHandler.UploadIcon)
		upload.POST("/favicon", uploadHandler.UploadFavicon)
	}

	// 设置相关路由
	settings := api.Group("/settings")
	{
		settings.GET("/", settingsHandler.GetSettings)
		settings.POST("/", middleware.RequireAuth(), settingsHandler.UpdateSettings)
	}

	// 前端数据接口
	r.GET("/api/data", func(c *gin.Context) {
		categories, err := store.GetCategories()
		if err != nil {
			log.Printf("Error reading categories: %v", err)
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "读取数据失败",
			})
			return
		}

		bookmarks, err := store.GetBookmarks()
		if err != nil {
			log.Printf("Error reading bookmarks: %v", err)
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "读取数据失败",
			})
			return
		}

		settings, err := store.GetSettings()
		if err != nil {
			log.Printf("Error reading settings: %v", err)
			// 使用默认设置
			settings = models.Settings{
				SiteTitle:    "极简网站导航",
				CardWidth:    180,
				CardHeight:   80,
				IconWidth:    50,
				IconHeight:   50,
				SidebarWidth: 300,
				Theme:        "auto",
			}
		}

		// 按排序值排序
		sort.Slice(categories, func(i, j int) bool {
			return categories[i].Sort < categories[j].Sort
		})

		sort.Slice(bookmarks, func(i, j int) bool {
			if bookmarks[i].Sort == bookmarks[j].Sort {
				return bookmarks[i].CreatedAt.Before(bookmarks[j].CreatedAt)
			}
			return bookmarks[i].Sort < bookmarks[j].Sort
		})

		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Data: models.DataResponse{
				Categories: categories,
				Bookmarks:  bookmarks,
				Settings:   settings,
			},
		})
	})

	// 后台登录页面（不需要认证）
	r.GET("/admin/login.html", func(c *gin.Context) {
		c.File("./public/admin/login.html")
	})

	// 需要认证的后台页面
	r.GET("/admin/categories.html", func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")
		log.Printf("访问 /admin/categories.html - Session中的username: %v", username)
		if username == nil {
			log.Printf("Session中没有username，重定向到登录页面")
			c.Redirect(http.StatusFound, "/admin/login.html")
			return
		}
		log.Printf("Session验证通过，返回categories页面")
		c.File("./public/admin/categories.html")
	})

	r.GET("/admin/category-detail.html", func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")
		if username == nil {
			c.Redirect(http.StatusFound, "/admin/login.html")
			return
		}
		c.File("./public/admin/category-detail.html")
	})

	r.GET("/admin/settings.html", func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")
		if username == nil {
			c.Redirect(http.StatusFound, "/admin/login.html")
			return
		}
		c.File("./public/admin/settings.html")
	})

	// 默认路由
	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	r.GET("/admin", func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")
		if username != nil {
			c.Redirect(http.StatusFound, "/admin/categories.html")
		} else {
			c.Redirect(http.StatusFound, "/admin/login.html")
		}
	})

	r.GET("/admin/", func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")
		if username != nil {
			c.Redirect(http.StatusFound, "/admin/categories.html")
		} else {
			c.Redirect(http.StatusFound, "/admin/login.html")
		}
	})

	// 启动服务器
	log.Printf("服务器启动成功 - 端口: %s", port)
	log.Printf("前端页面: http://localhost:%s", port)
	log.Printf("后台管理: http://localhost:%s/admin", port)

	// 检查是否使用默认密码
	users, err := store.GetUsers()
	if err == nil {
		if adminUser, exists := users["admin"]; exists && adminUser.Password == "123456" {
			log.Printf("默认账号: admin / 123456，请在 data/users.json 中修改密码！")
		}
	}

	// 安全提示
	if secretKey == "your-secure-random-key-2025-navdesk-session" {
		log.Printf("安全警告: 正在使用默认secretKey，请在 data/users.json 中修改为随机安全的值！")
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
