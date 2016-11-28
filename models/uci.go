package models

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
)

const (
	PairTableName    = "user_course_pair"
	CourseTableName  = "courses"
	FULL             = 0
	OPEN             = 1
	WAITLIST         = 2
	NONEXISTENT      = 3
	DELETED          = 4
	ENTRYEXISTS      = 5
	NOTDELETED       = 6
	NEWONLY_FULL     = 7
	NEWONLY_WAITLIST = 8
)

func NewCourse(db *sqlx.DB) *Course {
	course := &Course{}
	course.db = db
	course.table = CourseTableName
	course.hasID = true

	return course
}
func NewUserCoursePair(db *sqlx.DB) *UserCoursePair {
	pair := &UserCoursePair{}
	pair.db = db
	pair.table = PairTableName
	pair.hasID = true

	return pair
}

type CourseRow struct {
	ID         int64  `db:"id" json:"courseId"`
	CourseCode string `db:"coursecode" json:"courseCode"`
	Status     int    `db:"status" json:"courseStatus"`
	Quarter    string `db:"quarter" json:"quarter"`
}
type UserCoursePairRow struct {
	ID       int64 `db:"id"`
	CourseID int64 `db:"course_id"`
	UserID   int64 `db:"user_id"`
}

type Course struct {
	Base
}
type UserCoursePair struct {
	Base
}

func (u *Course) courseRowFromSqlResult(tx *sqlx.Tx, sqlResult sql.Result) (*CourseRow, error) {
	courseId, err := sqlResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	return u.GetCourseById(tx, courseId)
}
func (u *UserCoursePair) userCoursePairRowFromSqlResult(tx *sqlx.Tx, sqlResult sql.Result) (*UserCoursePairRow, error) {
	courseId, err := sqlResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	return u.GetPairById(tx, courseId)
}

func (u *UserCoursePair) GetPairById(tx *sqlx.Tx, id int64) (*UserCoursePairRow, error) {
	pair := &UserCoursePairRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE id=$1", u.table)
	err := u.db.Get(pair, query, id)

	return pair, err
}

func (u *Course) GetCourseById(tx *sqlx.Tx, id int64) (*CourseRow, error) {
	course := &CourseRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE id=$1", u.table)
	err := u.db.Get(course, query, id)

	return course, err
}

// AllCourses returns all course rows.
func (u *Course) AllCourses(tx *sqlx.Tx) ([]*CourseRow, error) {
	courses := []*CourseRow{}
	query := fmt.Sprintf("SELECT * FROM %v", u.table)
	err := u.db.Select(&courses, query)

	return courses, err
}

func (p *UserCoursePair) GetPairsByCourseId(tx *sqlx.Tx, courseId int64) (*[]UserCoursePairRow, error) {
	pairs := &[]UserCoursePairRow{}

	query := fmt.Sprintf("SELECT * FROM %v WHERE course_id=$1", p.table)
	err := p.db.Select(pairs, query, courseId)

	return pairs, err
}

func (u *Course) GetCoursesByUserIdAndQuarter(tx *sqlx.Tx, userId int64, quarter string) (*[]CourseRow, error) {
	courses := &[]CourseRow{}

	//fix P C
	query := fmt.Sprintf("SELECT courses.id, courses.coursecode, courses.status, courses.quarter FROM %v, %v WHERE user_course_pair.user_id=$1 AND user_course_pair.course_id = courses.id AND courses.quarter=$2", u.table, PairTableName)
	err := u.db.Select(courses, query, userId, quarter)

	return courses, err
}

func (u *Course) GetCourseByCourseCodeAndQuarter(tx *sqlx.Tx, code, quarter string) (*CourseRow, error) {
	course := &CourseRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE coursecode=$1 AND quarter=$2", u.table)
	err := u.db.Get(course, query, code, quarter)

	return course, err
}
func (p *UserCoursePair) GetPairByCourseIdAndUserId(tx *sqlx.Tx, courseId, userId int64) (*UserCoursePairRow, error) {
	pair := &UserCoursePairRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE course_id=$1 AND user_id=$2", p.table)
	err := p.db.Get(pair, query, courseId, userId)

	return pair, err
}

func (u *Course) UpdateCourse(tx *sqlx.Tx, courseId int64, status int) {
	query := fmt.Sprintf("UPDATE %v SET status=$1 WHERE id=$2", u.table)
	u.db.Exec(query, status, courseId)
}

func (p *UserCoursePair) RemoveUserCoursePair(tx *sqlx.Tx, userId int64, code, quarter string) int {
	query := fmt.Sprintf("DELETE FROM %v P USING %v C WHERE P.user_id=$1 AND C.coursecode=$2 AND C.quarter=$3 AND C.id=P.course_id", p.table, CourseTableName)
	_, err := p.db.Exec(query, userId, code, quarter)

	if err != nil {
		return NOTDELETED
	}
	return DELETED
}

func (u *UserCoursePair) AddUserCoursePair(tx *sqlx.Tx, courseId, userId int64) (*UserCoursePairRow, error, bool) {
	if courseId <= 0 {
		return nil, errors.New("courseId must be bigger than 0."), false
	}
	if userId <= 0 {
		return nil, errors.New("userId must be bigger than 0."), false
	}
	previousEntry, err0 := u.GetPairByCourseIdAndUserId(nil, courseId, userId)
	if err0 == nil {
		return previousEntry, nil, true
	}

	data := make(map[string]interface{})
	data["course_id"] = courseId
	data["user_id"] = userId

	sqlResult, err1 := u.InsertIntoTable(tx, data)
	if err1 != nil {
		return nil, err1, false
	}

	confirmingEntry, err2 := u.userCoursePairRowFromSqlResult(tx, sqlResult)
	return confirmingEntry, err2, false
}

func (u *Course) AddCourse(tx *sqlx.Tx, status int, code, quarter string) (*CourseRow, error) {
	if code == "" {
		return nil, errors.New("Code cannot be blank.")
	}
	if quarter == "" {
		return nil, errors.New("Quarter cannot be blank.")
	}
	previousEntry, err0 := u.GetCourseByCourseCodeAndQuarter(nil, code, quarter)
	if err0 == nil {
		return previousEntry, nil
	}

	data := make(map[string]interface{})
	data["coursecode"] = code
	data["status"] = status
	data["quarter"] = quarter

	sqlResult, err1 := u.InsertIntoTable(tx, data)
	if err1 != nil {
		return nil, err1
	}

	return u.courseRowFromSqlResult(tx, sqlResult)
}
