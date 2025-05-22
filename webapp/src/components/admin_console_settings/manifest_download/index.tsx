// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';

interface Props {
    id: string;
    label: string;
    helpText?: React.ReactNode;
    disabled?: boolean;
    config?: Record<string, any>;
}

const ManifestDownload: React.FC<Props> = (props) => {
    // Debug log available props
    console.log('ManifestDownload props:', props);
    console.log('ManifestDownload config:', props.config);

    const [isDownloadEnabled, setIsDownloadEnabled] = useState(false);
    
    // Check if all required fields have values
    useEffect(() => {
        if (!props.config) {
            return;
        }
        
        // Get the plugin settings
        const pluginSettings = props.config.PluginSettings?.Plugins?.['com.mattermost.plugin-msteams-devsecops'];
        if (!pluginSettings) {
            return;
        }
        
        // Check if all required fields have values
        const hasAllValues = Boolean(
            pluginSettings.app_id?.trim() &&
            pluginSettings.app_name?.trim() &&
            pluginSettings.app_version?.trim()
        );
        
        console.log('ManifestDownload checking settings:', {
            appId: pluginSettings.app_id,
            appName: pluginSettings.app_name,
            appVersion: pluginSettings.app_version,
            hasAllValues
        });
        
        setIsDownloadEnabled(hasAllValues);
    }, [props.config]);

    return (
        <div className='form-group'>
            <label className='control-label'>{props.label}</label>
            <div className='help-text'>{props.helpText}</div>
            <div className='col-sm-8'>
                <a
                    href={isDownloadEnabled ? '/plugins/com.mattermost.plugin-msteams-devsecops/iframe-manifest' : '#'}
                    className={`btn ${isDownloadEnabled ? 'btn-primary' : 'btn-inactive'}`}
                    rel='noreferrer'
                    target='_self'
                    style={{
                        marginTop: '8px',
                        pointerEvents: isDownloadEnabled ? 'auto' : 'none',
                        opacity: isDownloadEnabled ? 1 : 0.6,
                    }}
                    download={isDownloadEnabled}
                    onClick={isDownloadEnabled ? undefined : (e) => e.preventDefault()}
                >
                    {'Download Manifest'}
                </a>
                <p>
                    {'To embed Mattermost within Microsoft Teams, an application manifest can be downloaded and installed as a MS Teams app. '}
                    {'Clicking the Download button below will generate an application manifest that will embed this instance of Mattermost. '}
                </p>
            </div>
        </div>
    );
};

export default ManifestDownload;
