module github.com/azizndao/glib/example/orm

go 1.25.1

replace github.com/azizndao/glib => ../../http

replace github.com/azizndao/glib/common => ../../common

replace github.com/azizndao/glib/database => ../../database

require (
	github.com/azizndao/glib/database v0.0.0
	github.com/google/uuid v1.6.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	golang.org/x/text v0.32.0 // indirect
)
