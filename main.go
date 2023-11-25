package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

    "github.com/gin-contrib/sessions"
    "github.com/gin-contrib/sessions/cookie"

	"todolist.go/db"
	"todolist.go/service"
)

const port = 8000

func main() {
	// initialize DB connection
	dsn := db.DefaultDSN(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))
	if err := db.Connect(dsn); err != nil {
		log.Fatal(err)
	}

	// initialize Gin engine
	engine := gin.Default()
	engine.LoadHTMLGlob("views/*.html")

    // prepare session
    store := cookie.NewStore([]byte("my-secret"))
    engine.Use(sessions.Sessions("user-session", store))
	// routing
	engine.Static("/assets", "./assets")
	engine.GET("/", service.Home)
	engine.GET("/list", service.LoginCheck, service.TaskList)

	taskGroup := engine.Group("/task")
    taskGroup.Use(service.LoginCheck)
    {
        taskGroup.GET("/:id", service.ShowTask)
        taskGroup.GET("/new", service.NewTaskForm)
        taskGroup.POST("/new", service.RegisterTask)
        taskGroup.GET("/edit/:id", service.EditTaskForm)
        taskGroup.POST("/edit/:id", service.UpdateTask)
        taskGroup.GET("/delete/:id", service.DeleteTask)
		taskGroup.GET("/share_form", service.ShareForm)
		taskGroup.POST("/share_task",service.ShareTask)
		taskGroup.GET("/completed_rate",service.CompletedRate)
    }


	// ユーザ登録
    engine.GET("/user/new", service.NewUserForm)
    engine.POST("/user/new", service.RegisterUser)
    engine.GET("/login", service.LoginForm)
    engine.POST("/login", service.Login)
	engine.GET("/change_name_form", service.LoginCheck,service.ChangeNameForm)
	engine.POST("/change_name", service.LoginCheck,service.ChangeName)
	engine.GET("/change_password_form", service.LoginCheck,service.ChangePasswordForm)
	engine.POST("/change_password", service.LoginCheck,service.ChangePassword)
	engine.GET("/logout", service.LoginCheck, service.Logout)
	engine.GET("/delete_account_form",service.LoginCheck,service.DeleteForm)
	engine.POST("/delete_account", service.LoginCheck, service.DeleteAccount,service.Logout)
	// start server
	engine.Run(fmt.Sprintf(":%d", port))
}
