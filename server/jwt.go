// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/sirupsen/logrus"
)

const (
	MicrosoftOnlineJWKSURL = "https://login.microsoftonline.com/common/discovery/v2.0/keys"
	ExpectedAudienceFmt    = "api://%s/%s"
)

type validationError struct {
	StatusCode int
	Message    string
	Err        error
}

func (ve validationError) Error() string {
	return ve.Message
}

type validateTokenParams struct {
	jwtKeyFunc          keyfunc.Keyfunc
	token               string
	expectedTenantIDs   []string
	skipTokenValidation bool
	siteURL             string
	clientID            string
}

func setupJWKSet() (keyfunc.Keyfunc, context.CancelFunc) {
	// Setup JWK set to assist in verifying JWTs passed from Microsoft Teams.
	ctx, cancelCtx := context.WithCancel(context.Background())

	k, err := keyfunc.NewDefaultCtx(ctx, []string{MicrosoftOnlineJWKSURL})
	if err != nil {
		logrus.WithError(err).WithField("jwks_url", MicrosoftOnlineJWKSURL).Error("Failed to create a keyfunc.Keyfunc")
	}
	logrus.Info("Started JWKS monitor")

	return k, cancelCtx
}

func validateToken(params *validateTokenParams) (jwt.MapClaims, *validationError) {
	if params.token == "" {
		if params.skipTokenValidation {
			logrus.Warn("Skipping token validation check for empty token since skip token validation mode enabled")
			return nil, nil
		}
		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Missing token",
		}
	}

	if params.jwtKeyFunc == nil {
		return nil, &validationError{
			StatusCode: http.StatusInternalServerError,
			Message:    "Failed to initialize token validation",
		}
	}

	options := []jwt.ParserOption{
		// See https://golang-jwt.github.io/jwt/usage/signing_methods/ -- this is effectively all
		// asymetric signing methods so that we exclude both the symmetric signing methods as
		// well as the "none" algorithm.
		//
		// In practice, the upstream library already chokes on the HMAC validate method expecting
		// a []byte but getting a public key object, but this is more explicit.
		jwt.WithValidMethods([]string{
			jwt.SigningMethodES256.Alg(),
			jwt.SigningMethodES384.Alg(),
			jwt.SigningMethodES512.Alg(),
			jwt.SigningMethodRS256.Alg(),
			jwt.SigningMethodRS384.Alg(),
			jwt.SigningMethodRS512.Alg(),
			jwt.SigningMethodPS256.Alg(),
			jwt.SigningMethodPS384.Alg(),
			jwt.SigningMethodPS512.Alg(),
			jwt.SigningMethodEdDSA.Alg(),
		}),
		// Require iat claim, and verify the token is not used before issue.
		jwt.WithIssuedAt(),
		// Require the exp claim: the library always verifies if the claim is present.
		jwt.WithExpirationRequired(),
		// There's no WithNotBefore() helper, but the library always verifies if the claim is present.
	}

	mmServerURL, err := url.Parse(params.siteURL)
	if err != nil {
		logrus.WithError(err).WithField("site_url", params.siteURL).Warn("Failed to parse site URL, check your system console")
		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Failed to authenticate due to Mattermost server missconfiguration. Contact your system administrator.",
		}
	}

	// Verify that this token was signed for the expected app, unless skip token validation mode is enabled.
	switch {
	case params.skipTokenValidation:
		logrus.Warn("Skipping aud claim check for token since skip token validation mode enabled")
	default:
		options = append(options, jwt.WithAudience(fmt.Sprintf(ExpectedAudienceFmt, mmServerURL.Host, params.clientID)))
	}

	parsed, err := jwt.Parse(
		params.token,
		params.jwtKeyFunc.Keyfunc,
		options...,
	)
	if err != nil {
		logrus.WithError(err).Warn("Rejected invalid token")

		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Failed to parse token",
			Err:        err,
		}
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		logrus.Warn("Validated token, but failed to parse claims")

		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Unexpected claims",
		}
	}

	logger := logrus.WithFields(logrus.Fields{
		"aud":                 claims["aud"],
		"tid":                 claims["tid"],
		"oid":                 claims["oid"],
		"expected_tenant_ids": params.expectedTenantIDs,
	})

	// Verify the iat was present. The library is configured above to check
	// its value is not in the future if present, but doesn't enforce its
	// presence.
	if iat, _ := parsed.Claims.GetIssuedAt(); iat == nil {
		logger.Warn("Validated token, but rejected request on missing iat")
		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Unexpected claims",
		}
	}

	// Verify the nbp was present. The library is configured above to check
	// its value is not in the future if present, but doesn't enforce its
	// presence.
	if nbf, _ := parsed.Claims.GetNotBefore(); nbf == nil {
		logger.Warn("Validated token, but rejected request on missing nbf")
		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Unexpected claims",
		}
	}

	// Verify the tid is a GUID
	if tid, ok := claims["tid"].(string); !ok {
		logger.Warn("Validated token, but rejected request on missing tid")
		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Unexpected claims",
		}
	} else if _, err = uuid.Parse(tid); err != nil {
		logger.Warn("Validated token, but rejected request on non-GUID tid")
		return nil, &validationError{
			StatusCode: http.StatusUnauthorized,
			Message:    "Unexpected claims",
		}
	}

	for _, expectedTenantID := range params.expectedTenantIDs {
		if claims["tid"] == expectedTenantID {
			logger.Info("Validated token, and authorized request from matching tenant")
			return claims, nil
		} else if params.skipTokenValidation && expectedTenantID == "*" {
			logger.Warn("Validated token, but authorized request from wildcard tenant since skip token validation mode enabled")
			return claims, nil
		}
	}

	logger.Warn("Validated token, but rejected request on tenant mismatch")
	return nil, &validationError{
		StatusCode: http.StatusUnauthorized,
		Message:    "Unexpected claims",
	}
}
