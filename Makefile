dev: killdebug buffalo adminer migrate grifts

migrate: db
	docker-compose run --rm buffalo whenavail db 5432 10 buffalo-pop pop migrate up

grifts: db
	docker-compose run --rm buffalo /bin/bash -c "buffalo task db:seed && buffalo task minio:seed"

migratestatus: db
	docker-compose run --rm buffalo buffalo-pop pop migrate status

migratetestdb: testdb
	docker-compose run --rm test whenavail testdb 5432 10 buffalo-pop pop migrate up

adminer:
	docker-compose up -d adminer

buffalo: db
	docker-compose up -d buffalo

debug: killbuffalo killdebug rmdebug
	docker-compose up -d debug
	docker-compose logs -f debug

swagger: swaggerspec
	docker-compose run --rm --service-ports swagger swagger serve -p 8082 --no-open swagger.json

swaggerspec:
	docker-compose run --rm swagger swagger generate spec -m -o swagger.json

bounce: db
	docker-compose kill buffalo
	docker-compose rm buffalo
	docker-compose up -d buffalo

logs:
	docker-compose logs buffalo

minio:
	docker-compose up -d minio

db:
	docker-compose up -d db

testdb:
	docker-compose up -d testdb

rmtestdb:
	docker-compose kill testdb && docker-compose rm -f testdb

test: testdb minio
	rm -f application/migrations/schema.sql
	docker-compose run --rm test whenavail testdb 5432 10 buffalo test

testenv: rmtestdb migratetestdb
	@echo "\n\nIf minio hasn't been initialized, run buffalo task minio:seed\n"
	docker-compose run --rm test bash

killbuffalo:
	docker-compose kill buffalo

killdebug:
	docker-compose kill debug

rmdebug:
	docker-compose rm -f debug

clean:
	docker-compose kill
	docker-compose rm -f
	rm -f application/migrations/schema.sql

fresh: clean dev
