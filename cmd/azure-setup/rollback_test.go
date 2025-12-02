// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteRollback(t *testing.T) {
	t.Run("no rollback functions", func(t *testing.T) {
		config := &SetupConfig{
			rollback: []func() error{},
			Verbose:  true,
		}

		// Should not panic with empty rollback slice
		executeRollback(config)
	})

	t.Run("single successful rollback", func(t *testing.T) {
		executed := false
		config := &SetupConfig{
			rollback: []func() error{
				func() error {
					executed = true
					return nil
				},
			},
			Verbose: false,
		}

		executeRollback(config)
		assert.True(t, executed, "rollback function should be executed")
	})

	t.Run("multiple rollback functions in reverse order", func(t *testing.T) {
		var executionOrder []int
		config := &SetupConfig{
			rollback: []func() error{
				func() error {
					executionOrder = append(executionOrder, 1)
					return nil
				},
				func() error {
					executionOrder = append(executionOrder, 2)
					return nil
				},
				func() error {
					executionOrder = append(executionOrder, 3)
					return nil
				},
			},
			Verbose: false,
		}

		executeRollback(config)

		// Should execute in reverse order: 3, 2, 1
		assert.Len(t, executionOrder, 3)
		assert.Equal(t, []int{3, 2, 1}, executionOrder, "rollback should execute in reverse order")
	})

	t.Run("continues on rollback error", func(t *testing.T) {
		var executionOrder []int
		config := &SetupConfig{
			rollback: []func() error{
				func() error {
					executionOrder = append(executionOrder, 1)
					return nil
				},
				func() error {
					executionOrder = append(executionOrder, 2)
					return errors.New("rollback error")
				},
				func() error {
					executionOrder = append(executionOrder, 3)
					return nil
				},
			},
			Verbose: true,
		}

		// Should not panic even if a rollback function fails
		executeRollback(config)

		// All rollback functions should still be attempted
		assert.Len(t, executionOrder, 3)
		assert.Equal(t, []int{3, 2, 1}, executionOrder, "should execute all rollback functions despite errors")
	})

	t.Run("all rollback functions fail", func(t *testing.T) {
		executionCount := 0
		config := &SetupConfig{
			rollback: []func() error{
				func() error {
					executionCount++
					return errors.New("error 1")
				},
				func() error {
					executionCount++
					return errors.New("error 2")
				},
			},
			Verbose: false,
		}

		// Should not panic
		executeRollback(config)

		// All functions should still be called
		assert.Equal(t, 2, executionCount)
	})

	t.Run("verbose mode", func(t *testing.T) {
		config := &SetupConfig{
			rollback: []func() error{
				func() error {
					return nil
				},
			},
			Verbose: true,
		}

		// Should not panic in verbose mode
		executeRollback(config)
	})
}

func TestRollbackIntegration(t *testing.T) {
	t.Run("cleanup tracking", func(t *testing.T) {
		// Simulate creating resources and tracking them for rollback
		var createdResources []string
		config := &SetupConfig{
			rollback: []func() error{},
			Verbose:  false,
		}

		// Simulate creating resource 1
		createdResources = append(createdResources, "resource-1")
		config.rollback = append(config.rollback, func() error {
			// Remove resource 1
			for i, r := range createdResources {
				if r == "resource-1" {
					createdResources = append(createdResources[:i], createdResources[i+1:]...)
					break
				}
			}
			return nil
		})

		// Simulate creating resource 2
		createdResources = append(createdResources, "resource-2")
		config.rollback = append(config.rollback, func() error {
			// Remove resource 2
			for i, r := range createdResources {
				if r == "resource-2" {
					createdResources = append(createdResources[:i], createdResources[i+1:]...)
					break
				}
			}
			return nil
		})

		// Verify resources were created
		assert.Len(t, createdResources, 2)

		// Execute rollback
		executeRollback(config)

		// Verify all resources were cleaned up
		assert.Len(t, createdResources, 0, "all resources should be cleaned up")
	})
}
