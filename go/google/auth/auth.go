package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/auth/credentials/impersonate"
	"github.com/duizendstra/alexandria/go/retry/gcp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	envImpersonateServiceAccount = "GOOGLE_IMPERSONATE_SERVICE_ACCOUNT"
	envOAuthClient               = "GOOGLE_OAUTH_CLIENT"
	defaultServerTimeout         = 3 * time.Second
)

//nolint:gochecknoglobals // Stubbable for testing.
var execCommand = exec.CommandContext

var (
	// ErrNilValidator is returned when the DWD validator or its underlying service is nil.
	ErrNilValidator = errors.New("validator or service is nil")

	// ErrUnsafeURL is returned when trying to open a URL that does not use a secure scheme (https://).
	ErrUnsafeURL = errors.New("refusing to open URL: unsafe scheme or protocol")

	// ErrInvalidPassKey is returned when a pass store key starts with a hyphen to prevent argument injection.
	ErrInvalidPassKey = errors.New("invalid pass key: cannot start with a hyphen")
	// ErrNoImpersonationAccount is returned when the required impersonation service account
	// email address is not supplied and the fallback environment variable
	// GOOGLE_IMPERSONATE_SERVICE_ACCOUNT is also empty.
	ErrNoImpersonationAccount = errors.New("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT environment variable is not set")

	// ErrNoSubjectEmail is returned when the subjectEmail argument is empty, which is a required
	// parameter for Google Domain-Wide Delegation (DWD) impersonation.
	ErrNoSubjectEmail = errors.New("subjectEmail must not be empty for Domain-Wide Delegation")

	// ErrInvalidServiceAccount is returned when the service account email address does not follow
	// a standard email address formatting.
	ErrInvalidServiceAccount = errors.New("invalid service account email: must follow a standard email address formatting")

	// ErrInvalidSubjectEmail is returned when the subject email address being impersonated does
	// not resemble a correct, valid email pattern.
	ErrInvalidSubjectEmail = errors.New("invalid subject email: must resemble a correct email pattern")

	// ErrNoAuthenticationMode is returned when no valid authentication options are provided to the builder.
	ErrNoAuthenticationMode = errors.New("no Google authentication mode was configured")
)

// defaultTokenPath is where the cached OAuth2 token is stored.
const defaultTokenPath = ".kratos/tokens/google-oauth.json" //nolint:gosec // Path is not a hardcoded secret.

// defaultPassKey is the pass store key for the OAuth2 client credentials.
const defaultPassKey = "dui/google-oauth-client" //nolint:gosec // Key is not a hardcoded secret.

// Option represents a functional configuration for Google client factories.
type Option func(*config)

type config struct {
	targetSA      string
	subjectEmail  string
	isDWD         bool
	passKey       string
	tokenPath     string
	scopes        []string
	logger        *slog.Logger
	httpClient    *http.Client
	isInteractive bool
}

// WithServiceAccountImpersonation configures direct SA-to-SA impersonation.
func WithServiceAccountImpersonation(targetSA string) Option {
	return func(c *config) {
		c.targetSA = targetSA
	}
}

// WithDomainWideDelegation configures Domain-Wide Delegation (DWD) impersonation of a target user.
func WithDomainWideDelegation(targetSA, subjectEmail string) Option {
	return func(c *config) {
		c.targetSA = targetSA
		c.subjectEmail = subjectEmail
		c.isDWD = true
	}
}

// WithInteractiveConsent configures the interactive desktop OAuth2 flow with token caching.
func WithInteractiveConsent(passKey, tokenPath string) Option {
	return func(c *config) {
		c.passKey = passKey
		c.tokenPath = tokenPath
		c.isInteractive = true
	}
}

// WithScopes sets the requested Google API scopes.
func WithScopes(scopes ...string) Option {
	return func(c *config) {
		c.scopes = scopes
	}
}

// WithLogger configures a structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithHTTPClient injects a pre-authenticated custom client (perfect for mock testing).
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.httpClient = client
	}
}

// IsValidEmail checks if a string resembles a correct email pattern.
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}
	parts := strings.Split(email, "@")
	const expectedParts = 2
	if len(parts) != expectedParts {
		return false
	}
	local, domain := parts[0], parts[1]

	return local != "" && domain != "" && !strings.ContainsAny(email, " \t\n\r")
}

// IsValidServiceAccount checks if the email follows a standard email address formatting.
func IsValidServiceAccount(email string) bool {
	return IsValidEmail(email)
}

