// Command spotify-auth performs a one-time Spotify OAuth authorization to obtain
// a long-lived refresh token for the "now playing" badge.
//
// Prerequisite: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set (in .env
// or the shell), and the app's Redirect URI in the Spotify dashboard must be
// EXACTLY http://127.0.0.1:8888/callback.
//
// Usage: make spotify-auth   (or: go run ./cmd/spotify-auth)
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"dconn.dev/internal/config"
)

const (
	redirectURI = "http://127.0.0.1:8888/callback"
	scopes      = "user-read-currently-playing user-read-recently-played"
	authBase    = "https://accounts.spotify.com/authorize"
	tokenURL    = "https://accounts.spotify.com/api/token"
)

func main() {
	config.LoadDotEnv(".env")

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		fmt.Println("ERROR: set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET (in .env or your shell) first.")
		os.Exit(1)
	}

	authURL := authBase + "?" + url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"redirect_uri":  {redirectURI},
		"scope":         {scopes},
	}.Encode()

	fmt.Println()
	fmt.Println("1. Open this URL in a browser and approve access:")
	fmt.Println()
	fmt.Println("   " + authURL)
	fmt.Println()
	fmt.Println("2. Spotify will redirect to " + redirectURI + "?code=...")
	fmt.Println("   - Browser on THIS machine? The code is caught automatically.")
	fmt.Println("   - Browser elsewhere (e.g. headless server)? The page will fail to")
	fmt.Println("     load — just copy the full URL from the address bar and paste it below.")
	fmt.Println()

	codeCh := make(chan string, 1)

	// Auto-catcher: works when the browser runs on this machine.
	srv := &http.Server{Addr: "127.0.0.1:8888"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "no code in callback (error="+r.URL.Query().Get("error")+")", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "Got it! You can close this tab and return to the terminal.")
		trySend(codeCh, code)
	})
	go srv.ListenAndServe()

	// Manual fallback: works when the browser is on a different machine.
	go func() {
		fmt.Print("Paste the code or redirect URL here (or just wait for auto-catch): ")
		var line string
		fmt.Scanln(&line)
		if c := extractCode(line); c != "" {
			trySend(codeCh, c)
		}
	}()

	code := <-codeCh
	_ = srv.Shutdown(context.Background())

	refreshToken, err := exchange(clientID, clientSecret, code)
	if err != nil {
		fmt.Println("\nERROR exchanging code:", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("SUCCESS. Add this line to your .env (keep it secret, never commit it):")
	fmt.Println()
	fmt.Println("   SPOTIFY_REFRESH_TOKEN=" + refreshToken)
	fmt.Println()
}

func trySend(ch chan<- string, v string) {
	select {
	case ch <- v:
	default:
	}
}

// extractCode pulls the authorization code out of either a bare code, a full
// redirect URL, or a "...?code=XXX&..." fragment.
func extractCode(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "code=") {
		if u, err := url.Parse(s); err == nil {
			if c := u.Query().Get("code"); c != "" {
				return c
			}
		}
		rest := s[strings.Index(s, "code=")+len("code="):]
		if amp := strings.IndexByte(rest, '&'); amp >= 0 {
			rest = rest[:amp]
		}
		return rest
	}
	return s // assume a bare code was pasted
}

// exchange trades the authorization code for a refresh token.
func exchange(clientID, clientSecret, code string) (string, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body struct {
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK || body.RefreshToken == "" {
		return "", fmt.Errorf("status %d: %s %s", resp.StatusCode, body.Error, body.ErrorDesc)
	}
	return body.RefreshToken, nil
}
