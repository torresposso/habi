package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/torresposso/habi/internal/pocketbase"
	"github.com/torresposso/habi/views"
)

func init() {
	log.SetOutput(os.Stdout)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("%s %s %s", r.Method, r.URL.Path, duration)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered: %v", r)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func getOAuthRedirectURL() string {
	if url := os.Getenv("OAUTH_REDIRECT_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func main() {
	pbURL := os.Getenv("PB_URL")
	if pbURL == "" {
		pbURL = "http://127.0.0.1:8090"
	}

	if err := pocketbase.TestConnection(pbURL); err != nil {
		log.Printf("Warning: Could not connect to PocketBase at %s: %v", pbURL, err)
	}

	// Handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		isLoggedIn := false
		if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
			isLoggedIn = true
		}

		recoveryMiddleware(templ.Handler(views.LandingPage(isLoggedIn))).ServeHTTP(w, r)
	})

	http.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
			http.Redirect(w, r, "/me", http.StatusSeeOther)
			return
		}
		recoveryMiddleware(templ.Handler(views.LoginPage())).ServeHTTP(w, r)
	})

	http.HandleFunc("/auth/google", func(w http.ResponseWriter, r *http.Request) {
		methods, err := pocketbase.GetAuthMethods(pbURL)
		if err != nil || len(methods) == 0 {
			http.Error(w, "Auth methods not available", http.StatusInternalServerError)
			return
		}

		// Buscar Google provider
		var google pocketbase.AuthMethod
		for _, m := range methods {
			if m.Name == "google" {
				google = m
				break
			}
		}

		if google.Name == "" {
			http.Error(w, "Google auth not configured", http.StatusInternalServerError)
			return
		}

		// Guardar state y verifier en cookies para el callback
		http.SetCookie(w, &http.Cookie{Name: "auth_state", Value: google.State, Path: "/", HttpOnly: true})
		http.SetCookie(w, &http.Cookie{Name: "auth_verifier", Value: google.CodeVerifier, Path: "/", HttpOnly: true})

		redirectURL := google.AuthURL + "&redirect_uri=" + getOAuthRedirectURL() + "/auth/callback"
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		stateCookie, err := r.Cookie("auth_state")
		if err != nil || stateCookie.Value != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		verifierCookie, _ := r.Cookie("auth_verifier")

		auth, err := pocketbase.AuthWithOAuth2(pbURL, "google", code, verifierCookie.Value, getOAuthRedirectURL()+"/auth/callback")
		if err != nil {
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Guardar token en cookie
		authData, _ := json.Marshal(auth)
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    string(authData),
			Path:     "/",
			HttpOnly: true,
			MaxAge:   3600 * 24,
		})

		http.Redirect(w, r, "/me", http.StatusSeeOther)
	})

	http.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		var auth pocketbase.AuthResponse
		if err := json.Unmarshal([]byte(cookie.Value), &auth); err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		recoveryMiddleware(templ.Handler(views.UserProfile(&auth))).ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
