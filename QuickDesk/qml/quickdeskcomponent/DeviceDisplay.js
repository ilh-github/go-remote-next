// Copyright 2026 QuickDesk Authors

.pragma library

function isLocalDevice(deviceId, currentDeviceId) {
    return !!deviceId && !!currentDeviceId &&
           String(deviceId) === String(currentDeviceId)
}

function deviceLabel(device, currentDeviceId, fallbackName, localSuffix) {
    const row = device || {}
    const deviceId = row.device_id || ""
    const preferredName = row.remark || row.device_name || fallbackName || "Device"
    let label = preferredName + (deviceId ? " (" + deviceId + ")" : "")
    if (isLocalDevice(deviceId, currentDeviceId) && localSuffix) {
        label += " - " + localSuffix
    }
    return label
}
