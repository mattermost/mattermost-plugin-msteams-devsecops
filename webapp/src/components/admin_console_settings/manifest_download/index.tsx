// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';

// Custom event for input changes
const EVENT_APP_INPUT_CHANGE = 'app_input_change';

// Type for the custom event data
interface AppInputChangeEvent {
    id: string;
    value: string;
}

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
    const [currentValues, setCurrentValues] = useState<Record<string, string>>({
        app_id: '',
        app_name: '',
        app_version: ''
    });
    
    // Initialize from config
    useEffect(() => {
        if (!props.config) {
            return;
        }
        
        // Get the plugin settings
        const pluginSettings = props.config.PluginSettings?.Plugins?.['com.mattermost.plugin-msteams-devsecops'];
        if (!pluginSettings) {
            return;
        }
        
        // Update our tracking of current values with config values
        setCurrentValues(prev => ({
            ...prev,
            app_id: pluginSettings.app_id || '',
            app_name: pluginSettings.app_name || '',
            app_version: pluginSettings.app_version || ''
        }));
    }, [props.config]);
    
    // Listen for input changes from other components
    useEffect(() => {
        const handleInputChange = (e: CustomEvent<AppInputChangeEvent>) => {
            const {id, value} = e.detail;
            console.log('ManifestDownload received input change:', id, value);
            
            // Extract the setting key from the ID (e.g., app_id from PluginSettings.Plugins.com+mattermost+plugin-msteams-devsecops.app_id)
            const settingKey = id.split('.').pop() || '';
            
            if (['app_id', 'app_name', 'app_version'].includes(settingKey)) {
                setCurrentValues(prev => ({
                    ...prev,
                    [settingKey]: value
                }));
            }
        };
        
        // Add event listener
        window.addEventListener(EVENT_APP_INPUT_CHANGE, handleInputChange as EventListener);
        
        return () => {
            // Remove event listener on cleanup
            window.removeEventListener(EVENT_APP_INPUT_CHANGE, handleInputChange as EventListener);
        };
    }, []);
    
    // Validate settings whenever they change
    useEffect(() => {
        // Check if all required fields have values
        const hasAllValues = Boolean(
            currentValues.app_id?.trim() &&
            currentValues.app_name?.trim() &&
            currentValues.app_version?.trim()
        );
        
        console.log('ManifestDownload validating values:', currentValues, hasAllValues);
        setIsDownloadEnabled(hasAllValues);
    }, [currentValues]);

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
