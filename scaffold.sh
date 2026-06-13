#!/usr/bin/env bash
# scaffold.sh — 从 go-nx-admin 骨架创建新项目（Git Submodule 模式）
# 用法: 在已有 go-nx-admin submodule 的目录下执行
#   git submodule add https://github.com/xxx/go-nx-admin.git go-nx-admin
#   ./go-nx-admin/scaffold.sh <项目名>
#
# 脚本在当前目录（二开项目根目录）生成骨架文件

set -euo pipefail

PROJECT_NAME="${1:-}"
if [ -z "$PROJECT_NAME" ]; then
    echo "用法: $0 <项目名>"
    echo "示例: $0 my-admin"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$(dirname "$SCRIPT_DIR")"

if [ "$(basename "$SCRIPT_DIR")" != "go-nx-admin" ]; then
    echo "错误: scaffold.sh 必须在 go-nx-admin 仓库内执行"
    echo "请确保 go-nx-admin 已作为 submodule 放在项目根目录下:"
    echo "  git submodule add <url> go-nx-admin"
    echo "  ./go-nx-admin/scaffold.sh $PROJECT_NAME"
    exit 1
fi

if ! [ -d "$TARGET_DIR" ]; then
    echo "错误: 目标目录不存在: $TARGET_DIR"
    exit 1
fi

# 检查 go-nx-admin/backend 存在（submodule 已初始化）
if ! [ -f "$SCRIPT_DIR/backend/go.mod" ]; then
    echo "错误: go-nx-admin 子模块未完整初始化"
    echo "请执行: git submodule update --init --recursive"
    exit 1
fi

echo "==> 生成骨架文件到当前目录..."

cd "$TARGET_DIR"

# —— go.mod ——
if [ -f "go.mod" ]; then
    echo "跳过: go.mod 已存在"
else
    cat > go.mod << GOMOD
module ${PROJECT_NAME}

go 1.25

require (
	go-nx-admin v0.0.0
)

replace go-nx-admin => ./go-nx-admin/backend
GOMOD
    echo "  -> go.mod"
fi

# —— cmd/main.go ——
mkdir -p cmd
cat > cmd/main.go << GOMAIN
package main

import (
	"embed"
	"io/fs"
	"log"

	"go-nx-admin/app"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	sub, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		log.Fatalf("embedded dist not found, run 'make build-frontend' first: %v", err)
	}

	app.Run(app.Options{
		EmbedFS: sub,
		// 二开项目注入点：
		// ExtraModels: []interface{}{&models.MyModel{}},
		// ExtraRoutes: func(api fiber.Router, db *gorm.DB, enforcer *casbin.Enforcer) {
		//     api.Get("/orders", handlers.ListOrders(db))
		// },
		// AfterMigrate: func(db *gorm.DB) {
		//     // 二开自定义种子数据
		// },
		// ExtraPermissions: []app.PermissionDef{
		//     {Key: "orders:list", Name: "查看订单", Group: "订单管理"},
		// },
		// ExtraPermissionExpand: func(key string) [][]string {
		//     if key == "orders:list" { return [][]string{{"GET", "/api/v1/orders"}} }
		//     return nil
		// },
	})
}
GOMAIN
echo "  -> cmd/main.go"

# —— internal skeleton ——
mkdir -p internal/handlers internal/models
cat > internal/handlers/.gitkeep << 'KEEP'
KEEP
cat > internal/models/.gitkeep << 'KEEP'
KEEP
echo "  -> internal/ (目录已就绪，在此添加业务代码)"

# —— config.example.yaml ——
if [ -f "config.example.yaml" ]; then
    echo "跳过: config.example.yaml 已存在"
else
    cp "$SCRIPT_DIR/config.example.yaml" config.example.yaml
    echo "  -> config.example.yaml"
fi

# —— Makefile ——
cat > Makefile << MAKE
BACKEND_DIR = go-nx-admin/backend
FRONTEND_DIR = frontend
FRONTEND_DIST_DIR = cmd/frontend/dist
BINARY = ${PROJECT_NAME}

.PHONY: all build dev-backend dev-frontend clean submodule

all: build

submodule:
	git submodule update --init --recursive

