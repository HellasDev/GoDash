package calendar

import (
	"context"
	"encoding/base64"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"GoDash/internal/config"
)

//go:embed credentials.json
var credentialsFile []byte

//go:embed 1761.png
var logoImage []byte

var (
	ErrAuthRequired = fmt.Errorf("authentication required")
)

// GetCalendarService creates a new Google Calendar service client.
// It handles the OAuth 2.0 flow.
func GetCalendarService() (*calendar.Service, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}

	client, err := getClient(cfg)
	if err != nil {
		return nil, err
	}

	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %v", err)
	}

	return srv, nil
}

// getClient retrieves a token, saves the token, then returns the generated client.
func getClient(cfg *oauth2.Config) (*http.Client, error) {
	tokFile, err := getTokenPath()
	if err != nil {
		return nil, fmt.Errorf("unable to get token path: %v", err)
	}

	tok, err := tokenFromFile(tokFile)
	if err != nil {
		return nil, ErrAuthRequired
	}
	return cfg.Client(context.Background(), tok), nil
}

// StartAuthFlow starts the OAuth flow, trying automatic server first, then falling back to manual.
func StartAuthFlow() (string, error) {
	// Try to start local server for automatic flow
	go tryStartCallbackServer()
	
	// Wait for server to start (or fail)
	select {
	case <-serverStarted:
		// Server started successfully
		useManualFlow = false
	case <-time.After(2 * time.Second):
		// Server failed to start, use manual flow
		useManualFlow = true
	}
	
	cfg, err := getConfig()
	if err != nil {
		return "", err
	}

	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	return authURL, nil
}

// IsUsingManualFlow returns true if we're using manual code flow
func IsUsingManualFlow() bool {
	return useManualFlow
}

// GetAuthURL returns the URL the user needs to visit to authorize the application.
// This is kept for backward compatibility but now uses the new flow.
func GetAuthURL() (string, error) {
	return StartAuthFlow()
}

// CompleteAuth exchanges an authorization code for a token and saves it.
func CompleteAuth(authCode string) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	tok, err := cfg.Exchange(context.TODO(), authCode)
	if err != nil {
		return fmt.Errorf("unable to retrieve token from web: %v", err)
	}

	return saveToken(tok)
}

var authComplete = make(chan *oauth2.Token, 1)
var authError = make(chan error, 1)

// tryStartCallbackServer tries to start a callback server on available ports
func tryStartCallbackServer() {
	var listener net.Listener
	var err error
	
	// Try each port in our range
	for _, port := range portRange {
		listener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			serverPort = port
			break
		}
	}
	
	// If no port was available, signal failure
	if listener == nil {
		return
	}
	
	defer listener.Close()
	
	// Notify that server is ready
	serverStarted <- true

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", handleOAuthCallback)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/callback"+r.URL.RawQuery, http.StatusTemporaryRedirect)
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Callback server error: %v", err)
		}
	}()

	// Wait for auth completion or timeout
	select {
	case <-authComplete:
		server.Shutdown(context.Background())
	case <-authError:
		server.Shutdown(context.Background())
	case <-time.After(5 * time.Minute):
		log.Println("OAuth timeout")
		server.Shutdown(context.Background())
		authError <- fmt.Errorf("authentication timeout")
	}
}

