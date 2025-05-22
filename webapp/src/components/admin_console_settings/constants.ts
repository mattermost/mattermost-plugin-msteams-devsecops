// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Custom event for input changes
export const EVENT_APP_INPUT_CHANGE = 'app_input_change';

// Type for the custom event data
export interface AppInputChangeEvent {
    id: string;
    value: string;
}