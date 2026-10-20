BACKEND_DIR = backend
FRONTEND_DIR = frontend
FRONTEND_DIST_DIR = backend/cmd/frontend/dist
BINARY = nx-admin

.PHONY: all build build-frontend build-backend dev-backend dev-backend-air dev-frontend clean

all: build

build-frontend:
	cd $(FRONTEND_DIR) && npm ci && npm run build
	mkdir -p $(FRONTEND_DIST_DIR)
	find $(FRONTEND_DIST_DIR) -mindepth 1 ! -name .gitkeep -delete
	cp -r $(FRONTEND_DIR)/dist/* $(FRONTEND_DIST_DIR)/

build-backend:
	cd $(BACKEND_DIR) && go build -o ../$(BINARY) ./cmd

build: build-frontend build-backend

dev-backend:
	cd $(BACKEND_DIR) && go run ./cmd

dev-backend-air:
	air

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

clean:
	rm -rf $(BINARY)
	find $(FRONTEND_DIR)/dist $(FRONTEND_DIST_DIR) -mindepth 1 ! -name .gitkeep -delete 2>/dev/null || true
