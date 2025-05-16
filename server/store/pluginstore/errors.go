// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package pluginstore

import "errors"

type ErrNotFound struct {
	Message string
}

func (e *ErrNotFound) Error() string {
	return e.Message
}

// NewErrNotFound creates a new ErrNotFound with the given message.
func NewErrNotFound(msg string) *ErrNotFound {
	return &ErrNotFound{Message: msg}
}

// IsErrNotFound returns true if the error is of type ErrNotFound, even if wrapped.
func IsErrNotFound(err error) bool {
	var notFoundErr *ErrNotFound
	return errors.As(err, &notFoundErr)
}
