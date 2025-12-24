module glib/example/fullstack

go 1.25.1

replace github.com/azizndao/glib => ../../http

replace github.com/azizndao/glib/common => ../../common

replace github.com/azizndao/glib/database => ../../database

replace github.com/azizndao/glib/foundation => ../../foundation

replace github.com/azizndao/glib/validation => ../../validation

require (
	github.com/azizndao/glib v0.0.0
	github.com/azizndao/glib/common v0.0.0
	github.com/azizndao/glib/database v0.0.0
	github.com/azizndao/glib/foundation v0.0.0
	github.com/azizndao/glib/validation v0.0.0
	github.com/google/uuid v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.10 // indirect
	github.com/go-chi/chi/v5 v5.2.3 // indirect
	github.com/go-chi/cors v1.2.2 // indirect
	github.com/go-chi/httplog/v3 v3.3.0 // indirect
	github.com/go-chi/httprate v0.15.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.28.0 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	github.com/samber/lo v1.52.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
	gorm.io/driver/sqlite v1.6.0 // indirect
)