// ResolveClient builds and returns the option-based client or credentials configuration.
func ResolveClient(ctx context.Context, defaultScopes []string, opts ...Option) ([]option.ClientOption, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.httpClient != nil {
		return []option.ClientOption{option.WithHTTPClient(cfg.httpClient)}, nil
	}

	// Dynamic fallback to Environment service account if no mode is selected.
	if cfg.targetSA == "" && !cfg.isInteractive {
		cfg.targetSA = os.Getenv(envImpersonateServiceAccount)
	}

	if cfg.isDWD && cfg.targetSA == "" {
		return nil, ErrNoImpersonationAccount
	}

	// Apply default scopes if none were explicitly configured.
	if len(cfg.scopes) == 0 {
		cfg.scopes = defaultScopes
	}

	if cfg.targetSA != "" {
		return resolveImpersonationClient(cfg)
	}

	if cfg.isInteractive {
		return resolveInteractiveClient(ctx, cfg)
	}

	// Fall back to Google Application Default Credentials (ADC).
	return nil, nil
}

// resolveImpersonationClient resolves direct or DWD impersonated service credentials.
func resolveImpersonationClient(cfg *config) ([]option.ClientOption, error) {
	if !IsValidServiceAccount(cfg.targetSA) {
		return nil, ErrInvalidServiceAccount
	}

	if cfg.isDWD && cfg.subjectEmail == "" {
		return nil, ErrNoSubjectEmail
	}

	var subject string
	if cfg.subjectEmail != "" {
		if !IsValidEmail(cfg.subjectEmail) {
			return nil, ErrInvalidSubjectEmail
		}
		subject = cfg.subjectEmail
	}

	creds, err := impersonate.NewCredentials(&impersonate.CredentialsOptions{
		TargetPrincipal: cfg.targetSA,
		Scopes:          cfg.scopes,
		Subject:         subject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonated credentials: %w", err)
	}

	return []option.ClientOption{option.WithAuthCredentials(creds)}, nil
}

// resolveInteractiveClient resolves a consent-based interactive client.
func resolveInteractiveClient(ctx context.Context, cfg *config) ([]option.ClientOption, error) {
	passKey := cfg.passKey
	if passKey == "" {
		passKey = defaultPassKey
	}

	tokenPath := cfg.tokenPath
	if tokenPath == "" {
		home, _ := os.UserHomeDir()
		tokenPath = filepath.Join(home, defaultTokenPath)
	}

	client, err := interactiveConsentClient(ctx, passKey, tokenPath, cfg.scopes, cfg.logger)
	if err != nil {
		return nil, err
	}

	return []option.ClientOption{option.WithHTTPClient(client)}, nil
}

// DWDValidator provides a mechanism to verify that Domain-Wide Delegation (DWD)
// is authorized and active for a given subject email address.
type DWDValidator struct {
	service *drive.Service
}

// NewDWDValidator creates a new DWDValidator instance.
func NewDWDValidator(srv *drive.Service) *DWDValidator {
	return &DWDValidator{service: srv}
}

// ValidateAccess performs a basic root-level validation.
func (v *DWDValidator) ValidateAccess(ctx context.Context, userEmail string) error {
	if v == nil || v.service == nil {
		return fmt.Errorf("DWD validation failed for %s: %w", userEmail, ErrNilValidator)
	}

	err := gcp.WithRetry(ctx, func() error {
		_, innerErr := v.service.Files.Get("root").Fields("id").Context(ctx).Do()
		if innerErr != nil {
			return fmt.Errorf("files.get root: %w", innerErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("DWD validation failed for %s: %w", userEmail, err)
	}

	return nil
}

// Interactive Consent Flow Helpers.

var (
	errNoCode        = errors.New("no code in callback")
	errStateMismatch = errors.New("state token mismatch: security warning")
)

type callbackHandler struct {
	stateToken string
	codeCh     chan<- string
	errCh      chan<- error
}

func (h *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state != h.stateToken {
		http.Error(w, "invalid state parameter (CSRF detected)", http.StatusForbidden)
		select {
		case h.errCh <- errStateMismatch:
		default:
		}

		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		select {
		case h.errCh <- errNoCode:
		default:
		}

		return
	}

	_, _ = fmt.Fprint(w, "<html><body><h2>Authorization complete</h2><p>You can close this tab.</p></body></html>")
	select {
	case h.codeCh <- code:
	default:
	}
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(ctx context.Context, authURL string) error {
	if !strings.HasPrefix(authURL, "https://") {
		return ErrUnsafeURL
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = execCommand(ctx, "open", authURL)
	case "windows":
		cmd = execCommand(ctx, "rundll32", "url.dll,FileProtocolHandler", authURL)
	default:
		cmd = execCommand(ctx, "xdg-open", authURL)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start browser process: %w", err)
	}

	return nil
}

func interactiveConsentClient(ctx context.Context, passKey, tokenPath string, scopes []string, logger *slog.Logger) (*http.Client, error) {
	log := logger
	if log == nil {
		log = slog.Default()
	}

	credsJSON, err := resolveCredentials(ctx, passKey)
	if err != nil {
		return nil, fmt.Errorf("oauth: resolve credentials: %w", err)
	}

	oauthCfg, err := google.ConfigFromJSON(credsJSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("oauth: parse client credentials: %w", err)
	}

	tok, err := loadToken(tokenPath)
	if err != nil {
		log.Info("no cached token, starting consent flow", slog.String("reason", err.Error()))

		tok, err = consentFlow(ctx, oauthCfg, log)
		if err != nil {
			return nil, fmt.Errorf("oauth: consent flow: %w", err)
		}

		if err := saveToken(tokenPath, tok); err != nil {
			log.Warn("failed to cache token", slog.String("error", err.Error()))
		}
	}

	return oauthCfg.Client(ctx, tok), nil
}

func resolveCredentials(ctx context.Context, passKey string) ([]byte, error) {
	if v := os.Getenv(envOAuthClient); v != "" {
		return []byte(v), nil
	}

	if strings.HasPrefix(passKey, "-") {
		return nil, ErrInvalidPassKey
	}

	out, err := execCommand(ctx, "pass", "show", passKey).Output()
	if err != nil {
		return nil, fmt.Errorf("set GOOGLE_OAUTH_CLIENT or run: pass insert %s: %w", passKey, err)
	}

	return out, nil
}

func consentFlow(ctx context.Context, cfg *oauth2.Config, logger *slog.Logger) (*oauth2.Token, error) {
	const stateBytesLen = 16
	stateBytes := make([]byte, stateBytesLen)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("generate secure state token: %w", err)
	}
	stateToken := hex.EncodeToString(stateBytes)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	handler := &callbackHandler{
		stateToken: stateToken,
		codeCh:     codeCh,
		errCh:      errCh,
	}

	//nolint:noctx // CLI consent tool callback server, runs directly to completion.
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // TCP protocol strictly guarantees TCPAddr type.
	cfg.RedirectURL = fmt.Sprintf("http://localhost:%d", port)

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: defaultServerTimeout,
	}

	go func() {
		if sErr := srv.Serve(listener); sErr != nil && !errors.Is(sErr, http.ErrServerClosed) {
			select {
			case errCh <- sErr:
			default:
			}
		}
	}()

	//nolint:contextcheck // Teardown executes on fresh background context with custom timeout.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultServerTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	authURL := cfg.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)
	fmt.Printf("\nOpening browser for Google OAuth2 consent...\n")

	if oErr := openBrowser(ctx, authURL); oErr != nil {
		if logger != nil {
			logger.Warn("could not open browser automatically", slog.String("error", oErr.Error()))
		}
		fmt.Printf("Could not open browser. Visit this URL manually:\n\n  %s\n\n", authURL)
	}

	select {
	case code := <-codeCh:
		tok, tErr := cfg.Exchange(ctx, code)
		if tErr != nil {
			return nil, fmt.Errorf("exchange code for token: %w", tErr)
		}

		return tok, nil
	case err := <-errCh:
		return nil, fmt.Errorf("callback error: %w", err)
	case <-ctx.Done():
		return nil, fmt.Errorf("consent context cancelled: %w", ctx.Err())
	}
}

func loadToken(path string) (*oauth2.Token, error) {
	//nolint:gosec // Subprocess opens strictly cached user token paths.
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("loadToken: %w", err)
	}
	defer func() { _ = f.Close() }()

	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return tok, nil
}

func saveToken(path string, tok *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { //nolint:mnd // Standard secure permission.
		return fmt.Errorf("create token dir: %w", err)
	}

	//nolint:gosec // Safely writes token cache to isolated home subfolders.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:mnd // Standard private secure permissions.
	if err != nil {
		return fmt.Errorf("create token file: %w", err)
	}

	if err := json.NewEncoder(f).Encode(tok); err != nil { //nolint:gosec // Writing OAuth2 token cache is safe.
		_ = f.Close()

		return fmt.Errorf("encode token: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close token file: %w", err)
	}

	return nil
}
