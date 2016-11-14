package application

import (
	"github.com/carbocation/interpose"
	gorilla_mux "github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"time"

	"github.com/jpatrickpark/server1/handlers"
	"github.com/jpatrickpark/server1/middlewares"
	"github.com/jpatrickpark/server1/models"
)

func SendCourseOpenEmail(courseCode, quarter, email string) {
	from := mail.NewEmail("My UCI Class Is Full", "myuciclassisfull@gmail.com")
	to := mail.NewEmail(email, email)
	title := "Your course " + courseCode + " is available!"
	content := "<p>Your course " + courseCode + " for " + handlers.Readable(quarter) + " quarter is available!</p><p>Go ahead and enroll in now on <a href='https://www.reg.uci.edu'>Webreg</a>!</p>"
	newContent := mail.NewContent("text/html", content)
	message := mail.NewV3MailInit(from, title, to, newContent)
	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = mail.GetRequestBody(message)
	// I should think more about what to do when the email fails to send.
	sendgrid.API(request)
}
func SendToAccordingUsers(db *sqlx.DB, courseId int64, courseCode, quarter string) {
	pair := models.NewUserCoursePair(db)
	userStruct := models.NewUser(db)
	pairs, err1 := pair.GetPairsByCourseId(nil, courseId)
	if err1 == nil {
		for _, item := range *pairs {
			user, err2 := userStruct.GetById(nil, item.UserID)
			if err2 == nil {
				SendCourseOpenEmail(courseCode, quarter, user.Email)
			}
		}
	}
}
func My_uci_class_is_full(db *sqlx.DB) {
	for {
		course := models.NewCourse(db)
		courses, err := course.AllCourses(nil)
		if err == nil {
			now := time.Now()
			//now = time.Date(2016, time.March, 10, 23, 0, 0, 0, time.UTC)
			possibleQuarters := handlers.PossibleQuarters(now)
			for _, item := range courses {
				if handlers.Contains(possibleQuarters, item.Quarter) {
					newStatus := handlers.CourseStatus(item.Quarter, item.CourseCode)
					if item.Status != newStatus {
						course.UpdateCourse(nil, item.ID, newStatus)
						if item.Status == models.FULL && (newStatus == models.OPEN || newStatus == models.WAITLIST) {
							go SendToAccordingUsers(db, item.ID, item.CourseCode, item.Quarter)
						}
					}
				}
			}
		}
		time.Sleep(time.Minute)
	}
}

// New is the constructor for Application struct.
func New(config *viper.Viper) (*Application, error) {
	dsn := config.Get("dsn").(string)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	go My_uci_class_is_full(db)

	cookieStoreSecret := config.Get("cookie_secret").(string)

	app := &Application{}
	app.config = config
	app.dsn = dsn
	app.db = db
	app.sessionStore = sessions.NewCookieStore([]byte(cookieStoreSecret))
	return app, err
}

// Application is the application object that runs HTTP server.
type Application struct {
	config       *viper.Viper
	dsn          string
	db           *sqlx.DB
	sessionStore sessions.Store
}

func (app *Application) MiddlewareStruct() (*interpose.Middleware, error) {
	middle := interpose.New()
	middle.Use(middlewares.SetDB(app.db))
	middle.Use(middlewares.SetSessionStore(app.sessionStore))

	middle.UseHandler(app.mux())

	return middle, nil
}

func (app *Application) mux() *gorilla_mux.Router {
	MustLogin := middlewares.MustLogin

	router := gorilla_mux.NewRouter()

	router.Handle("/search-golang", MustLogin(http.HandlerFunc(handlers.GetHome))).Methods("GET")
	router.Handle("/my-uci-class-is-full", MustLogin(http.HandlerFunc(handlers.GetUciClass))).Methods("GET")
	router.Handle("/whiteboard", MustLogin(http.HandlerFunc(handlers.GetWhiteboardHome))).Methods("GET")

	router.HandleFunc("/signup", handlers.GetSignup).Methods("GET")
	router.HandleFunc("/signup", handlers.PostSignup).Methods("POST")
	router.HandleFunc("/login", handlers.GetLogin).Methods("GET")
	router.HandleFunc("/login", handlers.PostLogin).Methods("POST")
	router.HandleFunc("/logout", handlers.GetLogout).Methods("GET")
	router.HandleFunc("/search-golang/intersectRepo", handlers.PostIntersectRepo).Methods("Post")
	router.HandleFunc("/search-golang/intersectHuman", handlers.PostIntersectHuman).Methods("Post")
	router.HandleFunc("/search-golang/search", handlers.GetSearch).Methods("GET")

	router.Handle("/my-uci-class-is-full/term/{quarter}", MustLogin(http.HandlerFunc(handlers.PutTerm))).Methods("PUT")
	router.Handle("/my-uci-class-is-full/term/{quarter}/{courseCode}", MustLogin(http.HandlerFunc(handlers.DeleteTerm))).Methods("DELETE")
	router.Handle("/my-uci-class-is-full/term/{quarter}", MustLogin(http.HandlerFunc(handlers.GetTerm))).Methods("GET")
	router.Handle("/users/{id:[0-9]+}", MustLogin(http.HandlerFunc(handlers.PostPutDeleteUsersID))).Methods("POST", "PUT", "DELETE")

	// Path of static files must be last!
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return router
}
