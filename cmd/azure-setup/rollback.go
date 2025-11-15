// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
)

// executeRollback executes all rollback functions in reverse order
func executeRollback(config *SetupConfig) {
	if len(config.rollback) == 0 {
		return
	}

	if config.Verbose {
		fmt.Println("\n🔄 Executing rollback operations...")
	}

	// Execute rollback functions in reverse order
	for i := len(config.rollback) - 1; i >= 0; i-- {
		rollbackFunc := config.rollback[i]
		if err := rollbackFunc(); err != nil {
			if config.Verbose {
				fmt.Printf("   ⚠️  Rollback operation failed: %v\n", err)
			}
		}
	}

	if config.Verbose {
		fmt.Println("✅ Rollback complete")
	}
}
