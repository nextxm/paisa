.PHONY: docs
.PHONY: fixture/main.transactions.json

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X 'github.com/ananthakumaran/paisa/cmd.Version=$(VERSION)'"
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

develop:
	./node_modules/.bin/concurrently --names "GO,JS" -c "auto" "make serve" "npm run dev"

serve:
	./node_modules/.bin/nodemon --signal SIGTERM --delay 2000ms --watch '.' --ext go,json --exec 'go run . serve || exit 1'

debug:
	./node_modules/.bin/concurrently --names "GO,JS" -c "auto" "make serve-now" "npm run dev"

serve-now:
	./node_modules/.bin/nodemon --signal SIGTERM --delay 2000ms --watch '.' --ext go,json --exec 'TZ=UTC go run . serve --now 2022-02-07 || exit 1'


watch:
	npm run "build:watch"
docs:
	mkdocs serve -a 0.0.0.0:8000

sample:
	go build $(LDFLAGS) && ./paisa init && ./paisa update

publish:
	nix develop --command bash -c 'mkdocs build'

parser:
	npm run parser-build-debug

proto:
	PATH="$(PWD)/node_modules/.bin:$(PATH)" protoc \
	  --proto_path=proto \
	  --go_out=internal/gen \
	  --go_opt=module=github.com/ananthakumaran/paisa/internal/gen \
	  --connect-go_out=internal/gen \
	  --connect-go_opt=module=github.com/ananthakumaran/paisa/internal/gen \
	  --es_out=src/lib/gen \
	  --es_opt=target=ts \
	  proto/api.proto

sqlc-generate:
	$(SQLC) generate

lint:
	$(MAKE) sqlc-generate
	./node_modules/.bin/prettier --check src
	npm run check
	test -z "$$(gofmt -l .)"

regen:
	go build $(LDFLAGS)
	unset PAISA_CONFIG && REGENERATE=true TZ=UTC bun test tests

jstest:
	bun test --preload ./src/happydom.ts src
	go build $(LDFLAGS)
	unset PAISA_CONFIG && TZ=UTC bun test tests

jsbuild:
	npm run build

test: sqlc-generate jsbuild jstest
	go test ./...

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build $(LDFLAGS)


deploy:
	fly scale count 2 --region lax --yes
	docker build -t paisa . --file Dockerfile.demo
	fly deploy -i paisa:latest --local-only
	fly scale count 1 --region lax --yes

install:
	npm run build
	go build $(LDFLAGS)
	go install $(LDFLAGS)

fixture/main.transactions.json:
	cd /tmp && paisa init
	cp fixture/main.ledger /tmp/main.ledger
	cd /tmp && paisa update --journal && paisa serve -p 6500 &
	sleep 1
	curl http://localhost:6500/api/transaction | jq .transactions > fixture/main.transactions.json
	pkill -f 'paisa serve -p 6500'

generate-fonts:
	bun download-svgs.js
	node generate-font.js

node2nix:
	npm install --lockfile-version 2
	node2nix --development -18 --input package.json \
	--lock package-lock.json \
	--node-env ./flake/node-env.nix \
	--composition ./flake/default.nix \
	--output ./flake/node-package.nix
