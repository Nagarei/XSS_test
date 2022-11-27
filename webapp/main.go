package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"golang.org/x/crypto/bcrypt"
)

const (
	httpOnly     = false
	secureCookie = false
)

var (
	db                  *sqlx.DB
	sessionStore        sessions.Store
	mySQLConnectionData *MySQLConnectionEnv
)

//util
func getEnv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}

//render
type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func init() {
	sessionStore = sessions.NewCookieStore([]byte(getEnv("SESSION_KEY", "xsstest")))
}

func main() {
	e := echo.New()
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)

	t := &Template{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
	e.Renderer = t

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	//user
	e.GET("/", func(c echo.Context) error { return c.Redirect(http.StatusFound, "/product") })
	e.GET("/product", getProduct)
	e.POST("/comment", postComment)
	e.GET("/media/keyboard.png", func(c echo.Context) error { return c.File("media/keyboard.png") })

	//admin
	e.GET("/login", getLogin)
	e.POST("/login", postLogin)
	adminAPI := e.Group("/admin")
	adminAPI.Use(echo.MiddlewareFunc(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session, err := getSession(c.Request())
			if err != nil {
				return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get session: %v", err))
			}
			userName, ok := session.Values["user_name"]
			if !ok {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			if userName != "admin" {
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			return h(c)
		}
	}))
	adminAPI.GET("/comment", getAdminComment)
	adminAPI.POST("/approve", postAdminApprove)
	adminAPI.GET("/secret", getAdminSecret)

	mySQLConnectionData = NewMySQLConnectionEnv()

	var err error
	db, err = mySQLConnectionData.ConnectDB()
	if err != nil {
		e.Logger.Fatalf("failed to connect db: %v", err)
		return
	}
	db.SetMaxOpenConns(20)
	defer db.Close()
	err = initDB()
	if err != nil {
		e.Logger.Fatalf("failed to init db: %v", err)
		return
	}

	serverPort := fmt.Sprintf(":%v", getEnv("SERVER_APP_PORT", "3000"))
	e.Logger.Fatal(e.Start(serverPort))
}

func getSession(r *http.Request) (*sessions.Session, error) {
	session, err := sessionStore.Get(r, "xsstest")
	if err != nil {
		return nil, err
	}
	return session, nil
}

//
//user
//

func getProduct(c echo.Context) error {

	commentList := []ModelComment{}
	err := db.Select(
		&commentList,
		"SELECT * FROM `comments` WHERE `admitted` = 1 ORDER BY `id` ASC")
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	commentListStr := ""
	for _, comment := range commentList {
		commentListStr += `<p class="comment">` + comment.Comment + "<br>@" + comment.Name + "</p>\n"
	}

	comment_posted := ""
	if c.QueryParam("posted") == "true" {
		comment_posted = "<div><font color=\"red\">Your comment is pending admin approval.</font></div>"
	}

	//セッションを付けておく(ヒント)
	session, err := getSession(c.Request())
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get session: %v", err))
	}
	if _, ok := session.Values["user_name"]; !ok {
		session.Options = &sessions.Options{
			//有効な時間(sec)
			MaxAge: 3600,
			//trueでjsからのアクセス拒否
			HttpOnly: httpOnly,
			Secure:   secureCookie,
		}
		session.Values["user_name"] = "guest"
	}
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to save session: %v", err))
	}

	return c.Render(http.StatusOK, "product.html", map[string]interface{}{
		"comments":       commentListStr,
		"comment_posted": comment_posted,
	})
}

func postComment(c echo.Context) error {

	name := c.FormValue("name")
	comment := c.FormValue("comment")
	_, err := db.Exec(
		`INSERT INTO comments (name, comment) VALUES (?, ?)`,
		name, comment,
	)
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.Redirect(http.StatusSeeOther, "/product?posted=true")
}

//
//admin
//

func getLogin(c echo.Context) error {
	session, err := getSession(c.Request())
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get session: %v", err))
	}
	userName, ok := session.Values["user_name"]
	if !ok {
		return c.File("views/login.html")
	}
	if userName != "admin" {
		return c.File("views/login.html")
	}
	return c.Redirect(http.StatusSeeOther, "/admin/comment")
}
func postLogin(c echo.Context) error {
	password := c.FormValue("password")
	correct_password, err := bcrypt.GenerateFromPassword([]byte("supersecurepassword"), 10) //パスワードのハードコードは本当は禁止
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	if bcrypt.CompareHashAndPassword(correct_password, []byte(password)) != nil {
		return c.String(http.StatusForbidden, "forbidden")
	}

	session, err := getSession(c.Request())
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get session: %v", err))
	}
	session.Options = &sessions.Options{
		//有効な時間(sec)
		MaxAge: 3600,
		//trueでjsからのアクセス拒否
		HttpOnly: httpOnly,
		Secure:   secureCookie,
	}
	session.Values["user_name"] = "admin"
	err = session.Save(c.Request(), c.Response())
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to save session: %v", err))
	}

	return c.Redirect(http.StatusSeeOther, "/admin/comment")
}

func getAdminComment(c echo.Context) error {

	commentList := []ModelComment{}
	err := db.Select(
		&commentList,
		"SELECT * FROM `comments` ORDER BY `id` ASC")
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	commentListStr := ""
	for _, comment := range commentList {
		if comment.Admitted == 0 {
			commentListStr += `<form method="POST" action="/admin/approve"><p class="penddingcomment">`
		} else {
			commentListStr += `<p class="comment">`
		}
		commentListStr += comment.Comment + "<br>@" + comment.Name
		if comment.Admitted == 0 {
			commentListStr += `<br><input type="hidden" name="id" value="` + fmt.Sprint(comment.ID) + `">
			<button type="submit">approve</button></p></form>`
		} else {
			commentListStr += "</p>"
		}
	}

	return c.Render(http.StatusOK, "admin_comment.html", map[string]interface{}{
		"comments": commentListStr,
	})
}

func postAdminApprove(c echo.Context) error {
	id := c.FormValue("id")
	_, err := db.Exec("UPDATE comments SET admitted=1 WHERE id=?", id)
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.Redirect(http.StatusSeeOther, "/admin/comment")
}

func getAdminSecret(c echo.Context) error {
	return c.File("views/secret.html")
}
