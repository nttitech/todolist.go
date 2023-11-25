package service

import (
	"net/http"
	"strconv"
	"fmt"
	"github.com/gin-gonic/gin"
	database "todolist.go/db"
	"github.com/gin-contrib/sessions"
	"time"
)

// TaskList renders list of tasks in DB
// TaskList renders list of tasks in DB
func TaskList(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // Get query parameter
    kw := ctx.Query("kw")
	is_done_value:=ctx.Query("is_done")


	pageStr := ctx.Query("page")
	if pageStr == ""{
		pageStr = "1"
	}
    pageSize := 10 // 1ページあたりのタスク数

    page, err := strconv.Atoi(pageStr)
    if err != nil || page < 1 {
        Error(http.StatusBadRequest, "Invalid page number")(ctx)
        return
    }
 
    // オフセットの計算
    offset := (page - 1) * pageSize


    // Get tasks in DB
    var tasks []database.Task
	var total int
    query := "SELECT id, title, created_at, is_done FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"
    switch {
	case kw != "":
		is_done, err := strconv.ParseBool(is_done_value)
		if err != nil {
			Error(http.StatusBadRequest, err.Error())(ctx)
			return
		}
		err = db.Select(&tasks, query + " AND title LIKE ? AND is_done = ? LIMIT ? OFFSET ?",userID, "%" + kw + "%",is_done,pageSize,offset)
		err = db.QueryRow("SELECT COUNT(*) FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ? AND title LIKE ? AND is_done = ?",userID, "%" + kw + "%",is_done).Scan(&total)
	case is_done_value != "":
		is_done, err := strconv.ParseBool(is_done_value)
		if err != nil {
			Error(http.StatusBadRequest, err.Error())(ctx)
			return
		}
		err = db.Select(&tasks, query + " AND is_done = ? LIMIT ? OFFSET ?",userID,is_done,pageSize,offset)
		err = db.QueryRow("SELECT COUNT(*) FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ? AND is_done = ?",userID,is_done).Scan(&total)
	default:
        err = db.Select(&tasks, query + " LIMIT ? OFFSET ?",userID,pageSize,offset)
		err = db.QueryRow("SELECT COUNT(*) FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?",userID).Scan(&total)
    }
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

	TotalPage := (total / pageSize) + 1
	var NextPage int

	if page != TotalPage{
		NextPage = page + 1
	}else{
		NextPage = 0
	}
    // Render tasks
    ctx.HTML(http.StatusOK, "task_list.html", gin.H{"Title": "Task list", "Tasks": tasks, "Kw": kw, "IsDone" : is_done_value,"NextPage":NextPage,"PreviousPage":page-1})
}

// ShowTask renders a task with given ID
func ShowTask(ctx *gin.Context) {
	userID := sessions.Default(ctx).Get("user")
	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// parse ID given as a parameter
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}
    
	var ownership database.Ownership
	err = db.Get(&ownership,"SELECT user_id FROM ownership WHERE task_id = ?",id)
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	if userID != ownership.User_id{
		Error(http.StatusBadRequest, "This is not your task")(ctx)
		return
	}

	// Get a task with given ID
	var task database.Task
	err = db.Get(&task, "SELECT * FROM tasks WHERE id=?", id) // Use DB#Get for one entry
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// Render task
	ctx.HTML(http.StatusOK, "task.html", task)
}

func NewTaskForm(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "form_new_task.html", gin.H{"Title": "Task registration"})
}

func RegisterTask(ctx *gin.Context) {
	    userID := sessions.Default(ctx).Get("user")
		// Get task title
		title, exist := ctx.GetPostForm("title")
		if !exist {
			Error(http.StatusBadRequest, "No title is given")(ctx)
			return
		}
		// Get DB connection
		db, err := database.GetConnection()
		if err != nil {
			Error(http.StatusInternalServerError, err.Error())(ctx)
			return
		}
		tx := db.MustBegin()
		// Create new data with given title on DB
	   result, err := db.Exec("INSERT INTO tasks (title) VALUES (?)", title)
	   if err != nil {
		   tx.Rollback()
		   Error(http.StatusInternalServerError, err.Error())(ctx)
		   return
	   }
	   taskID, err := result.LastInsertId()
	   if err != nil {
		   tx.Rollback()
		   Error(http.StatusInternalServerError, err.Error())(ctx)
		   return
	   }
	   _, err = tx.Exec("INSERT INTO ownership (user_id, task_id) VALUES (?, ?)", userID, taskID)
	   if err != nil {
		   tx.Rollback()
		   Error(http.StatusInternalServerError, err.Error())(ctx)
		   return
	   }
	   tx.Commit()
	   // Render status
	   path := "/list"  // デフォルトではタスク一覧ページへ戻る
	   if id, err := result.LastInsertId(); err == nil {
		   path = fmt.Sprintf("/task/%d", id)   // 正常にIDを取得できた場合は /task/<id> へ戻る
	   }
	   ctx.Redirect(http.StatusFound, path)
}

