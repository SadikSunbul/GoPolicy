package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"gopolicy/internal/handlers"
	"gopolicy/internal/policy"
)

//go:embed web/static/*
var staticFiles embed.FS

func main() {
	fmt.Println("Policy Plus - Go Edition")
	fmt.Println("Local Group Policy Editor for all Windows editions")
	fmt.Println("========================================")

	// Create main workspace
	workspace := policy.NewAdmxBundle()

	// Load default ADMX folder
	admxPath := os.Getenv("SystemRoot")
	if admxPath == "" {
		admxPath = "C:\\Windows"
	}
	admxPath += "\\PolicyDefinitions"

	fmt.Printf("Loading ADMX files: %s\n", admxPath)
	if _, err := os.Stat(admxPath); err == nil {
		languageCode := "tr-TR" //getSystemLanguage()
		fmt.Printf("Using language: %s\n", languageCode)
		failures, err := workspace.LoadFolder(admxPath, languageCode)
		if err != nil {
			log.Printf("ADMX loading error: %v\n", err)
		}
		if len(failures) > 0 {
			log.Printf("%d files failed to load\n", len(failures))
		}
	}

	// HTTP handlers
	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "web/static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// API endpoints
	handler := handlers.NewPolicyHandler(workspace)
	mux.HandleFunc("/", handler.HandleIndex)
	mux.HandleFunc("/api/categories", handler.HandleCategories)
	mux.HandleFunc("/api/policies", handler.HandlePolicies)
	mux.HandleFunc("/api/policy/", handler.HandlePolicy)
	mux.HandleFunc("/api/policy/set", handler.HandleSetPolicy)
	mux.HandleFunc("/api/sources", handler.HandleSources)
	mux.HandleFunc("/api/save", handler.HandleSave)

	port := ":8080"
	fmt.Printf("\nStarting web interface: http://localhost%s\n", port)
	fmt.Println("Open in your browser and start using it!")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}

// getSystemLanguage gets system language from environment variables
func getSystemLanguage() string {
	// Try LANG environment variable first
	if lang := os.Getenv("LANG"); lang != "" {
		// Convert from format like "en_US.UTF-8" to "en-US"
		lang = strings.ReplaceAll(lang, "_", "-")
		if idx := strings.Index(lang, "."); idx != -1 {
			lang = lang[:idx]
		}
		if lang != "" {
			return lang
		}
	}

	// Try LC_ALL
	if lang := os.Getenv("LC_ALL"); lang != "" {
		lang = strings.ReplaceAll(lang, "_", "-")
		if idx := strings.Index(lang, "."); idx != -1 {
			lang = lang[:idx]
		}
		if lang != "" {
			return lang
		}
	}

	// Try LC_MESSAGES
	if lang := os.Getenv("LC_MESSAGES"); lang != "" {
		lang = strings.ReplaceAll(lang, "_", "-")
		if idx := strings.Index(lang, "."); idx != -1 {
			lang = lang[:idx]
		}
		if lang != "" {
			return lang
		}
	}

	// Default fallback to English
	return "en-US"
}
