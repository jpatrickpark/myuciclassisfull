# Documentations for My UCI Class is Full

##REST
This app uses RESTful URL to receive requests and display ajax responses.

However, users must submit request in the browser through the web app because user information is saved in session.

Verb	|URL	|Action
---|---|---
PUT	|/term/{quarter}	|Puts a course to the user area designated for the given term. If a given term or course code is invalid, it responds with an appropriate status code to notify the user through ajax.
DELETE	|/term/{quarter}/{courseCode}	|Deletes user request for the given course of the given quarter.
GET	|/	|Gets the html document with user information included.
GET	|/term/{quarter}	|Gets the html document for the given term. If the given term is invalid or is not open for students at the moment, it ignores the given term and generates an html document for the current term.
{quarter} is a length 7 string that indicates a specific quarter. Example: 2017-03

{courseCode} is a length 5 string that indicates a specific course code. Example: 20025

A {courseCode} is only distinct within a quarter; a {courseCode} may be used again in the next quarter for a different class.

The app checks this URL for the status of each course: https://www.reg.uci.edu/perl/WebSoc?YearTerm={quarter}&ShowFinals=0&ShowComments=0&CourseCodes={courseCode}

Example:https://www.reg.uci.edu/perl/WebSoc?YearTerm=2017-03&ShowFinals=0&ShowComments=0&CourseCodes=20025

##What It Actually Does
This app sends notification email using SendGrid whenever a course changes its status from full to open or waitlist.

It saves the status of the requested courses in the database and compares with the school website every minute.

##Databases
###Courses
id | courseCode | status | quarter
---|---|---|---
BIGSERIAL | TEXT | INT | TEXT
###User_Course_Pairs
id | course_id | user_id
---|---|---
BIGSERIAL | BIGSERIAL | BIGSERIAL
###Users
id | email
---|---
BIGSERIAL | TEXT
