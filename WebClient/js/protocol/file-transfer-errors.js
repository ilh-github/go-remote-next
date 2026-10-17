export const FILE_TRANSFER_ERROR_CANCELED = 1;
export const FILE_TRANSFER_ERROR_FAILED = 2;

export function formatFileTransferError(type) {
    const numericType = Number(type);
    switch (numericType) {
        case 0:
            return 'Host error type=0: unknown file transfer failure';
        case FILE_TRANSFER_ERROR_CANCELED:
            return 'Host canceled the file transfer (type=1)';
        case FILE_TRANSFER_ERROR_FAILED:
            return 'Host failed to save the file (type=2). Check file-transfer permission, the target folder, and disk space.';
        default:
            return `Host error type=${Number.isFinite(numericType) ? numericType : String(type)}`;
    }
}
