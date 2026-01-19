// Package web provides embedded frontend assets for Yggstack-GUI.
//
// The frontend is built with Vue.js 3 and Vite, and compiled assets
// are embedded into the Go binary at build time using go:embed.
//
// Usage:
//
//	import "github.com/JB-SelfCompany/yggstack-gui/internal/web"
//
//	// Get embedded filesystem
//	fs := web.Assets
//
//	// Serve static files
//	http.Handle("/", http.FileServer(http.FS(fs)))
package web

import "embed"

// Assets contains the embedded frontend files from the dist/ directory.
// This includes all compiled JavaScript, CSS, HTML, and static assets.
//
// The dist/ directory is populated by running:
//
//	cd frontend && npm run build
//	cp -r frontend/dist internal/web/dist
//
// Or by using the build.sh script:
//
//	bash build.sh
//
//go:embed dist/*
var Assets embed.FS
