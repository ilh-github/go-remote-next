export const FILE_TRANSFER_ERROR_CANCELED = 1;
export const FILE_TRANSFER_ERROR_FAILED = 2;
export const FILE_TRANSFER_ERROR_PROTOCOL = 3;
export const FILE_TRANSFER_ERROR_PERMISSION_DENIED = 4;
export const FILE_TRANSFER_ERROR_OUT_OF_DISK_SPACE = 5;
export const FILE_TRANSFER_ERROR_IO = 6;
export const FILE_TRANSFER_ERROR_NOT_LOGGED_IN = 7;

export function formatFileTransferError(type) {
    const numericType = Number(type);
    switch (numericType) {
        case 0:
            return 'Host error type=0: unspecified file transfer failure';
        case FILE_TRANSFER_ERROR_CANCELED:
            return 'Host canceled the file transfer (type=1)';
        case FILE_TRANSFER_ERROR_FAILED:
            return 'Host failed to save the file (type=2). Check file-transfer permission, the target folder, and disk space.';
        case FILE_TRANSFER_ERROR_PROTOCOL:
            return 'Host rejected the file transfer protocol message (type=3). Reconnect and retry with a compatible QuickDesk version.';
        case FILE_TRANSFER_ERROR_PERMISSION_DENIED:
            return 'Host denied permission to the destination folder (type=4). Check the remote user profile and folder permissions.';
        case FILE_TRANSFER_ERROR_OUT_OF_DISK_SPACE:
            return 'Host ran out of disk space while saving the file (type=5). Free space on the remote drive and retry.';
        case FILE_TRANSFER_ERROR_IO:
            return 'Host could not read or write the transferred file (type=6). Check the destination drive and retry.';
        case FILE_TRANSFER_ERROR_NOT_LOGGED_IN:
            return 'Host has no logged-in interactive desktop session (type=7). Sign in on the remote Windows device and retry.';
        default:
            return `Host error type=${Number.isFinite(numericType) ? numericType : String(type)}`;
    }
}
