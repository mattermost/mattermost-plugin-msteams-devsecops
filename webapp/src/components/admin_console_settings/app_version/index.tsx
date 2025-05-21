// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

interface Props {
    id: string;
    label: string;
    helpText?: React.ReactNode;
    placeholder?: string;
    value: string;
    disabled?: boolean;
    onChange: (id: string, value: string) => void;
}

const AppVersion: React.FC<Props> = (props) => {
    // Basic debugging log
    console.log('AppVersion props:', props);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = e.target.value;
        props.onChange(props.id, newValue);
    };

    return (
        <div className='form-group'>
            <label className='control-label'>{props.label}</label>
            <div className='help-text'>{props.helpText}</div>
            <div className='col-sm-8'>
                <input
                    id={props.id}
                    className='form-control'
                    type='text'
                    placeholder={props.placeholder}
                    value={props.value}
                    onChange={handleChange}
                    disabled={props.disabled}
                />
            </div>
        </div>
    );
};

export default AppVersion;
