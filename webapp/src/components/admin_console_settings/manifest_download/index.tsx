// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

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

    return (
        <div className='form-group'>
            <label className='control-label'>{props.label}</label>
            <div className='help-text'>{props.helpText}</div>
            <div className='col-sm-8'>
                <a
                    href={'/plugins/com.mattermost.plugin-msteams-devsecops/iframe-manifest'}
                    className='btn btn-primary'
                    rel='noreferrer'
                    target='_self'
                    style={{marginTop: '8px'}}
                    download={true}
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
