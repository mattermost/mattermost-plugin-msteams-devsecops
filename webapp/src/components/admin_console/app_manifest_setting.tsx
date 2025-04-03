// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

type Props = {
    label: string;
    disabled: boolean;
};

const MSTeamsAppManifestSetting = (props: Props) => {
    return (
        <div>
            <p>
                {'To embed Mattermost within Microsoft Teams, an application manifest can be downloaded and installed as a MS Teams app. '}
                {'Clicking the Download button below will generate an application manifest that will embed this instance of Mattermost. '}
            </p>
            <a
                href={props.disabled ? undefined : '/plugins/com.mattermost.plugin-msteams-devsecops/iframe-manifest'}
                className={`btn btn-primary ${props.disabled ? 'disabled' : ''}`}
                rel='noreferrer'
                target='_self'
                style={styles.buttonBorder}
                download={!props.disabled}
                onClick={(e) => {
                    if (props.disabled) {
                        e.preventDefault();
                    }
                }}
            >
                {props.label}
            </a>
        </div>
    );
};

const styles = {
    buttonBorder: {
        marginTop: '8px',
    },
};

export default MSTeamsAppManifestSetting;
