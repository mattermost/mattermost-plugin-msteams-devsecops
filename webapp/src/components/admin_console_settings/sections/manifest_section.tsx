// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

interface Props {
    children: React.ReactNode;
    settings?: Record<string, string | boolean>;
    id?: string;
    label?: string;
    helpText?: React.ReactNode;
    disabled?: boolean;
    setByEnv?: boolean;
}

const ManifestSection: React.FC<Props> = (props) => {
    // Debug props
    console.log('ManifestSection props:', props);

    // Debug log
    console.log('ManifestSection props:', props);
    
    return (
        <div className='wrapper--fixed'>
            <div className='admin-console__header'>
                <h1>{'Manifest Settings'}</h1>
                <p className='admin-console__header-subtitle'>
                    {'These settings are used to generate the Microsoft Teams app manifest which you will upload to your Microsoft Teams tenant.'}
                </p>
            </div>
            <div className='admin-console__wrapper'>
                <div className='admin-console__content'>
                    {props.children}
                </div>
            </div>
        </div>
    );
};

export default ManifestSection;
