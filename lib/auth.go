package zbz

import (
	"context"
	"errors"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Auth is an interface that defines methods for user authentication.
type Auth interface {
	VerifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error)
	TokenMiddleware(ctx *gin.Context)
	LoginHandler(ctx *gin.Context)
	CallbackHandler(ctx *gin.Context)
	LogoutHandler(ctx *gin.Context)
}

// Auth is used to authenticate our users.
type ZbzAuth struct {
	config   Config
	log      Logger
	oauth    oauth2.Config
	provider *oidc.Provider
}

// NewAuth instantiates the Auth module
func NewAuth(l Logger, config Config) Auth {
	provider, err := oidc.NewProvider(
		context.Background(),
		"https://"+config.AuthDomain()+"/",
	)
	if err != nil {
		l.Fatal("Failed to create OIDC provider:", err)
	}

	oauth := oauth2.Config{
		ClientID:     config.AuthClientID(),
		ClientSecret: config.AuthClientSecret(),
		RedirectURL:  config.AuthCallback(),
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	return &ZbzAuth{
		provider: provider,
		oauth:    oauth,
		config:   config,
	}
}

// VerifyIDToken verifies that an *oauth2.Token is a valid *oidc.IDToken.
func (a *ZbzAuth) VerifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	oidcConfig := &oidc.Config{
		ClientID: a.oauth.ClientID,
	}

	return a.provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
}

// TokenMiddleware is a middleware that checks if the user is authenticated.
func (a *ZbzAuth) TokenMiddleware(ctx *gin.Context) {
	if sessions.Default(ctx).Get("profile") == nil {
		ctx.Redirect(http.StatusSeeOther, "/")
	} else {
		ctx.Next()
	}
}

// LoginHandler initiates the OAuth2 authorization code flow by redirecting the user to the Auth0 authorization server
func (a *ZbzAuth) LoginHandler(ctx *gin.Context) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	state := base64.StdEncoding.EncodeToString(b)

	session := sessions.Default(ctx)
	session.Set("state", state)
	if err := session.Save(); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, a.oauth.AuthCodeURL(state))
}

// CallbackHandler handles the OAuth2 callback from the Auth0 authorization server
func (a *ZbzAuth) CallbackHandler(ctx *gin.Context) {
	session := sessions.Default(ctx)
	if ctx.Query("state") != session.Get("state") {
		ctx.String(http.StatusBadRequest, "Invalid state parameter.")
		return
	}

	token, err := a.oauth.Exchange(ctx.Request.Context(), ctx.Query("code"))
	if err != nil {
		ctx.String(http.StatusUnauthorized, "Failed to exchange an authorization code for a token.")
		return
	}

	idToken, err := a.VerifyIDToken(ctx.Request.Context(), token)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "Failed to verify ID Token.")
		return
	}

	var profile map[string]any
	if err := idToken.Claims(&profile); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	session.Set("access_token", token.AccessToken)
	session.Set("profile", profile)
	if err := session.Save(); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, "/user")
}

// LogoutHandler handles the logout process by redirecting the user to the Auth0 logout endpoint
func (a *ZbzAuth) LogoutHandler(ctx *gin.Context) {
	logoutUrl, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/v2/logout")
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}

	returnTo, err := url.Parse(scheme + "://" + ctx.Request.Host)
	if err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	parameters := url.Values{}
	parameters.Add("returnTo", returnTo.String())
	parameters.Add("client_id", os.Getenv("AUTH0_CLIENT_ID"))
	logoutUrl.RawQuery = parameters.Encode()

	ctx.Redirect(http.StatusTemporaryRedirect, logoutUrl.String())
}
