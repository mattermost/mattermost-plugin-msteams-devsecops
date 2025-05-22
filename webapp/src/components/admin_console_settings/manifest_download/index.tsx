// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';

import {EVENT_APP_INPUT_CHANGE} from '../constants';
import type {AppInputChangeEvent} from '../constants';

interface Props {
    label: string;
    helpText?: React.ReactNode;
    config?: Record<string, any>;
}

const ManifestDownload: React.FC<Props> = (props) => {
    // Component for downloading the manifest

    const [isDownloadEnabled, setIsDownloadEnabled] = useState(false);
    const [currentValues, setCurrentValues] = useState<Record<string, string>>({
        app_id: '',
        app_name: '',
        app_version: '',
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
        setCurrentValues((prev) => ({
            ...prev,
            app_id: pluginSettings.app_id || '',
            app_name: pluginSettings.app_name || '',
            app_version: pluginSettings.app_version || '',
        }));
    }, [props.config]);

    // Listen for input changes from other components
    useEffect(() => {
        const handleInputChange = (e: CustomEvent<AppInputChangeEvent>) => {
            const {id, value} = e.detail;

            // Process input change from other components
            // Extract the setting key from the ID (e.g., app_id from PluginSettings.Plugins.com+mattermost+plugin-msteams-devsecops.app_id)
            const settingKey = id.split('.').pop() || '';

            if (['app_id', 'app_name', 'app_version'].includes(settingKey)) {
                setCurrentValues((prev) => ({
                    ...prev,
                    [settingKey]: value,
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
            currentValues.app_version?.trim(),
        );

        // Update button state based on validation
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
            </div>
        </div>
    );
};

export default ManifestDownload;