func EditTaskForm(ctx *gin.Context) {
    // ID の取得
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    // Get target task
    var task database.Task
    err = db.Get(&task, "SELECT * FROM tasks WHERE id=?", id)
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }
    // Render edit form
    ctx.HTML(http.StatusOK, "form_edit_task.html",
        gin.H{"Title": fmt.Sprintf("Edit task %d", task.ID), "Task": task})
}

func UpdateTask(ctx *gin.Context){
		// Get task title
		title, exist := ctx.GetPostForm("title")
		if !exist {
			Error(http.StatusBadRequest, "No title is given")(ctx)
			return
		}
		// Get task is_done
		is_done_value, exist := ctx.GetPostForm("is_done")
		if !exist {
			Error(http.StatusBadRequest, "No is_done is given")(ctx)
			return
		}
		is_done, err := strconv.ParseBool(is_done_value)
		if err != nil {
			Error(http.StatusBadRequest, err.Error())(ctx)
			return
		}
		// Get ID
		id, err := strconv.Atoi(ctx.Param("id"))
		if err != nil {
			Error(http.StatusBadRequest, err.Error())(ctx)
			return
		}
		// Get DB connection
		db, err := database.GetConnection()
		if err != nil {
			Error(http.StatusInternalServerError, err.Error())(ctx)
			return
		}
		// Create new data with given title on DB
		_, err = db.Exec("UPDATE tasks SET title = ?, is_done = ? WHERE id = ?", title, is_done, id)
	   if err != nil {
		   Error(http.StatusInternalServerError, err.Error())(ctx)
		   return
	   }
	   // Render status
	   path := fmt.Sprintf("/task/%d", id)   // 正常にIDを取得できた場合は /task/<id> へ戻る
	   ctx.Redirect(http.StatusFound, path)
}

func DeleteTask(ctx *gin.Context) {
    // ID の取得
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    // Delete the task from DB
    _, err = db.Exec("DELETE FROM tasks WHERE id=?", id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    // Redirect to /list
    ctx.Redirect(http.StatusFound, "/list")
}

func ShareForm(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "share_form.html", gin.H{"Title": "Share Task"})
}

func ShareTask(ctx *gin.Context) {
	// Get task title
	number, exist := ctx.GetPostForm("number")
	if !exist {
		Error(http.StatusBadRequest, "No number is given")(ctx)
		return
	}
	Username, exist := ctx.GetPostForm("username")
	if !exist {
		Error(http.StatusBadRequest, "No User Name is given")(ctx)
		return
	}
	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}
	var user database.User
    err = db.Get(&user, "SELECT id FROM users WHERE name=?", Username)
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

   _, err = db.Exec("INSERT INTO ownership (user_id, task_id) VALUES (?,?)", user.ID,number)
   if err != nil {
	   Error(http.StatusInternalServerError, err.Error())(ctx)
	   return
   }

   // Render status
   path := "/list"  
   ctx.Redirect(http.StatusFound, path)
}

func CompletedRate(ctx *gin.Context){
	userID := sessions.Default(ctx).Get("user")
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
	oneWeekAgo := time.Now().AddDate(0,0,-7)


	var total int
    query := "SELECT COUNT(*) FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ? AND created_at >= ?"
	err = db.QueryRow(query ,userID,oneWeekAgo).Scan(&total)
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}
	var completed int
	err = db.QueryRow(query +" AND is_done = 1",userID,oneWeekAgo).Scan(&completed)
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	var completed_rate int
	if total > 0{
		completed_rate = completed * 100 / total
	}else{
		completed_rate = 0
	}
    

    ctx.HTML(http.StatusOK, "completed_rate.html", gin.H{"Title": "Completed Rate", "Total": total, "Completed": completed, "Completed_rate" : completed_rate})

}