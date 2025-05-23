// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Store, Action} from 'redux';

import type {GlobalState} from '@mattermost/types/store';

import AppID from '@/components/admin_console_settings/app_id';
import AppName from '@/components/admin_console_settings/app_name';
import AppVersion from '@/components/admin_console_settings/app_version';
import IconUpload from '@/components/admin_console_settings/icon_upload';
import ManifestDownload from '@/components/admin_console_settings/manifest_download';
import ManifestSection from '@/components/admin_console_settings/sections/manifest_section';
import manifest from '@/manifest';
import type {PluginRegistry} from '@/types/mattermost-webapp';

class Plugin {
    public async initialize(
        registry: PluginRegistry,
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        _store: Store<GlobalState, Action<Record<string, unknown>>>,
    ): Promise<void> {
        // Register components directly without providers

        // Register custom settings components
        registry.registerAdminConsoleCustomSetting('app_id', AppID);
        registry.registerAdminConsoleCustomSetting('app_name', AppName);
        registry.registerAdminConsoleCustomSetting('app_version', AppVersion);
        registry.registerAdminConsoleCustomSetting('icon_color_path', IconUpload);
        registry.registerAdminConsoleCustomSetting('icon_outline_path', IconUpload);
        registry.registerAdminConsoleCustomSetting('app_manifest_download', ManifestDownload);

        // Register the section
        registry.registerAdminConsoleCustomSetting('manifest_settings', ManifestSection);

        // @see https://developers.mattermost.com/extend/plugins/webapp/reference/
    }
}

declare global {
    interface Window {
        registerPlugin: (pluginId: string, plugin: Plugin) => void;
    }
}

window.registerPlugin(manifest.id, new Plugin());

