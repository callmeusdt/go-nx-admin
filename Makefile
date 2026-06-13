BACKEND_DIR = backend
FRONTEND_DIR = frontend
FRONTEND_DIST_DIR = backend/cmd/frontend/dist
BINARY = nx-admin

.PHONY: all build dev-backend dev-frontend clean

all: build

build-frontend:
	cd $(FRONTEND_DIR) && npm install && npm run build
	rm -rf $(FRONTEND_DIST_DIR)
	mkdir -p $(FRONTEND_DIST_DIR)
	cp -r $(FRONTEND_DIR)/dist/* $(FRONTEND_DIST_DIR)/

build-backend:
	cd $(BACKEND_DIR) && go mod tidy && go build -o ../$(BINARY) ./cmd

build: build-frontend build-backend

dev-backend:
	cd $(BACKEND_DIR) && go run ./cmd

dev-backend-air:
	cd $(BACKEND_DIR) && air

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

clean:
	rm -rf $(BINARY) $(FRONTEND_DIR)/dist $(FRONTEND_DIST_DIR)
