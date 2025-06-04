package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultCSPConnectSrc = "https://*.microsoft.com https://*.teams.microsoft.com https://*.cdn.office.net"
	DefaultCSPScriptSrc  = "https://res.cdn.office.net https://cdn.jsdelivr.net"
)

// return returnCSPHeaderssets and returns the Content Security Policy headers for the iframe context.
func (a *API) returnCSPHeaders(w http.ResponseWriter, iFrameCtx iFrameContext) {
	cspDirectives := []string{
		// default-src: Block all resources by default
		"default-src 'none'",
		// script-src: Allow scripts from provided sources (like Microsoft Teams CDN and jsdelivr) with nonce
		"script-src " + iFrameCtx.CSPScriptSrc + " 'nonce-" + iFrameCtx.Nonce + "'",
		// style-src: Allow inline styles with nonce
		"style-src 'nonce-" + iFrameCtx.Nonce + "'",
		// script-src-attr: Allow inline event handlers with nonce
		"script-src-attr 'nonce-" + iFrameCtx.Nonce + "'",
		// connect-src: Allow connections to provided domains (like Microsoft and Teams domains)
		"connect-src " + iFrameCtx.CSPConnectSrc,
		// img-src: Allow images from the same origin
		"img-src 'self'",
		// report-to: Send CSP violation reports to our endpoint
		"report-to csp-endpoint",
	}

	if iFrameCtx.CSPFrameSrc != "" {
		cspDirectives = append(cspDirectives, "frame-src '"+iFrameCtx.CSPFrameSrc+"'")
	}

	// Set the Report-To header to define the reporting endpoint group
	reportToJSON := `{"group":"csp-endpoint","max_age":10886400,"endpoints":[{"url":"/plugins/` + iFrameCtx.PluginID + `/csp-report"}]}`
	w.Header().Set("Report-To", reportToJSON)

	w.Header().Set("Content-Security-Policy", strings.Join(cspDirectives, "; "))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
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