// handleOAuthCallback handles the OAuth callback from Google
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		errorMsg := r.URL.Query().Get("error")
		if errorMsg != "" {
			http.Error(w, fmt.Sprintf("OAuth error: %s", errorMsg), http.StatusBadRequest)
			authError <- fmt.Errorf("oauth error: %s", errorMsg)
			return
		}
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		authError <- fmt.Errorf("no authorization code received")
		return
	}

	// Exchange code for token
	cfg, err := getConfig()
	if err != nil {
		http.Error(w, "Configuration error", http.StatusInternalServerError)
		authError <- err
		return
	}

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		authError <- err
		return
	}

	// Save token
	if err := saveToken(tok); err != nil {
		http.Error(w, "Failed to save token", http.StatusInternalServerError)
		authError <- err
		return
	}

	// Encode the logo as base64 for embedding
	logoBase64 := base64.StdEncoding.EncodeToString(logoImage)
	
	// Success page with One Dark theme
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>GoDash - Authentication Successful</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'JetBrains Mono', monospace, Roboto, sans-serif; 
            text-align: center; 
            padding: 50px; 
            background: #282c34;
            color: #abb2bf;
            margin: 0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container { 
            background: #21252b; 
            padding: 50px 40px; 
            border-radius: 12px; 
            box-shadow: 0 10px 30px rgba(0,0,0,0.4);
            max-width: 500px;
            border: 1px solid #3e4451;
            animation: slideUp 0.5s ease-out;
        }
        @keyframes slideUp {
            from { opacity: 0; transform: translateY(30px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .title { 
            color: #98c379; 
            font-size: 32px; 
            margin-bottom: 30px;
            font-weight: 600;
        }
        .logo {
            margin: 20px 0;
        }
        .logo img {
            max-width: 200px;
            height: auto;
        }
        .message { 
            color: #abb2bf; 
            font-size: 18px;
            line-height: 1.6;
            margin-bottom: 20px;
        }
        .app-name { 
            color: #61afef; 
            font-weight: 600; 
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="title">Authentication Successful!</div>
        <div class="logo">
            <img src="data:image/png;base64,%s" alt="GoDash Logo" />
        </div>
        <div class="message">
            You can now close this browser window.<br><br>
            <span class="app-name">GoDash Application</span> has been authorized to access your Google Calendar.
        </div>
    </div>
</body>
</html>
`, logoBase64)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))

	authComplete <- tok
}

// WaitForAuth waits for the OAuth flow to complete and returns any error
func WaitForAuth() error {
	select {
	case <-authComplete:
		return nil
	case err := <-authError:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authentication timeout")
	}
}

// IsAuthorized checks if the user has a valid token.
func IsAuthorized() bool {
	tokFile, err := getTokenPath()
	if err != nil {
		return false
	}
	_, err = tokenFromFile(tokFile)
	return err == nil
}

var (
	serverPort int
	serverStarted = make(chan bool, 1)
	useManualFlow bool
	// Port range to try for OAuth callback server
	portRange = []int{8080, 8081, 8082, 8083, 8084, 8085, 8086, 8087, 8088, 8089, 8090}
)

// getConfig loads the OAuth2 config from the embedded credentials file.
func getConfig() (*oauth2.Config, error) {
	// For security, the credentials.json file should be provided by the user
	// and embedded into the application at compile time.
	// We are providing a placeholder file for now.
	config, err := google.ConfigFromJSON(credentialsFile, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, err
	}
	
	// Set redirect URL based on flow type
	if useManualFlow {
		// Use out-of-band for manual flow (though deprecated, still works)
		config.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"
	} else {
		// Use localhost with assigned port for automatic flow
		config.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", serverPort)
	}
	return config, nil
}

// tokenFromFile retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path.
func saveToken(token *oauth2.Token) error {
	path, err := getTokenPath()
	if err != nil {
		return fmt.Errorf("unable to get token path: %v", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// getTokenPath returns the path to the token.json file.
func getTokenPath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "token.json"), nil
}

// GetCalendarEvents fetches events for a specific day from the user's primary calendar.
func GetCalendarEvents(srv *calendar.Service, day time.Time) ([]*calendar.Event, error) {
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	events, err := srv.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(startOfDay.Format(time.RFC3339)).
		TimeMax(endOfDay.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events for the selected day: %v", err)
	}
	return events.Items, nil
}

// --- Caching Functions ---

func getCalendarCachePath() (string, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "calendar_cache.json"), nil
}

// LoadCalendarCache reads the event cache from disk.
func LoadCalendarCache() (map[string][]*calendar.Event, error) {
	path, err := getCalendarCachePath()
	if err != nil {
		return nil, fmt.Errorf("unable to get calendar cache path: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist, return an empty cache
			return make(map[string][]*calendar.Event), nil
		}
		return nil, fmt.Errorf("could not read calendar cache file: %v", err)
	}

	var cache map[string][]*calendar.Event
	err = json.Unmarshal(content, &cache)
	if err != nil {
		// If unmarshalling fails, maybe the file is corrupt. Return an empty cache and log the error.
		fmt.Printf("Warning: could not unmarshal calendar cache, starting fresh: %v\n", err)
		return make(map[string][]*calendar.Event), nil
	}

	return cache, nil
}

// SaveCalendarCache writes the event cache to disk.
func SaveCalendarCache(cache map[string][]*calendar.Event) error {
	path, err := getCalendarCachePath()
	if err != nil {
		return fmt.Errorf("unable to get calendar cache path: %v", err)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal calendar cache: %v", err)
	}

	return os.WriteFile(path, data, 0644)
}

// GetCalendarEventsForMonth fetches events for a specific month from the user's primary calendar.
func GetCalendarEventsForMonth(srv *calendar.Service, month time.Time) ([]*calendar.Event, error) {
	firstDayOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	firstDayOfNextMonth := firstDayOfMonth.AddDate(0, 1, 0)

	events, err := srv.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(firstDayOfMonth.Format(time.RFC3339)).
		TimeMax(firstDayOfNextMonth.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events for the selected month: %v", err)
	}
	return events.Items, nil
}
