// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {ReactNode} from 'react';
import React, {createContext, useState, useContext, useEffect} from 'react';

export interface ManifestSettings {
    appId: string;
    appName: string;
    appVersion: string;
}

interface ManifestContextType {
    manifestSettings: ManifestSettings;
    updateSetting: (key: keyof ManifestSettings, value: string) => void;
    isValid: boolean;
}

const defaultSettings: ManifestSettings = {
    appId: '',
    appName: '',
    appVersion: '',
};

export const ManifestContext = createContext<ManifestContextType>({
    manifestSettings: defaultSettings,
    updateSetting: () => {},
    isValid: false,
});

interface ManifestProviderProps {
    children: ReactNode;
    initialValues?: Partial<ManifestSettings>;
}

export const ManifestProvider: React.FC<ManifestProviderProps> = ({children, initialValues = {}}) => {
    console.log('ManifestProvider initialValues:', initialValues);

    // Check if all required fields are present in initial values
    const hasAllValues = Boolean(
        initialValues.appId?.trim() &&
        initialValues.appName?.trim() &&
        initialValues.appVersion?.trim(),
    );

    const [manifestSettings, setManifestSettings] = useState<ManifestSettings>({
        ...defaultSettings,
        ...initialValues,
    });

    // Initialize to true if all required fields are already present
    const [isValid, setIsValid] = useState(hasAllValues);

    // One-time check of initial values
    useEffect(() => {
        console.log('ManifestProvider initializing with:', initialValues);
        const hasInitialValues = Boolean(
            initialValues.appId?.trim() &&
            initialValues.appName?.trim() &&
            initialValues.appVersion?.trim(),
        );
        console.log('Initial values valid?', hasInitialValues);

        if (hasInitialValues) {
            setIsValid(true);
        }
    }, [initialValues]);

    const updateSetting = (key: keyof ManifestSettings, value: string) => {
        console.log('ManifestProvider updateSetting:', key, value);

        setManifestSettings((prevSettings) => {
            const newSettings = {
                ...prevSettings,
                [key]: value,
            };
            console.log('ManifestProvider new settings:', newSettings);
            return newSettings;
        });
    };

    // Validate settings whenever they change
    useEffect(() => {
        const valid = Boolean(
            manifestSettings.appId.trim() &&
            manifestSettings.appName.trim() &&
            manifestSettings.appVersion.trim(),
        );
        console.log('ManifestProvider validation:', manifestSettings, valid);
        setIsValid(valid);
    }, [manifestSettings]);

    return (
        <ManifestContext.Provider value={{manifestSettings, updateSetting, isValid}}>
            {children}
        </ManifestContext.Provider>
    );
};

export const useManifestContext = () => useContext(ManifestContext);
