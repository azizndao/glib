module github.com/azizndao/glib/example/cache-api

go 1.25.1

replace (
	github.com/azizndao/glib/cache => ../../cache
	github.com/azizndao/glib/common => ../../common
)

require github.com/azizndao/glib/cache v0.0.0

require github.com/azizndao/glib/common v0.0.0-00010101000000-000000000000 // indirect
