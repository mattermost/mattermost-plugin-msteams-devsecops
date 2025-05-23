// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useRef, useEffect} from 'react';

import {EVENT_APP_INPUT_CHANGE} from '../constants';

interface Props {
    id: string;
    label: string;
    helpText?: React.ReactNode;
    value?: string;
    disabled?: boolean;
    onChange: (id: string, value: string) => void;
}

// Component for uploading manifest icons
const IconUpload: React.FC<Props> = (props) => {
    const [preview, setPreview] = useState<string | null>(props.value || null);
    const [defaultIcon, setDefaultIcon] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isUploading, setIsUploading] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    // Determine which default icon to fetch based on the label
    const isColorIcon = props.label.toLowerCase().includes('color');
    const defaultIconPath = isColorIcon ? '/plugins/com.mattermost.plugin-msteams-devsecops/icons/default/color' : '/plugins/com.mattermost.plugin-msteams-devsecops/icons/default/outline';

    // Fetch default icon on component mount
    useEffect(() => {
        const fetchDefaultIcon = async () => {
            try {
                const response = await fetch(defaultIconPath);
                if (response.ok) {
                    const blob = await response.blob();
                    const reader = new FileReader();
                    reader.onload = () => {
                        setDefaultIcon(reader.result as string);
                    };
                    reader.readAsDataURL(blob);
                } else {
                    setError(`Failed to load default ${props.label.toLowerCase()} icon.`);
                }
            } catch (err) {
                setError(`Failed to load default ${props.label.toLowerCase()} icon.`);
            }
        };

        fetchDefaultIcon();
    }, [defaultIconPath, props.label]);

    const validateImage = (file: File): Promise<boolean> => {
        return new Promise((resolve) => {
            // Check file type
            if (!file.type.startsWith('image/png')) {
                setError('Please upload a PNG image file.');
                resolve(false);
                return;
            }

            // Check file size (reasonable limit for 192x192 PNG)
            if (file.size > 5 * 1024 * 1024) { // 5MB limit
                setError('File size too large. Please upload an image under 5MB.');
                resolve(false);
                return;
            }

            // Check image dimensions
            const img = new Image();
            img.onload = () => {
                if (img.width !== 192 || img.height !== 192) {
                    setError('Image must be exactly 192x192 pixels.');
                    resolve(false);
                } else {
                    setError(null);
                    resolve(true);
                }
            };
            img.onerror = () => {
                setError('Invalid image file.');
                resolve(false);
            };
            img.src = URL.createObjectURL(file);
        });
    };

    const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) {
            return;
        }

        setIsUploading(true);
        setError(null);

        const isValid = await validateImage(file);
        if (!isValid) {
            setIsUploading(false);
            return;
        }

        // Create preview
        const reader = new FileReader();
        reader.onload = (event) => {
            const result = event.target?.result as string;
            setPreview(result);
            setIsUploading(false);

            // For now, we'll store the data URL
            // TODO: In the next phase, this will upload to the file store
            props.onChange(props.id, result);

            // Dispatch custom event for real-time validation
            window.dispatchEvent(new CustomEvent(EVENT_APP_INPUT_CHANGE, {
                detail: {
                    id: props.id,
                    value: result,
                },
            }));
        };
        reader.readAsDataURL(file);
    };

    const handleUploadClick = () => {
        fileInputRef.current?.click();
    };

    const handleRemove = () => {
        setPreview(null);
        setError(null);
        props.onChange(props.id, '');
        if (fileInputRef.current) {
            fileInputRef.current.value = '';
        }

        // Dispatch custom event for real-time validation
        window.dispatchEvent(new CustomEvent(EVENT_APP_INPUT_CHANGE, {
            detail: {
                id: props.id,
                value: '',
            },
        }));
    };

    // Show custom icon if available, otherwise show default icon
    const iconToShow = preview || defaultIcon;
    const isCustomIcon = Boolean(preview);

    return (
        <div className='form-group'>
            <label className='control-label'>{props.label}</label>
            <div className='help-text'>{props.helpText}</div>
            <div className='col-sm-8'>
                <div style={{marginBottom: '10px'}}>
                    {iconToShow ? (
                        <div style={{display: 'flex', alignItems: 'center', gap: '10px'}}>
                            <img
                                src={iconToShow}
                                alt={`${props.label} preview`}
                                style={{
                                    width: '48px',
                                    height: '48px',
                                    border: '1px solid #ddd',
                                    borderRadius: '4px',
                                    objectFit: 'cover',
                                }}
                            />
                            <div style={{display: 'flex', flexDirection: 'column', gap: '5px'}}>
                                <button
                                    type='button'
                                    className='btn btn-sm btn-primary'
                                    onClick={handleUploadClick}
                                    disabled={props.disabled || isUploading}
                                >
                                    {(() => {
                                        if (isUploading) {
                                            return 'Uploading...';
                                        }
                                        if (isCustomIcon) {
                                            return 'Replace';
                                        }
                                        return `Upload ${props.label} Icon`;
                                    })()}
                                </button>
                                {isCustomIcon && (
                                    <button
                                        type='button'
                                        className='btn btn-sm btn-link'
                                        onClick={handleRemove}
                                        disabled={props.disabled || isUploading}
                                        style={{padding: '0', fontSize: '12px'}}
                                    >
                                        {'Remove'}
                                    </button>
                                )}
                            </div>
                        </div>
                    ) : (
                        <button
                            type='button'
                            className='btn btn-primary'
                            onClick={handleUploadClick}
                            disabled={props.disabled || isUploading}
                        >
                            {isUploading ? 'Uploading...' : `Upload ${props.label} Icon`}
                        </button>
                    )}
                </div>
                {error && (
                    <div
                        className='help-text text-danger'
                        style={{marginBottom: '10px'}}
                    >
                        {error}
                    </div>
                )}
                <div
                    className='help-text text-muted'
                    style={{fontSize: '12px'}}
                >
                    {'Must be a PNG image, exactly 192x192 pixels'}
                </div>
                <input
                    ref={fileInputRef}
                    type='file'
                    accept='image/png'
                    onChange={handleFileSelect}
                    style={{display: 'none'}}
                />
            </div>
        </div>
    );
};

export default IconUpload;
