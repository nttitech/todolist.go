package service
 
import (
    "crypto/sha256"
    "encoding/hex"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/sessions"
    database "todolist.go/db"
)
 
func NewUserForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "new_user_form.html", gin.H{"Title": "Register user"})
}

func hash(pw string) []byte {
    const salt = "todolist.go#"
    h := sha256.New()
    h.Write([]byte(salt))
    h.Write([]byte(pw))
    return h.Sum(nil)
}

func RegisterUser(ctx *gin.Context) {
    // フォームデータの受け取り
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
    switch {
    case username == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Usernane is not provided", "Username": username})
    case password == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Password is not provided", "Password": password})
    }
    
    // DB 接続
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // 重複チェック
    var duplicate int
    err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name=?", username)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    if duplicate > 0 {
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Username is already taken", "Username": username, "Password": password})
        return
    }
 
    // DB への保存
    result, err := db.Exec("INSERT INTO users(name, password) VALUES (?, ?)", username, hash(password))
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // 保存状態の確認
    id, _ := result.LastInsertId()
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE id = ?", id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    ctx.Redirect(http.StatusFound, "/login")
}

func LoginForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "login.html", gin.H{"Title": "Login user"})
}
const userkey = "user"
 
func Login(ctx *gin.Context) {
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
 
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE name = ?", username)
    if err != nil {
        ctx.HTML(http.StatusBadRequest, "login.html", gin.H{"Title": "Login", "Username": username, "Error": "No such user"})
        return
    }
 
    // パスワードの照合
    if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(password)) {
        ctx.HTML(http.StatusBadRequest, "login.html", gin.H{"Title": "Login", "Username": username, "Error": "Incorrect password"})
        return
    }
 
    // セッションの保存
    session := sessions.Default(ctx)
    session.Set(userkey, user.ID)
    session.Save()
 
    ctx.Redirect(http.StatusFound, "/list")
}

func LoginCheck(ctx *gin.Context) {
    if sessions.Default(ctx).Get(userkey) == nil {
        ctx.Redirect(http.StatusFound, "/login")
        ctx.Abort()
    } else {
        ctx.Next()
    }
}

func Logout(ctx *gin.Context) {
    session := sessions.Default(ctx)
    session.Clear()
    session.Options(sessions.Options{MaxAge: -1})
    session.Save()
    ctx.Redirect(http.StatusFound, "/")
}

func ChangeNameForm(ctx *gin.Context){
	ctx.HTML(http.StatusOK, "change_name_form.html", gin.H{"Title": "Change username"})
}

func ChangeName(ctx *gin.Context){
	new_username, exist := ctx.GetPostForm("new_username")
	if !exist {
		Error(http.StatusBadRequest, "No username is given")(ctx)
		return
	}
	
	userID := sessions.Default(ctx).Get("user")

	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	_, err = db.Exec("UPDATE users SET name  = ? WHERE id = ?", new_username, userID)	   
	if err != nil {
		   Error(http.StatusInternalServerError, err.Error())(ctx)
		   return
	   }

   ctx.Redirect(http.StatusFound, "/list")
}

func ChangePasswordForm(ctx *gin.Context){
	ctx.HTML(http.StatusOK, "change_password_form.html", gin.H{"Title": "Change password"})
}

func ChangePassword(ctx *gin.Context){
	new_password, exist := ctx.GetPostForm("new_password")
	if !exist {
		Error(http.StatusBadRequest, "New password is not given")(ctx)
		return
	}
	current_password, exist := ctx.GetPostForm("current_password")
	if !exist {
		Error(http.StatusBadRequest, "Current password is not given")(ctx)
		return
	}

	userID := sessions.Default(ctx).Get("user")
	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT password FROM users WHERE id = ?", userID)
    if err != nil {
        Error(http.StatusBadRequest,  "No such user")(ctx)
        return
    }
 
    // パスワードの照合
    if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(current_password)) {
        Error(http.StatusBadRequest,  "Incorrect password")(ctx)
        return
    }
	// Create new data with given title on DB
	_, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", hash(new_password), userID)	   
	if err != nil {
		   Error(http.StatusInternalServerError, err.Error())(ctx)
		   return
	   }
   ctx.Redirect(http.StatusFound, "list")
}

func DeleteForm(ctx *gin.Context){
	ctx.HTML(http.StatusOK, "delete_account_form.html", gin.H{"Title": "Delete account"})
}

func DeleteAccount(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusBadRequest,  "Incorrect password")(ctx)
		return
	}

	// トランザクション開始
	tx := db.MustBegin()

	// task テーブルから関連するレコードを削除
	_, err = tx.Exec("DELETE FROM tasks WHERE id IN (SELECT task_id FROM ownership WHERE user_id = ?)", userID) 
	if err != nil {
		tx.Rollback()
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// ownership テーブルから関連するレコードを削除
	_, err = tx.Exec("DELETE FROM ownership WHERE user_id = ?", userID)
	if err != nil {
		tx.Rollback()
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// user テーブルからユーザーを削除
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		tx.Rollback()
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	tx.Commit()

	ctx.Redirect(http.StatusFound, "/")
}
