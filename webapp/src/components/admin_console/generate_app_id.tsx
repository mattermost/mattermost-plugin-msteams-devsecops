// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {v4 as uuidv4} from 'uuid';

type Props = {
    label: string;
    disabled: boolean;
};

export default function GenerateAppID(props: Props) {
    const handleClick = () => {
        if (props.disabled) {
            return;
        }
        const newAppID = uuidv4();
        const appIDElement = document.getElementById('appID');
        if (appIDElement) {
            appIDElement.textContent = newAppID;
        }
    };

    return (
        <div>
            <a
                onClick={handleClick}
                className='btn btn-primary'
                rel='noreferrer'
                target='_self'
                style={styles.buttonBorder}
                download={true}
            >
                {props.label}
            </a>
        </div>
    );
}

const styles = {
    buttonBorder: {
        marginTop: '8px',
    },
};
