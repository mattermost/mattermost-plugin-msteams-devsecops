// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mattermost/mattermost-plugin-msteams-devsecops/assets"
	"github.com/mattermost/mattermost-plugin-msteams-devsecops/server/store/pluginstore"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	pluginapi "github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/v8/channels/utils"
	"github.com/sirupsen/logrus"
)

type iFrameContext struct {
	SiteURL  string
	PluginID string
	TenantID string
	UserID   string
	Nonce    string

	Post                       *model.Post
	PostJSON                   string
	NotificationPreviewContext iFrameNotificationPreviewContext
}

type iFrameNotificationPreviewContext struct {
	PostAuthor *model.User
	Channel    *model.Channel

	ChannelNameDisplay   string
	PostAuthorDisplay    string
	PostCreatedAtDisplay string
	Action               string
}

// iFrame returns the iFrame HTML needed to host Mattermost within a MS Teams app.
func (a *API) iFrame(w http.ResponseWriter, r *http.Request) {
	a.p.API.LogDebug("iFrame", "action", r.URL.Query().Get("action"), "sub_entity_id", r.URL.Query().Get("sub_entity_id"))

	iframeCtx, err := a.createIFrameContext("", nil)
	if err != nil {
		a.p.API.LogError("Failed to create iFrame context", "error", err.Error())
		http.Error(w, "Failed to create iFrame context", http.StatusInternalServerError)
		return
	}

	html, err := a.formatTemplate(assets.IFrameHTMLTemplate, iframeCtx)
	if err != nil {
		a.p.API.LogError("Failed to format iFrame HTML", "error", err.Error())
		http.Error(w, "Failed to format iFrame HTML", http.StatusInternalServerError)
		return
	}

	// cspDirectives is a CSP for the wrapper page
	// default-src: Block all resources by default
	// style-src: Allow inline styles with nonce
	// script-src: Allow scripts from Microsoft Teams CDN and inline scripts with nonce
	// connect-src: Allow connections to Microsoft and Teams domains
	// frame-src: Allow frames from the same origin
	// report-to: Send CSP violation reports to our endpoint
	cspDirectives := []string{
		"default-src 'none'",
		"style-src 'nonce-" + iframeCtx.Nonce + "'",
		"script-src https://res.cdn.office.net 'nonce-" + iframeCtx.Nonce + "';",
		"connect-src https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net",
		"frame-src 'self'",
		"report-to csp-endpoint",
	}

	// Set the Report-To header to define the reporting endpoint group
	reportToJSON := `{"group":"csp-endpoint","max_age":10886400,"endpoints":[{"url":"/plugins/` + iframeCtx.PluginID + `/csp-report"}]}`
	w.Header().Set("Report-To", reportToJSON)

	w.Header().Set("Content-Security-Policy", strings.Join(cspDirectives, "; "))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Type", "text/html")

	// set session cookie to indicate Mattermost is hosted in an iFrame, which allows
	// webapp to bypass "Where do you want to view this" page and set SameSite=none.
	http.SetCookie(w, &http.Cookie{
		Name:     "MMEMBED",
		Value:    "1",
		Path:     "/",
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})

	if _, err := w.Write([]byte(html)); err != nil {
		a.p.API.LogWarn("Unable to serve the iFrame", "error", err.Error())
	}
}

