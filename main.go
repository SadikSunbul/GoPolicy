package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"gopolicy/internal/handlers"
	"gopolicy/internal/policy"
)

//go:embed web/static/*
var staticFiles embed.FS

//go:embed docs/images/gopolicylogo.png
var logoFile []byte

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
		locales := detectLocales()
		fmt.Printf("Detected locales: %v\n", locales)
		failures, err := workspace.LoadFolder(admxPath, locales...)
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

	// Logo file
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(logoFile)
	})

	// API endpoints
	handler, err := handlers.NewPolicyHandler(workspace)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}
	mux.HandleFunc("/", handler.HandleIndex)
	mux.HandleFunc("/api/categories", handler.HandleCategories)
	mux.HandleFunc("/api/policies", handler.HandlePolicies)
	mux.HandleFunc("/api/policy/", handler.HandlePolicy)
	mux.HandleFunc("/api/policy/set", handler.HandleSetPolicy)
	mux.HandleFunc("/api/sources", handler.HandleSources)
	mux.HandleFunc("/api/save", handler.HandleSave)
	mux.HandleFunc("/api/search", handler.HandleSearch)
	mux.HandleFunc("/api/refresh-explorer", handler.HandleRefreshExplorer)

	// Parse command line flags
	portFlag := flag.Int("p", 8080, "Port number to run the server on")
	flag.Parse()

	port := fmt.Sprintf(":%d", *portFlag)
	fmt.Printf("\nStarting web interface: http://localhost%s\n", port)
	fmt.Println("Open in your browser and start using it!")
	fmt.Printf("Press Ctrl+C to stop the server\n\n")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}

func detectLocales() []string {
	localeSet := map[string]struct{}{}
	addLocale := func(loc string) {
		norm := normalizeLocale(loc)
		if norm != "" {
			localeSet[norm] = struct{}{}
		}
	}

	addLocale(os.Getenv("LANG"))
	addLocale(os.Getenv("PreferredLanguage"))
	addLocale(os.Getenv("UILanguage"))

	// backup locales (Turkish and English)
	addLocale("tr-TR")
	addLocale("en-US")

	var locales []string
	for loc := range localeSet {
		locales = append(locales, loc)
	}
	sort.Strings(locales)
	return locales
}

func normalizeLocale(loc string) string {
	loc = strings.TrimSpace(loc)
	if len(loc) < 4 {
		return ""
	}
	loc = strings.ReplaceAll(loc, "_", "-")
	parts := strings.Split(loc, "-")
	if len(parts) < 2 {
		return ""
	}
	lang := strings.ToLower(parts[0])
	region := strings.ToUpper(parts[1])
	return lang + "-" + region
}
