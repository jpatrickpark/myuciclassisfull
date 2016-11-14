package handlers

import (
	"encoding/json"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/jpatrickpark/server1/libhttp"
	"github.com/jpatrickpark/server1/models"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	spring        = "-14"
	summer1       = "-25"
	summer10      = "-39"
	summerCom     = "-51"
	summer2       = "-76"
	fall          = "-92"
	winter        = "-03"
	urlFirstHalf  = "https://www.reg.uci.edu/perl/WebSoc?YearTerm="
	urlSecondHalf = "&ShowFinals=0&ShowComments=0&CourseCodes="
)

func PossibleQuarters(now time.Time) []string {
	// Generate a list of all the quarters that might be open for the students at the moment
	year := now.String()[0:4]
	nextYear := now.Year() + 1
	var possibleQuarters []string
	switch now.Month() {
	case time.February:
		possibleQuarters = []string{year + spring}
	case time.March:
		possibleQuarters = []string{year + spring, year + summer1,
			year + summer10, year + summerCom, year + summer2}
	case time.April:
		possibleQuarters = []string{year + summer1,
			year + summer10, year + summerCom, year + summer2}
	case time.May:
		fallthrough
	case time.June:
		fallthrough
	case time.July:
		possibleQuarters = []string{year + summer1, year + summer10,
			year + summerCom, year + summer2, year + fall}
	case time.August:
		possibleQuarters = []string{year + summer2, year + fall}
	case time.September:
		fallthrough
	case time.October:
		possibleQuarters = []string{year + fall}
	case time.November:
		fallthrough
	case time.December:
		possibleQuarters = []string{strconv.Itoa(nextYear) + winter}
	case time.January:
		possibleQuarters = []string{year + winter}
	default:
	}
	return possibleQuarters
}
func CurrentQuarter(now time.Time) string {
	// Indicate either students are eligible for spring, fall, or winter quarter
	switch now.Month() {
	case time.February:
		fallthrough
	case time.March:
		return now.String()[0:4] + spring
	case time.April:
		fallthrough
	case time.May:
		fallthrough
	case time.June:
		fallthrough
	case time.July:
		fallthrough
	case time.August:
		fallthrough
	case time.September:
		fallthrough
	case time.October:
		return now.String()[0:4] + fall
	case time.November:
		fallthrough
	case time.December:
		return strconv.Itoa(now.Year()+1) + winter
	case time.January:
		return now.String()[0:4] + winter
	default:
		return ""
	}
}
func Contains(items []string, target string) bool {
	return Find(items, target) != -1
}
func Find(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return -1
}
func Readable(quarter string) string {
	// Convert 7-digit string 'quarter' into human readable format
	var readable string
	switch quarter[4:7] {
	case spring:
		readable = quarter[0:4] + " Spring"
	case summer1:
		readable = quarter[0:4] + " Summer Session 1"
	case summer10:
		readable = quarter[0:4] + " 10-wk Summer"
	case summerCom:
		readable = quarter[0:4] + " Summer Qtr (COM)"
	case summer2:
		readable = quarter[0:4] + " Summer Session 2"
	case fall:
		readable = quarter[0:4] + " Fall"
	case winter:
		readable = quarter[0:4] + " Winter"
	default:
		readable = "error"
	}
	return readable
}
func GetTerm(w http.ResponseWriter, r *http.Request) {
	// Display information about the given term
	// Validation is conducted in GetUciClass after redirection
	currentQuarter := mux.Vars(r)["quarter"]
	sessionStore := context.Get(r, "sessionStore").(sessions.Store)

	session, _ := sessionStore.Get(r, "server1-session")
	session.Values["currentQuarter"] = currentQuarter
	err := session.Save(r, w)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}
	http.Redirect(w, r, "/my-uci-class-is-full", 302)
}

type PutDeleteTermResponse struct {
	Status  int                `json:"status"`
	Courses []models.CourseRow `json:"courses"`
}