func (a *API) iframeNotificationPreview(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "user not authenticated", http.StatusUnauthorized)
		return
	}
	postID := r.URL.Query().Get("post_id")
	if postID == "" {
		http.Error(w, "post_id is required", http.StatusBadRequest)
		return
	}

	post, appErr := a.p.API.GetPost(postID)
	if appErr != nil {
		http.Error(w, "failed to get post", http.StatusInternalServerError)
		return
	}

	// Check user has access to the post
	if !a.p.API.HasPermissionToChannel(userID, post.ChannelId, model.PermissionReadChannel) {
		a.p.API.LogError("Blocking notification preview: user does not have access to the post", "user_id", userID, "post_id", postID)
		http.Error(w, "user does not have access to the post", http.StatusForbidden)
		return
	}

	iFrameCtx, err := a.createIFrameContext(userID, post)
	if err != nil {
		a.p.API.LogError("Failed to create iFrame context", "error", err.Error())
		http.Error(w, "Failed to create iFrame context", http.StatusInternalServerError)
		return
	}

	html, err := a.formatTemplate(assets.IFrameNotificationPreviewHTMLTemplate, iFrameCtx)
	if err != nil {
		a.p.API.LogError("Failed to format iFrame HTML", "error", err.Error())
		http.Error(w, "Failed to format iFrame HTML", http.StatusInternalServerError)
		return
	}

	// cspDirectives is a CSP for the notification preview page
	// default-src: Block all resources by default
	// style-src: Allow inline styles with nonce
	// script-src: Allow scripts from Microsoft Teams CDN and jsdelivr with nonce
	// script-src-attr: Allow inline event handlers with nonce
	// connect-src: Allow connections to Microsoft and Teams domains
	// img-src: Allow images from the same origin
	// report-to: Send CSP violation reports to our endpoint
	cspDirectives := []string{
		"default-src 'none'",
		"style-src 'nonce-" + iFrameCtx.Nonce + "'",
		"script-src https://res.cdn.office.net https://cdn.jsdelivr.net 'nonce-" + iFrameCtx.Nonce + "'",
		"script-src-attr 'nonce-" + iFrameCtx.Nonce + "'",
		"connect-src https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net",
		"img-src 'self'",
		"report-to csp-endpoint",
	}

	// Set the Report-To header to define the reporting endpoint group
	reportToJSON := `{"group":"csp-endpoint","max_age":10886400,"endpoints":[{"url":"/plugins/` + iFrameCtx.PluginID + `/csp-report"}]}`
	w.Header().Set("Report-To", reportToJSON)

	w.Header().Set("Content-Security-Policy", strings.Join(cspDirectives, "; "))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(html)); err != nil {
		a.p.API.LogError("Unable to serve the iFrame", "error", err.Error())
	}
}

// createIFrameContext creates the iFrame context for the iFrame and iFrameNotificationPreview HTML templates.
func (a *API) createIFrameContext(userID string, post *model.Post) (iFrameContext, error) {
	config := a.p.API.GetConfig()
	if *config.ServiceSettings.SiteURL == "" {
		return iFrameContext{}, fmt.Errorf("ServiceSettings.SiteURL cannot be empty for MS Teams iFrame")
	}

	iFrameCtx := iFrameContext{
		SiteURL:  *config.ServiceSettings.SiteURL,
		PluginID: url.PathEscape(manifest.Id),
		TenantID: a.p.getConfiguration().TenantID,
		UserID:   userID,
		Post:     post,
	}

	// Generate a random nonce for the script/style tags
	nonceBytes := make([]byte, 16)
	if _, nonceErr := rand.Read(nonceBytes); nonceErr != nil {
		return iFrameContext{}, fmt.Errorf("failed to generate nonce: %w", nonceErr)
	}
	iFrameCtx.Nonce = base64.StdEncoding.EncodeToString(nonceBytes)

	// If the post is nil, we don't need to do anything else.
	if post == nil {
		return iFrameCtx, nil
	}

	// create context for notification preview
	postJSON, err := json.Marshal(post)
	if err != nil {
		return iFrameContext{}, fmt.Errorf("failed to marshal post: %w", err)
	}
	iFrameCtx.PostJSON = string(postJSON)

	author, appErr := a.p.API.GetUser(post.UserId)
	if appErr != nil {
		return iFrameContext{}, fmt.Errorf("failed to get author: %w", appErr)
	}

	channel, appErr := a.p.API.GetChannel(post.ChannelId)
	if appErr != nil {
		return iFrameContext{}, fmt.Errorf("failed to get channel: %w", appErr)
	}

	channelDisplayName := channel.Name
	action := "mentioned you"
	switch channel.Type {
	case model.ChannelTypeDirect:
		channelDisplayName = "Direct Message"
		action = "sent you a direct message"
	case model.ChannelTypeGroup:
		channelDisplayName = "Group Message"
		action = "sent you a group message"
	}

	iFrameCtx.NotificationPreviewContext = iFrameNotificationPreviewContext{
		PostAuthor:           author,
		Channel:              channel,
		ChannelNameDisplay:   channelDisplayName,
		PostAuthorDisplay:    author.GetDisplayName(model.ShowNicknameFullName),
		PostCreatedAtDisplay: time.Unix(post.CreateAt/1000, 0).Format("January 2, 2006 • 03:04 PM"), // Format date in this way: "April 4, 2025 • 10:43 AM"
		Action:               action,
	}

	return iFrameCtx, nil
}

