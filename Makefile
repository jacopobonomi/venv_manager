.PHONY: build test install lint clean demo

build:
	go build -o bin/venv-manager cmd/venv-manager/main.go

install:
	./scripts/install.sh

lint:
	golangci-lint run

clean:
	rm -rf bin/

# Regenerate scripts/demo/demo.gif. Requires: vhs, python3.
# Builds a fresh venv-manager binary and puts it first on PATH so the tape
# always exercises the current code (not whatever is installed globally).
demo: build
	@echo ">> preparing demo workdir"
	@rm -rf /tmp/vm-demo-work
	@mkdir -p /tmp/vm-demo-work
	@cp scripts/demo/ai_writes_code.sh /tmp/vm-demo-work/
	@chmod +x /tmp/vm-demo-work/ai_writes_code.sh
	@PATH=$(CURDIR)/bin:$$PATH venv-manager remove vhs-demo >/dev/null 2>&1 || true
	@PATH=$(CURDIR)/bin:$$PATH venv-manager create vhs-demo >/dev/null
	@echo ">> running vhs"
	@cd /tmp/vm-demo-work && PATH=$(CURDIR)/bin:$$PATH vhs $(CURDIR)/scripts/demo/demo.tape
	@mv /tmp/vm-demo-work/demo.gif scripts/demo/demo.gif
	@echo ">> cleaning up"
	@PATH=$(CURDIR)/bin:$$PATH venv-manager remove vhs-demo >/dev/null 2>&1 || true
	@rm -rf /tmp/vm-demo-work
	@echo ">> demo.gif ready at scripts/demo/demo.gif"