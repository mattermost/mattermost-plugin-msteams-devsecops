// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Custom event for input changes
export const EVENT_APP_INPUT_CHANGE = 'com.mattermost.plugin-msteams-devsecops__app_input_change';

// Type for the custom event data
export interface AppInputChangeEvent {
    id: string;
    value: string;
}

// Type for plugin settings
export interface PluginSettings {
    app_id?: string;
    app_name?: string;
    app_version?: string;
    icon_color_path?: string;
    icon_outline_path?: string;
    [key: string]: string | undefined;
}

// Type for config object structure
export interface AdminConsoleConfig {
    PluginSettings?: {
        Plugins?: {
            'com.mattermost.plugin-msteams-devsecops'?: PluginSettings;
            [key: string]: unknown;
        };
        [key: string]: unknown;
    };
    [key: string]: unknown;
}