// formatTemplate formats the iFrame HTML template with the site URL and plugin ID
func (a *API) formatTemplate(templateBody string, iframeCtx iFrameContext) (string, error) {
	tmpl, err := template.New("iFrame").Parse(templateBody)
	if err != nil {
		return "", fmt.Errorf("failed to parse iFrame template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, iframeCtx); err != nil {
		return "", fmt.Errorf("failed to execute iFrame template: %w", err)
	}

	return buf.String(), nil
}

// authenticate expects a Microsoft Entra ID in the Authorization header, and uses that
// to authenticate to the corresponding user in Mattermost, if it exists.
func (a *API) authenticate(w http.ResponseWriter, r *http.Request) {
	var logger logrus.FieldLogger
	logger = logrus.StandardLogger()

	// If the user is already logged in, redirect to the home page.
	// TODO: Refactor the user properties setup to a function and call it from here if the user is already logged in
	// just in case the user logs in from a tabApp in a browser.
	if r.Header.Get("Mattermost-User-ID") != "" {
		logger = logger.WithField("user_id", r.Header.Get("Mattermost-User-ID"))
		logger.Info("Skipping authentication, user already logged in")

		user, err := a.p.client.User.Get(r.Header.Get("Mattermost-User-ID"))
		if err != nil {
			logger.WithError(err).Error("Failed to get user")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, a.p.getRedirectPathFromUser(logger, user, r.URL.Query().Get("sub_entity_id")), http.StatusSeeOther)
		return
	}

	// check if the `noroute` query param is set, which will skip the routing.
	noroute := false
	_, noroute = r.URL.Query()["noroute"]

	config := a.p.client.Configuration.GetConfig()

	enableDeveloper := config.ServiceSettings.EnableDeveloper

	// Ideally we'd accept the token via an Authorization header, but for now get it from the query string.
	// token := r.Header.Get("Authorization")
	token := r.URL.Query().Get("token")

	// Validate the token in the request, handling all errors if invalid.
	expectedTenantIDs := []string{a.p.getConfiguration().TenantID}
	params := &validateTokenParams{
		jwtKeyFunc:        a.p.tabAppJWTKeyFunc,
		token:             token,
		expectedTenantIDs: expectedTenantIDs,
		enableDeveloper:   enableDeveloper != nil && *enableDeveloper,
		siteURL:           *config.ServiceSettings.SiteURL,
		clientID:          a.p.configuration.AppClientID,
		disableRouting:    noroute,
	}

	claims, validationErr := validateToken(params)
	if validationErr != nil {
		handleErrorWithCode(logger, w, validationErr.StatusCode, validationErr.Message, validationErr.Err)
		return
	}

	if claims == nil {
		handleErrorWithCode(logger, w, http.StatusUnauthorized, "Invalid token claims", nil)
		return
	}

	oid, ok := claims["oid"].(string)
	if !ok || oid == "" {
		logger.Error("Missing or empty claim for oid")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	logger = logger.WithField("oid", oid)

	ssoUsername, ok := claims["unique_name"].(string)
	if !ok || ssoUsername == "" {
		logger.Warn("Missing or empty claim for unique_name")

		ssoUsername, ok = claims["preferred_username"].(string)
		if !ok || ssoUsername == "" {
			logger.Error("Missing or empty claim for unique_name or preferred_username")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	mmUser, err := a.p.client.User.GetByEmail(ssoUsername)
	if err != nil && err != pluginapi.ErrNotFound {
		logger.WithError(err).Error("Failed to query Mattermost user matching unique_name")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	} else if mmUser == nil {
		logger.Warn("No Mattermost user matching unique_name, redirecting to login")

		// Redirect to the home page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	logger = logger.WithField("user_id", mmUser.Id)

	if mmUser.DeleteAt != 0 {
		logger.Warn("Mattermost user is archived, redirecting to login")

		// Redirect to the home page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Keep track of the unique_name and oid in the user's properties to support
	// notifications in the future.
	storedUser := pluginstore.NewUser(mmUser.Id, oid, ssoUsername)
	err = a.p.pluginStore.StoreUser(storedUser)
	if err != nil {
		logger.WithError(err).Error("Failed to store user")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	appID := r.URL.Query().Get("app_id")
	if appID == "" {
		logger.Error("App ID was not sent with the authentication request")
	}

	err = a.p.pluginStore.StoreAppID(appID)
	if err != nil {
		logger.WithError(err).Error("Failed to store app ID")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// This is effectively copied from https://github.com/mattermost/mattermost/blob/a184e5677d28433495b0cde764bfd99700838740/server/channels/app/login.go#L287
	secure := true
	maxAgeSeconds := *config.ServiceSettings.SessionLengthWebInHours * 60 * 60
	domain := getCookieDomain(config)
	subpath, _ := utils.GetSubpathFromConfig(config)

	jwtExpiresAt, err := claims.GetExpirationTime()
	if err != nil || jwtExpiresAt == nil {
		logger.WithError(err).Error("Missing or invalid expiration time claim")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	expiresAt := jwtExpiresAt.Time

	session, err := a.p.client.Session.Create(&model.Session{
		UserId:    mmUser.Id,
		ExpiresAt: model.GetMillisForTime(expiresAt),
	})
	if err != nil {
		logger.WithError(err).Error("Failed to create session for Mattermost user")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	sessionCookie := &http.Cookie{
		Name:     model.SessionCookieToken,
		Value:    session.Token,
		Path:     subpath,
		MaxAge:   maxAgeSeconds,
		Expires:  expiresAt,
		HttpOnly: true,
		Domain:   domain,
		Secure:   secure,
		SameSite: http.SameSiteNoneMode,
	}

	userCookie := &http.Cookie{
		Name:     model.SessionCookieUser,
		Value:    mmUser.Id,
		Path:     subpath,
		MaxAge:   maxAgeSeconds,
		Expires:  expiresAt,
		Domain:   domain,
		Secure:   secure,
		SameSite: http.SameSiteNoneMode,
	}

	csrfCookie := &http.Cookie{
		Name:    model.SessionCookieCsrf,
		Value:   session.GetCSRF(),
		Path:    subpath,
		MaxAge:  maxAgeSeconds,
		Expires: expiresAt,
		Domain:  domain,
		Secure:  secure,
	}

	http.SetCookie(w, sessionCookie)
	http.SetCookie(w, userCookie)
	http.SetCookie(w, csrfCookie)

	// Redirect to the home page
	http.Redirect(w, r, a.p.getRedirectPathFromUser(logger, mmUser, r.URL.Query().Get("sub_entity_id")), http.StatusSeeOther)
}

// MessageHasBeenPosted is called when a message is posted in Mattermost. We rely on it to send a user activity notification
// to Microsoft Teams when a user is mentioned in a message.
// This is called in a controller Goroutine in the server side so there's no need to worry about concurrency here.
func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// Check if user activity notifications are enabled
	if p.getConfiguration().DisableUserActivityNotifications {
		return
	}

	parser := NewNotificationsParser(p.API, p.pluginStore, p.msteamsAppClient)
	if err := parser.ProcessPost(post); err != nil {
		p.API.LogError("Failed to process mentions", "error", err.Error())
		return
	}

	if err := parser.SendNotifications(); err != nil {
		p.API.LogError("Failed to send notifications", "error", err.Error())
	}
}

func getCookieDomain(config *model.Config) string {
	if config.ServiceSettings.AllowCookiesForSubdomains != nil && *config.ServiceSettings.AllowCookiesForSubdomains && config.ServiceSettings.SiteURL != nil {
		if siteURL, err := url.Parse(*config.ServiceSettings.SiteURL); err == nil {
			return siteURL.Hostname()
		}
	}
	return ""
}

// getRedirectPathFromUser generates a redirect path for the user based on the subEntityID.
// This is used to redirect the user to the correct URL when they click on a notification in Microsoft Teams.
func (p *Plugin) getRedirectPathFromUser(logger logrus.FieldLogger, user *model.User, subEntityID string) string {
	if subEntityID != "" {
		if strings.HasPrefix(subEntityID, "post_preview_") {
			postID := strings.TrimPrefix(subEntityID, "post_preview_")
			return fmt.Sprintf("/plugins/%s/iframe/notification_preview?post_id=%s", url.PathEscape(manifest.Id), url.QueryEscape(postID))
		} else if strings.HasPrefix(subEntityID, "post_") {
			var team *model.Team
			postID := strings.TrimPrefix(subEntityID, "post_")
			post, appErr := p.API.GetPost(postID)
			if appErr != nil {
				logger.WithError(appErr).Error("Failed to get post to generate redirect path from subEntityId")
				return "/"
			}

			channel, appErr := p.API.GetChannel(post.ChannelId)
			if appErr != nil {
				logger.WithError(appErr).Error("Failed to get channel to generate redirect path from subEntityId")
				return "/"
			}

			if channel.TeamId == "" {
				var teams []*model.Team
				teams, appErr = p.API.GetTeamsForUser(user.Id)
				if appErr != nil || len(teams) == 0 {
					logger.WithError(appErr).Error("Failed to get teams for user to generate redirect path from subEntityId")
					return "/"
				}
				team = teams[0]
			} else {
				team, appErr = p.API.GetTeam(channel.TeamId)
				if appErr != nil {
					logger.WithError(appErr).Error("Failed to get team to generate redirect path from subEntityId")
					return "/"
				}
			}

			return fmt.Sprintf("/%s/pl/%s", team.Name, post.Id)
		}
	}

	return "/"
}

// cspReport handles Content Security Policy violation reports
func (a *API) cspReport(w http.ResponseWriter, r *http.Request) {
	// Limit request body size to 32KB
	const maxBodySize = 32 * 1024 // 32KB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			a.p.API.LogError("CSP report request body too large", "max_size", maxBodySize)
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		a.p.API.LogError("Failed to read CSP report request body", "error", err.Error())
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse the report-to format (as an array)
	var reportArray []struct {
		Age  int `json:"age"`
		Body struct {
			BlockedURL         *string `json:"blockedURL"`
			ColumnNumber       *int    `json:"columnNumber"`
			Disposition        *string `json:"disposition"`
			DocumentURL        *string `json:"documentURL"`
			EffectiveDirective *string `json:"effectiveDirective"`
			LineNumber         *int    `json:"lineNumber"`
			OriginalPolicy     *string `json:"originalPolicy"`
			Referrer           *string `json:"referrer"`
			ScriptSample       *string `json:"scriptSample"`
			SourceFile         *string `json:"sourceFile"`
			ViolatedDirective  *string `json:"violatedDirective"`
		} `json:"body"`
	}

	// Parse the report
	if err := json.Unmarshal(body, &reportArray); err != nil {
		a.p.API.LogError("Failed to parse CSP report", "error", err.Error(), "body", string(body))
		http.Error(w, "Failed to parse report", http.StatusBadRequest)
		return
	}

	// Process each report in the array
	for i, report := range reportArray {
		// Create a map to store the fields that are not null
		fields := map[string]interface{}{
			"index": i,
			"age":   report.Age,
		}

		// Add non-null fields to the map
		if report.Body.BlockedURL != nil {
			fields["blocked-url"] = *report.Body.BlockedURL
		}
		if report.Body.ColumnNumber != nil {
			fields["column-number"] = *report.Body.ColumnNumber
		}
		if report.Body.Disposition != nil {
			fields["disposition"] = *report.Body.Disposition
		}
		if report.Body.DocumentURL != nil {
			fields["document-url"] = *report.Body.DocumentURL
		}
		if report.Body.EffectiveDirective != nil {
			fields["effective-directive"] = *report.Body.EffectiveDirective
		}
		if report.Body.LineNumber != nil {
			fields["line-number"] = *report.Body.LineNumber
		}
		if report.Body.OriginalPolicy != nil {
			fields["original-policy"] = *report.Body.OriginalPolicy
		}
		if report.Body.Referrer != nil {
			fields["referrer"] = *report.Body.Referrer
		}
		if report.Body.ScriptSample != nil {
			fields["script-sample"] = *report.Body.ScriptSample
		}
		if report.Body.SourceFile != nil {
			fields["source-file"] = *report.Body.SourceFile
		}
		if report.Body.ViolatedDirective != nil {
			fields["violated-directive"] = *report.Body.ViolatedDirective
		}

		// Log the CSP violation with only the non-null fields
		a.p.API.LogError("CSP violation detected", fields)
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)
}
