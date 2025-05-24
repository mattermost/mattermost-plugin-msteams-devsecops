// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useRef, useEffect} from 'react';

import {Client4} from 'mattermost-redux/client';

import {EVENT_APP_INPUT_CHANGE} from '../constants';

enum IconType {
    COLOR = 'color',
    OUTLINE = 'outline',
}

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

    // Determine which icon to fetch based on the label
    const isColorIcon = props.label.toLowerCase().includes('color');
    const iconType = isColorIcon ? IconType.COLOR : IconType.OUTLINE;
    const iconPath = `/plugins/com.mattermost.plugin-msteams-devsecops/icons/${iconType}`;

    // Fetch icon on component mount (custom or default)
    useEffect(() => {
        const fetchIcon = async () => {
            try {
                const response = await fetch(iconPath, Client4.getOptions({method: 'GET'}));
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

        fetchIcon();
    }, [iconPath, props.label]);

    const validateImage = (file: File): Promise<boolean> => {
        return new Promise((resolve) => {
            // Check file type
            if (!file.type.startsWith('image/png')) {
                setError('Please upload a PNG image file.');
                resolve(false);
                return;
            }

            // Check file size (1MB limit)
            if (file.size > 1024 * 1024) { // 1MB limit
                setError('File size too large. Please upload an image under 1MB.');
                resolve(false);
                return;
            }

            // Check image dimensions
            const img = new Image();
            img.onload = () => {
                if (img.width < 150 || img.height < 150 || img.width > 300 || img.height > 300) {
                    setError('Image must be between 150x150 and 300x300 pixels, and should be 192x192 pixels.');
                    resolve(false);
                } else if (img.width === img.height) {
                    setError(null);
                    resolve(true);
                } else {
                    setError('Image must be square (width and height must be equal).');
                    resolve(false);
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

        // Upload file to server
        const formData = new FormData();
        formData.append('icon', file);
        formData.append('iconType', iconType);

        try {
            const response = await fetch('/plugins/com.mattermost.plugin-msteams-devsecops/icons/upload',
                Client4.getOptions({
                    method: 'POST',
                    body: formData,
                }),
            );

            if (!response.ok) {
                const errorText = await response.text();
                setError(errorText || 'Failed to upload icon');
                setIsUploading(false);
                return;
            }

            const result = await response.json();

            // Create preview from uploaded file
            const reader = new FileReader();
            reader.onload = (event) => {
                const dataUrl = event.target?.result as string;
                setPreview(dataUrl);
                setIsUploading(false);

                // Update configuration with the icon path
                props.onChange(props.id, result.iconPath);

                // Dispatch custom event for real-time validation
                window.dispatchEvent(new CustomEvent(EVENT_APP_INPUT_CHANGE, {
                    detail: {
                        id: props.id,
                        value: result.iconPath,
                    },
                }));
            };
            reader.readAsDataURL(file);
        } catch (err) {
            setError('Failed to upload icon');
            setIsUploading(false);
        }
    };

    const handleUploadClick = () => {
        fileInputRef.current?.click();
    };

    const handleRemove = async () => {
        setIsUploading(true);
        setError(null);

        try {
            const response = await fetch(`/plugins/com.mattermost.plugin-msteams-devsecops/icons/${iconType}`,
                Client4.getOptions({
                    method: 'DELETE',
                }),
            );

            if (!response.ok) {
                const errorText = await response.text();
                setError(errorText || 'Failed to remove icon');
                setIsUploading(false);
                return;
            }

            // Clear the preview and reset to default
            setPreview(null);
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

            setIsUploading(false);
        } catch (err) {
            setError('Failed to remove icon');
            setIsUploading(false);
        }
    };

    // Show preview if we have one, otherwise show the fetched icon (which could be custom or default)
    const iconToShow = preview || defaultIcon;

    // An icon is custom if we have a preview (just uploaded) or if the props.value indicates a custom path
    const isCustomIcon = Boolean(preview) || Boolean(props.value && props.value.includes('/icons/'));

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
                                    width: '96px',
                                    height: '96px',
                                    border: '1px solid #ddd',
                                    borderRadius: '4px',
                                    objectFit: 'cover',
                                }}
                            />
                            <div style={{display: 'flex', flexDirection: 'column', gap: '5px'}}>
                                {isCustomIcon ? (
                                    <button
                                        type='button'
                                        className='btn btn-sm btn-danger'
                                        onClick={handleRemove}
                                        disabled={props.disabled || isUploading}
                                    >
                                        {isUploading ? 'Removing...' : 'Remove'}
                                    </button>
                                ) : (
                                    <button
                                        type='button'
                                        className='btn btn-sm btn-primary'
                                        onClick={handleUploadClick}
                                        disabled={props.disabled || isUploading}
                                    >
                                        {isUploading ? 'Uploading...' : `Upload ${props.label} Icon`}
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
                    {'Must be a square PNG image, 150x150 to 300x300 pixels (192x192 recommended), under 1MB'}
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
