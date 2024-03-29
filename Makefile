all: build
GO := GOPATH=`pwd`:$(GOPATH) go

.PHONY: help
help:
	@echo '-- Commands --'
	@echo 'build          -- Build app'
	@echo 'restart        -- eval ~/restart'
	@echo 'applog         -- eval ~/applog'
	@echo 'deploy         -- eval ~/deploy $(CURDIR)'
	@echo 'report         -- eval ~/make_report $(CURDIR)/app'

.PHONY: build
build:
	$(GO) build -o app app

.PHONY: race
race:
	$(GO) build -race -o app app

.PHONY: restart
restart:
	$(HOME)/restart

.PHONY: applog
applog:
	$(HOME)/applog

.PHONY: deploy
deploy:
	$(HOME)/deploy $(CURDIR)

.PHONY: report
report:
	$(HOME)/make_report $(CURDIR)/app

