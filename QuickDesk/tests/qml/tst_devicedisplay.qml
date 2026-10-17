// Copyright 2026 QuickDesk Authors

import QtQuick
import QtTest
import "../../qml/quickdeskcomponent/DeviceDisplay.js" as DeviceDisplay

TestCase {
    name: "DeviceDisplay"

    function test_current_device_matches_only_nonempty_ids() {
        compare(DeviceDisplay.isLocalDevice("123456789", "123456789"), true)
        compare(DeviceDisplay.isLocalDevice("123456789", "987654321"), false)
        compare(DeviceDisplay.isLocalDevice("", ""), false)
        compare(DeviceDisplay.isLocalDevice("123456789", ""), false)
    }

    function test_device_label_marks_current_device_and_keeps_name() {
        const device = {
            device_id: "123456789",
            remark: "Office PC"
        }

        compare(DeviceDisplay.deviceLabel(device, "123456789", "Device", "This device"),
                "Office PC (123456789) - This device")
        compare(DeviceDisplay.deviceLabel(device, "987654321", "Device", "This device"),
                "Office PC (123456789)")
    }
}