func CourseStatus(currentQuarter, courseCode string) int {
	// Get current status of a course from web
	resp, _ := http.Get(urlFirstHalf + currentQuarter + urlSecondHalf + courseCode)
	byteResp, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	stringResp := string(byteResp)
	if strings.Contains(stringResp, "FULL") {
		return models.FULL
	}
	if strings.Contains(stringResp, "OPEN") {
		return models.OPEN
	}
	if strings.Contains(stringResp, "No courses matched") {
		return models.NONEXISTENT
	}
	return models.WAITLIST
}
func DeleteTerm(w http.ResponseWriter, r *http.Request) {
	// Remove user's request for a given course for a given term
	w.Header().Set("Content-Type", "application/json")

	//Get Variables
	quarter := mux.Vars(r)["quarter"]
	courseCode := mux.Vars(r)["courseCode"]

	//Get user information and DB from Session and Context
	sessionStore := context.Get(r, "sessionStore").(sessions.Store)
	_, _ = quarter, courseCode
	session, _ := sessionStore.Get(r, "server1-session")
	currentUser, ok := session.Values["user"].(*models.UserRow)
	if !ok {
		http.Redirect(w, r, "/logout", 302)
		return
	}
	db := context.Get(r, "db").(*sqlx.DB)

	//Start constructing JSON object to return
	structResponse := PutDeleteTermResponse{}

	//Try deleting the given user-course pair
	structResponse.Status = models.NewUserCoursePair(db).RemoveUserCoursePair(nil, currentUser.ID, courseCode, quarter)

	//Return the list of user-course pair after the deletion
	courses, err := models.NewCourse(db).GetCoursesByUserIdAndQuarter(nil, currentUser.ID, quarter)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}
	structResponse.Courses = *courses

	jsonResponse, err := json.Marshal(structResponse)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}
	w.Write(jsonResponse)
}
func PutTerm(w http.ResponseWriter, r *http.Request) {
	// Record user's request for a given course for a given term
	w.Header().Set("Content-Type", "application/json")
	courseCode := r.FormValue("courseCode")
	currentQuarter := mux.Vars(r)["quarter"]
	sessionStore := context.Get(r, "sessionStore").(sessions.Store)

	session, _ := sessionStore.Get(r, "server1-session")
	currentUser, ok := session.Values["user"].(*models.UserRow)
	if !ok {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	db := context.Get(r, "db").(*sqlx.DB)

	// Construct JSON object for response
	structResponse := PutDeleteTermResponse{}
	structResponse.Status = CourseStatus(currentQuarter, courseCode)
	var exists bool
	if structResponse.Status != models.NONEXISTENT {
		requestedCourse, err := models.NewCourse(db).AddCourse(nil, structResponse.Status, courseCode, currentQuarter)
		if err != nil {
			libhttp.HandleErrorJson(w, err)
			return
		}
		_, err, exists = models.NewUserCoursePair(db).AddUserCoursePair(nil, requestedCourse.ID, currentUser.ID)
		if err != nil {
			libhttp.HandleErrorJson(w, err)
			return
		}
	}
	// If a duplicate entry already exists, indicate that in response
	if exists {
		structResponse.Status = models.ENTRYEXISTS
	}
	// Get current list of requested courses for the user for the given term.
	courses, err := models.NewCourse(db).GetCoursesByUserIdAndQuarter(nil, currentUser.ID, currentQuarter)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}
	structResponse.Courses = *courses

	jsonResponse, err := json.Marshal(structResponse)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}
	w.Write(jsonResponse)
}
func GetUciClass(w http.ResponseWriter, r *http.Request) {
	// Serve the main page for the single page web application
	w.Header().Set("Content-Type", "text/html")

	sessionStore := context.Get(r, "sessionStore").(sessions.Store)

	session, _ := sessionStore.Get(r, "server1-session")
	currentUser, ok := session.Values["user"].(*models.UserRow)
	if !ok {
		http.Redirect(w, r, "/logout", 302)
		return
	}

	now := time.Now()

	// generate list of quarters that are eligible for the current month.
	possibleQuarters := PossibleQuarters(now)

	// Validate currentQuarter
	var currentQuarter string
	currentQuarter, ok = session.Values["currentQuarter"].(string)
	if !ok || !Contains(possibleQuarters, currentQuarter) {
		currentQuarter = CurrentQuarter(now)
		session.Values["currentQuarter"] = currentQuarter
		err := session.Save(r, w)
		if err != nil {
			libhttp.HandleErrorJson(w, err)
			return
		}
	}

	// Get list of users requested courses
	db := context.Get(r, "db").(*sqlx.DB)
	courses, err := models.NewCourse(db).GetCoursesByUserIdAndQuarter(nil, currentUser.ID, currentQuarter)
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	// This part of the code is to be able to handle cases
	// where there are multiple quarters that are open to the students at a moment.
	// In that case, students will be able to navigate using 'prev', 'next' buttons.
	// Currently under development but has low priority.
	index := Find(possibleQuarters, currentQuarter)
	length := len(possibleQuarters)
	var prev, next string
	switch {
	case index <= 0:
		prev = ""
	case index > 0:
		prev = possibleQuarters[index-1]

	}
	switch {
	case index < 0:
		fallthrough
	case index >= length-1:
		next = ""
	case index < length-1:
		next = possibleQuarters[index+1]
	}

	data := struct {
		CurrentUser            *models.UserRow
		CurrentQuarter         string
		CurrentQuarterReadable string
		Prev                   string
		Next                   string
		ExistsPrev             bool
		ExistsNext             bool
		Courses                *[]models.CourseRow
	}{
		currentUser, currentQuarter, Readable(currentQuarter), prev, next, prev != "", next != "", courses,
	}

	tmpl, err := template.ParseFiles("templates/dashboard.html.tmpl", "templates/uci.html.tmpl")
	if err != nil {
		libhttp.HandleErrorJson(w, err)
		return
	}

	tmpl.Execute(w, data)
}