build-frontend:
	cd \$(FRONTEND_DIR) && npm install && npm run build
	rm -rf \$(FRONTEND_DIST_DIR)
	mkdir -p \$(FRONTEND_DIST_DIR)
	cp -r \$(FRONTEND_DIR)/dist/* \$(FRONTEND_DIST_DIR)/

build-backend:
	cd \$(BACKEND_DIR) && go mod tidy
	go mod tidy
	go build -o \$(BINARY) ./cmd

build: build-frontend build-backend

dev-backend:
	go run ./cmd

dev-backend-air:
	cd \$(BACKEND_DIR) && air

dev-frontend:
	cd \$(FRONTEND_DIR) && npm run dev

clean:
	rm -rf \$(BINARY) \$(FRONTEND_DIR)/dist \$(FRONTEND_DIST_DIR)

# 升级底座
upgrade:
	git submodule update --remote go-nx-admin
	@echo "底座已升级，请检查变更并重新 make build"
MAKE
echo "  -> Makefile"

# —— .gitignore ——
if [ -f ".gitignore" ]; then
    echo "跳过: .gitignore 已存在"
else
    cat > .gitignore << GITIGNORE
# Binary
${PROJECT_NAME}

# Database
data/nx.db

# Frontend build
frontend/dist/*
!frontend/dist/.gitkeep

# Embed frontend dist
cmd/frontend/dist/*
!cmd/frontend/dist/.gitkeep

# Dependencies
frontend/node_modules/

# Environment
config.yaml

# Logs / Uploads
logs/
uploads/

# IDE
.vscode/
.idea/
*.swp
.DS_Store
GITIGNORE
    echo "  -> .gitignore"
fi

# ===============================
# 前端骨架
# ===============================
FRONTEND_DIR="frontend"
mkdir -p "$FRONTEND_DIR/src/pages"
mkdir -p "$FRONTEND_DIR/public"

# —— package.json ——
cat > "$FRONTEND_DIR/package.json" << PKGJSON
{
  "name": "${PROJECT_NAME}-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "@radix-ui/react-avatar": "^1.0.4",
    "@radix-ui/react-dialog": "^1.0.5",
    "@radix-ui/react-dropdown-menu": "^2.0.6",
    "@refinedev/core": "^4.48.0",
    "@refinedev/react-router-v6": "^4.5.0",
    "@tiptap/extension-image": "^3.26.1",
    "@tiptap/extension-link": "^3.26.1",
    "@tiptap/react": "^3.26.1",
    "@tiptap/starter-kit": "^3.26.1",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.0",
    "flatpickr": "^4.6.13",
    "lucide-react": "^0.309.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.21.0",
    "tailwind-merge": "^2.2.0"
  },
  "devDependencies": {
    "@types/flatpickr": "^3.0.2",
    "@types/react": "^18.2.43",
    "@types/react-dom": "^18.2.17",
    "@vitejs/plugin-react": "^6.0.2",
    "autoprefixer": "^10.4.16",
    "postcss": "^8.4.32",
    "tailwindcss": "^3.4.0",
    "typescript": "^5.3.3",
    "vite": "^8.0.16"
  }
}
PKGJSON
echo "  -> frontend/package.json"

# —— vite.config.ts ——
cat > "$FRONTEND_DIR/vite.config.ts" << VITECFG
import { defineConfig, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

function resolveGoNxAdminModules(): Plugin {
  return {
    name: 'resolve-go-nx-admin-modules',
    enforce: 'pre',
    async resolveId(id, importer, _options) {
      if (!importer || !importer.includes('/go-nx-admin/frontend/src/')) return null
      if (id.startsWith('.') || id.startsWith('/') || id.startsWith('\0')) return null
      const resolved = await this.resolve(id, path.resolve(__dirname, 'src/main.tsx'), { skipSelf: true })
      return resolved ?? null
    },
  }
}

export default defineConfig({
  plugins: [react(), resolveGoNxAdminModules()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined
          if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/react-router-dom/')) {
            return 'vendor-react'
          }
          if (id.includes('/@refinedev/')) {
            return 'vendor-refine'
          }
          if (id.includes('/@radix-ui/')) {
            return 'vendor-radix'
          }
          if (id.includes('/lucide-react/')) {
            return 'vendor-icons'
          }
          return undefined
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      'go-nx-admin-frontend': path.resolve(__dirname, '../go-nx-admin/frontend/src'),
    },
  },
  server: {
    port: 5173,
    host: process.env.VITE_HOST || '172.18.0.1',
    proxy: {
      '/api': {
        target: 'http://172.18.0.1:19500',
        changeOrigin: true,
      },
    },
  },
})
VITECFG
echo "  -> frontend/vite.config.ts"

# —— tsconfig.json ——
cat > "$FRONTEND_DIR/tsconfig.json" << TSCFG
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": false,
    "noUnusedParameters": false,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"],
      "go-nx-admin-frontend/*": ["../go-nx-admin/frontend/src/*"]
    }
  },
  "include": ["src", "../go-nx-admin/frontend/src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
TSCFG
echo "  -> frontend/tsconfig.json"

# —— tsconfig.node.json ——
cat > "$FRONTEND_DIR/tsconfig.node.json" << TSNODECFG
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
TSNODECFG
echo "  -> frontend/tsconfig.node.json"

# —— postcss.config.js ——
cat > "$FRONTEND_DIR/postcss.config.js" << PCFG
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
PCFG
echo "  -> frontend/postcss.config.js"

# —— tailwind.config.js ——
cat > "$FRONTEND_DIR/tailwind.config.js" << TWCFG
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
    '../go-nx-admin/frontend/src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
TWCFG
echo "  -> frontend/tailwind.config.js"

# —— index.html ——
cat > "$FRONTEND_DIR/index.html" << HTML
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>${PROJECT_NAME}</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
HTML
echo "  -> frontend/index.html"

# —— src/main.tsx ——
cat > "$FRONTEND_DIR/src/main.tsx" << MAINX
import React from 'react'
import ReactDOM from 'react-dom/client'
import { createApp } from 'go-nx-admin-frontend/core'
import './index.css'
import 'flatpickr/dist/flatpickr.min.css'

const App = createApp({
  // 二开注入点：添加业务页面后取消注释
  // extraResources: [
  //   { name: 'orders', list: '/orders' },
  // ],
  // extraRoutes: [
  //   // <Route key="orders" path="/orders" element={<OrderListPage />} />,
  // ],
  // extraRouteLabels: {
  //   '/orders': '订单管理',
  // },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
MAINX
echo "  -> frontend/src/main.tsx"

# —— src/index.css ——
cat > "$FRONTEND_DIR/src/index.css" << CSS
@tailwind base;
@tailwind components;
@tailwind utilities;
CSS
echo "  -> frontend/src/index.css"

# —— src/vite-env.d.ts ——
cat > "$FRONTEND_DIR/src/vite-env.d.ts" << VITEENV
/// <reference types="vite/client" />
VITEENV
echo "  -> frontend/src/vite-env.d.ts"

# —— .gitkeep ——
touch "$FRONTEND_DIR/src/pages/.gitkeep"
touch "$FRONTEND_DIR/dist/.gitkeep" 2>/dev/null || true
mkdir -p "$FRONTEND_DIR/dist"
touch "$FRONTEND_DIR/dist/.gitkeep"

echo ""
echo "======================================"
echo "骨架文件已生成到当前目录"
echo ""
echo "根目录结构:"
echo "  go-nx-admin/     # git submodule（底座，只读）"
echo "  cmd/main.go      # ← 入口（注入你的路由和模型）"
echo "  internal/        # ← 写你的业务代码"
echo "  frontend/        # ← 扩展前端页面"
echo "  go.mod           # replace go-nx-admin => ./go-nx-admin/backend"
echo "  Makefile"
echo "  config.example.yaml"
echo ""
echo "快速开始:"
echo "  cp config.example.yaml config.yaml"
echo "  cd frontend && npm install  # 安装前端依赖"
echo "  make build-frontend         # 构建前端"
echo "  ./${PROJECT_NAME} serve     # 启动后端"
echo ""
echo "种子账号: admin / admin123"
echo ""
echo "升级底座: make upgrade"
echo "======================================"
