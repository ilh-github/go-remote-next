pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QuickDesk 1.0

import "../component"

Item {
    id: root

    required property var mainController

    signal showToast(string message, int toastType)
    signal loginRequested()

    property var manager: mainController && mainController.notificationManager
                          ? mainController.notificationManager : null
    property var cloudManager: mainController && mainController.cloudDeviceManager
                               ? mainController.cloudDeviceManager : null
    property bool isLoggedIn: mainController && mainController.authManager
                              ? mainController.authManager.isLoggedIn : false
    property int editingChannelId: 0
    property var draftEvents: []
    property int deleteChannelId: 0
    property string deleteChannelName: ""
    property int deliveryChannelId: 0
    property string selectedDeviceId: ""
    property bool quietSnoozeEnabled: false
    property date quietSnoozeDate: new Date(Date.now() + 3600000)
    property bool offlineSnoozeEnabled: false
    property date offlineSnoozeDate: new Date(Date.now() + 3600000)
    property bool metricSnoozeEnabled: false
    property date metricSnoozeDate: new Date(Date.now() + 3600000)
    property var quietWindows: []
    property var offlineWindows: []
    property bool refreshPending: false

    function eventLabel(event) {
        if (event === "device.access.denied") return qsTr("Access denied")
        if (event === "device.access.granted") return qsTr("Access granted")
        if (event === "device.access.session") return qsTr("Session started")
        if (event === "device.online.changed") return qsTr("Online status")
        if (event === "device.offline.sustained") return qsTr("Sustained offline")
        if (event === "device.metrics.updated") return qsTr("Host metrics")
        if (event === "device.metric.alert") return qsTr("Metric alert")
        if (event === "device.bound") return qsTr("Device bound")
        if (event === "device.unbound") return qsTr("Device unbound")
        if (event === "device.access_approval.requested") return qsTr("Approval requested")
        if (event === "device.access_approval.decided") return qsTr("Approval decided")
        if (event === "scheduled_task.failed") return qsTr("Scheduled task failed")
        return event || qsTr("Unknown event")
    }

    function deviceLabel(device) {
        return device.remark || device.device_name || device.device_id || qsTr("Device")
    }

    function formatTime(value) {
        if (!value) return "-"
        var parsed = new Date(value)
        return isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString()
    }

    function dateFrom(value, fallback) {
        if (!value) return fallback
        var parsed = new Date(value)
        return isNaN(parsed.getTime()) ? fallback : parsed
    }

    function cloneList(value) {
        return JSON.parse(JSON.stringify(value || []))
    }

    function numericOr(value, fallback) {
        return value === undefined || value === null || value === ""
               ? fallback : Number(value)
    }

    function percentText(value) {
        return Number(value || 0).toFixed(1)
    }

    function hasDraftEvent(event) {
        return root.draftEvents.indexOf(event) >= 0
    }

    function toggleDraftEvent(event, checked) {
        var next = root.draftEvents.slice()
        var position = next.indexOf(event)
        if (checked && position < 0) next.push(event)
        if (!checked && position >= 0) next.splice(position, 1)
        root.draftEvents = next
    }

    function openCreateChannel() {
        root.editingChannelId = 0
        channelNameField.text = ""
        channelUrlField.text = ""
        channelSecretField.text = ""
        channelEnabledSwitch.checked = true
        root.draftEvents = root.manager
                           ? root.manager.supportedEvents.slice() : []
        channelDialog.title = qsTr("Create notification channel")
        channelDialog.show()
    }

    function openEditChannel(channel) {
        root.editingChannelId = Number(channel.id || 0)
        channelNameField.text = channel.name || ""
        channelUrlField.text = channel.url || ""
        channelSecretField.text = ""
        channelEnabledSwitch.checked = !!channel.enabled
        root.draftEvents = cloneList(channel.events)
        channelDialog.title = qsTr("Edit notification channel")
        channelDialog.show()
    }

    function saveChannel() {
        if (!root.manager) return
        if (channelUrlField.text.trim() === "" || root.draftEvents.length === 0) {
            root.showToast(qsTr("Enter a webhook URL and select at least one event"),
                           QDToast.Type.Warning)
            return
        }
        if (root.editingChannelId > 0) {
            root.manager.updateChannel(
                root.editingChannelId,
                channelNameField.text,
                channelUrlField.text,
                channelSecretField.text,
                root.draftEvents,
                channelEnabledSwitch.checked)
        } else {
            root.manager.createChannel(
                channelNameField.text,
                channelUrlField.text,
                channelSecretField.text,
                root.draftEvents,
                channelEnabledSwitch.checked)
        }
    }

    function toggleChannel(channel) {
        if (!root.manager) return
        root.manager.updateChannel(
            Number(channel.id),
            channel.name || "",
            channel.url || "",
            "",
            channel.events || [],
            !channel.enabled)
    }

    function requestDeleteChannel(channel) {
        root.deleteChannelId = Number(channel.id || 0)
        root.deleteChannelName = channel.name || channel.url || ""
        deleteChannelDialog.show()
    }

    function deleteChannel() {
        if (!root.manager || root.deleteChannelId <= 0) return
        root.manager.deleteChannel(root.deleteChannelId)
    }

    function openDeliveries(channel) {
        root.deliveryChannelId = Number(channel.id || 0)
        root.manager.loadDeliveries(root.deliveryChannelId)
        Qt.callLater(function() {
            deliverySection.forceActiveFocus()
        })
    }

    function closeDeliveries() {
        root.deliveryChannelId = 0
    }

    function deliveryChannelName() {
        if (!root.manager || root.deliveryChannelId <= 0) return ""
        for (var i = 0; i < root.manager.channels.length; ++i) {
            var channel = root.manager.channels[i]
            if (Number(channel.id) === root.deliveryChannelId)
                return channel.name || channel.url || ""
        }
        return ""
    }

    function applyQuietHours() {
        if (!root.manager) return
        var value = root.manager.quietHours || {}
        root.quietSnoozeEnabled = !!value.snoozed_until
        root.quietSnoozeDate = dateFrom(
            value.snoozed_until, new Date(Date.now() + 3600000))
        quietSnoozeSwitch.checked = root.quietSnoozeEnabled
        quietSnoozePicker.selectedDate = root.quietSnoozeDate
        quietTimezoneField.text = value.timezone || ""
        root.quietWindows = cloneList(value.quiet_hours)
    }

    function saveQuietHours() {
        if (!root.manager) return
        root.manager.saveQuietHours(
            root.quietSnoozeEnabled
                ? root.quietSnoozeDate.toISOString() : "",
            root.quietWindows,
            quietTimezoneField.text)
    }

    function clearQuietHours() {
        if (!root.manager) return
        root.manager.saveQuietHours("", [], "")
    }

    function applyOfflineAlert() {
        if (!root.manager) return
        var value = root.manager.offlineAlert || {}
        offlineEnabledSwitch.checked = !!value.enabled
        offlineAfterField.text = String(Number(value.after_minutes || 10))
        offlineRepeatField.text = String(Number(value.repeat_minutes || 0))
        root.offlineSnoozeEnabled = !!value.snoozed_until
        root.offlineSnoozeDate = dateFrom(
            value.snoozed_until, new Date(Date.now() + 3600000))
        offlineSnoozeSwitch.checked = root.offlineSnoozeEnabled
        offlineSnoozePicker.selectedDate = root.offlineSnoozeDate
        offlineTimezoneField.text = value.maintenance_timezone || ""
        root.offlineWindows = cloneList(value.maintenance_windows)
    }

    function saveOfflineAlert() {
        if (!root.manager || root.selectedDeviceId === "") return
        root.manager.saveOfflineAlert(
            offlineEnabledSwitch.checked,
            Number(offlineAfterField.text || 10),
            Number(offlineRepeatField.text || 0),
            root.offlineSnoozeEnabled
                ? root.offlineSnoozeDate.toISOString() : "",
            root.offlineWindows,
            offlineTimezoneField.text)
    }

    function removeOfflineAlert() {
        if (!root.manager || root.selectedDeviceId === "") return
        root.manager.removeOfflineAlert()
    }

    function applyMetricAlert() {
        if (!root.manager) return
        var value = root.manager.metricAlert || {}
        metricEnabledSwitch.checked = !!value.enabled
        metricCpuField.text = String(root.numericOr(value.cpu_percent, 80))
        metricMemoryField.text = String(root.numericOr(value.memory_percent, 80))
        metricDiskField.text = String(root.numericOr(value.disk_percent, 85))
        metricSustainedField.text =
            String(root.numericOr(value.sustained_minutes, 5))
        metricRepeatField.text = String(root.numericOr(value.repeat_minutes, 0))
        root.metricSnoozeEnabled = !!value.snoozed_until
        root.metricSnoozeDate = dateFrom(
            value.snoozed_until, new Date(Date.now() + 3600000))
        metricSnoozeSwitch.checked = root.metricSnoozeEnabled
        metricSnoozePicker.selectedDate = root.metricSnoozeDate
    }

    function saveMetricAlert() {
        if (!root.manager || root.selectedDeviceId === "") return
        root.manager.saveMetricAlert(
            metricEnabledSwitch.checked,
            Number(metricCpuField.text || 80),
            Number(metricMemoryField.text || 80),
            Number(metricDiskField.text || 85),
            Number(metricSustainedField.text || 5),
            Number(metricRepeatField.text || 0),
            root.metricSnoozeEnabled
                ? root.metricSnoozeDate.toISOString() : "")
    }

    function removeMetricAlert() {
        if (!root.manager || root.selectedDeviceId === "") return
        root.manager.removeMetricAlert()
    }

    function refreshDeviceAlerts() {
        root.refreshOfflineAlert()
        if (root.manager && root.selectedDeviceId !== "")
            root.manager.loadMetricAlert(root.selectedDeviceId)
    }

    function rebuildDevices() {
        deviceModel.clear()
        var devices = root.cloudManager ? root.cloudManager.myDevices : []
        for (var i = 0; i < devices.length; ++i) {
            deviceModel.append({
                value: devices[i].device_id || "",
                text: root.deviceLabel(devices[i])
            })
        }
        if (root.selectedDeviceId === "" && deviceModel.count > 0) {
            root.selectedDeviceId = deviceModel.get(0).value
            deviceCombo.currentIndex = 0
            Qt.callLater(root.refreshDeviceAlerts)
        } else if (root.selectedDeviceId !== "") {
            var selectedIndex = deviceCombo.indexOfValue(root.selectedDeviceId)
            if (selectedIndex >= 0) deviceCombo.currentIndex = selectedIndex
        }
    }

    function refreshOfflineAlert() {
        if (!root.manager || root.selectedDeviceId === "") return
        root.manager.loadOfflineAlert(root.selectedDeviceId)
    }

    function refreshAll() {
        if (!root.manager || !root.isLoggedIn) return
        root.manager.refreshChannels()
        root.manager.loadQuietHours()
        root.refreshDeviceAlerts()
        if (root.deliveryChannelId > 0)
            root.manager.loadDeliveries(root.deliveryChannelId)
    }

    function scheduleRealtimeRefresh() {
        if (!root.manager || !root.isLoggedIn) return
        if (root.manager.busy) {
            root.refreshPending = true
            return
        }
        realtimeRefreshTimer.restart()
    }

    Connections {
        target: root.mainController && root.mainController.authManager
                ? root.mainController.authManager : null

        function onLoginStateChanged() {
            if (root.isLoggedIn) Qt.callLater(root.refreshAll)
        }
    }

    Connections {
        target: root.cloudManager

        function onMyDevicesChanged() {
            root.rebuildDevices()
        }

        function onSyncMessage(message) {
            if (root.manager) root.manager.handleRealtimeEvent(message)
        }
    }

    Connections {
        target: root.manager

        function onRequestFailed(message) {
            root.showToast(message, QDToast.Type.Error)
        }

        function onChannelCreated(channelId) {
            channelDialog.hide()
            root.showToast(qsTr("Notification channel created"),
                           QDToast.Type.Success)
        }

        function onChannelUpdated(channelId) {
            channelDialog.hide()
            root.showToast(qsTr("Notification channel updated"),
                           QDToast.Type.Success)
        }

        function onChannelDeleted(channelId) {
            deleteChannelDialog.hide()
            if (root.deliveryChannelId === Number(channelId))
                root.closeDeliveries()
            root.showToast(qsTr("Notification channel deleted"),
                           QDToast.Type.Success)
        }

        function onChannelTestQueued(channelId) {
            root.showToast(qsTr("Test notification queued"),
                           QDToast.Type.Success)
        }

        function onQuietHoursChanged() {
            root.applyQuietHours()
        }

        function onOfflineAlertChanged() {
            root.applyOfflineAlert()
        }

        function onMetricAlertChanged() {
            root.applyMetricAlert()
        }

        function onLastRequeuedCountChanged() {
            if (root.manager && root.manager.lastRequeuedCount > 0) {
                root.showToast(qsTr("%1 failed deliveries requeued")
                               .arg(root.manager.lastRequeuedCount),
                               QDToast.Type.Success)
            }
        }

        function onRealtimeEventReceived(type, deviceId) {
            if (type === "device.offline.sustained") {
                root.showToast(qsTr("A device has stayed offline past its alert threshold"),
                               QDToast.Type.Warning)
            } else if (type === "device.metric.alert") {
                root.showToast(qsTr("A host metric stayed above its alert threshold"),
                               QDToast.Type.Warning)
            }
        }

        function onRealtimeRefreshRequested() {
            root.scheduleRealtimeRefresh()
        }

        function onBusyChanged() {
            if (root.manager && !root.manager.busy && root.refreshPending) {
                root.scheduleRealtimeRefresh()
            }
        }
    }

    Timer {
        id: realtimeRefreshTimer
        interval: 200
        repeat: false
        onTriggered: {
            root.refreshPending = false
            root.refreshAll()
        }
    }

    Timer {
        interval: 60000
        repeat: true
        running: root.isLoggedIn && root.selectedDeviceId !== ""
        onTriggered: {
            if (root.manager && !root.manager.busy)
                root.manager.loadMetricAlert(root.selectedDeviceId)
        }
    }

    Component.onCompleted: {
        if (root.isLoggedIn) Qt.callLater(root.refreshAll)
        Qt.callLater(root.rebuildDevices)
    }

    ListModel {
        id: deviceModel
    }

    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.mailGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to manage notifications")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.textSecondary
            }

            QDButton {
                text: qsTr("Login")
                buttonType: QDButton.Type.Primary
                Layout.alignment: Qt.AlignHCenter
                onClicked: root.loginRequested()
            }
        }
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: Theme.spacingLarge
        visible: root.isLoggedIn
        spacing: Theme.spacingMedium

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingMedium

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 2

                Text {
                    text: qsTr("Notifications")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                Text {
                    text: qsTr("Webhook channels, quiet hours, offline and host metric alerts.")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    wrapMode: Text.WordWrap
                }
            }

            QDButton {
                text: qsTr("Retry all failed")
                iconText: FluentIconGlyph.syncGlyph
                buttonType: QDButton.Type.Secondary
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.manager.retryFailedDeliveriesForUser()
            }

            QDButton {
                text: qsTr("Refresh")
                iconText: FluentIconGlyph.syncGlyph
                buttonType: QDButton.Type.Secondary
                loading: root.manager ? root.manager.busy : false
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.refreshAll()
            }

            QDButton {
                text: qsTr("Create channel")
                iconText: FluentIconGlyph.addGlyph
                buttonType: QDButton.Type.Primary
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.openCreateChannel()
            }
        }

        Rectangle {
            Layout.fillWidth: true
            visible: root.manager && root.manager.lastError.length > 0
            Layout.preferredHeight: errorText.implicitHeight + Theme.spacingMedium * 2
            color: Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.08)
            radius: Theme.radiusSmall

            Text {
                id: errorText
                anchors.fill: parent
                anchors.margins: Theme.spacingMedium
                text: root.manager ? root.manager.lastError : ""
                color: Theme.error
                wrapMode: Text.WordWrap
                font.pixelSize: Theme.fontSizeSmall
            }
        }

        Flickable {
            Layout.fillWidth: true
            Layout.fillHeight: true
            contentWidth: width
            contentHeight: contentColumn.implicitHeight
            clip: true

            ScrollBar.vertical: QDScrollBar {}

            ColumnLayout {
                id: contentColumn
                width: parent.width
                spacing: Theme.spacingMedium

                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: channelColumn.implicitHeight
                                            + Theme.spacingLarge * 2
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    ColumnLayout {
                        id: channelColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium

                        RowLayout {
                            Layout.fillWidth: true

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 2

                                Text {
                                    text: qsTr("Webhook channels")
                                    font.pixelSize: Theme.fontSizeLarge
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("Subscribe to device access, presence, approvals and scheduled-task failures.")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                }
                            }

                            QDChip {
                                text: qsTr("%1 channels")
                                      .arg(root.manager ? root.manager.channels.length : 0)
                                chipType: QDChip.Type.Info
                                outlined: true
                            }
                        }

                        Text {
                            visible: !root.manager || root.manager.channels.length === 0
                            Layout.fillWidth: true
                            text: root.manager && root.manager.busy
                                  ? qsTr("Loading notification channels...")
                                  : qsTr("No notification channels yet.")
                            color: Theme.textSecondary
                            font.pixelSize: Theme.fontSizeMedium
                        }

                        Repeater {
                            model: root.manager ? root.manager.channels : []

                            Rectangle {
                                id: channelCard
                                required property var modelData

                                Layout.fillWidth: true
                                Layout.preferredHeight: channelCardContent.implicitHeight
                                                        + Theme.spacingLarge * 2
                                color: Theme.surfaceVariant
                                radius: Theme.radiusMedium
                                border.width: Theme.borderWidthThin
                                border.color: Theme.border

                                ColumnLayout {
                                    id: channelCardContent
                                    anchors.fill: parent
                                    anchors.margins: Theme.spacingLarge
                                    spacing: Theme.spacingSmall

                                    RowLayout {
                                        Layout.fillWidth: true
                                        spacing: Theme.spacingSmall

                                        Text {
                                            text: channelCard.modelData.name
                                                  || qsTr("Unnamed channel")
                                            font.pixelSize: Theme.fontSizeMedium
                                            font.weight: Font.DemiBold
                                            color: Theme.text
                                        }

                                        QDChip {
                                            text: channelCard.modelData.enabled
                                                  ? qsTr("Enabled") : qsTr("Paused")
                                            chipType: channelCard.modelData.enabled
                                                      ? QDChip.Type.Success
                                                      : QDChip.Type.Default
                                        }

                                        QDChip {
                                            visible: Number(channelCard.modelData.last_status || 0) > 0
                                            text: "HTTP " + Number(channelCard.modelData.last_status || 0)
                                            chipType: Number(channelCard.modelData.last_status || 0) >= 200
                                                      && Number(channelCard.modelData.last_status || 0) < 300
                                                      ? QDChip.Type.Success
                                                      : QDChip.Type.Error
                                            outlined: true
                                        }

                                        Item { Layout.fillWidth: true }

                                        QDButton {
                                            text: channelCard.modelData.enabled
                                                  ? qsTr("Pause") : qsTr("Enable")
                                            buttonType: QDButton.Type.Secondary
                                            onClicked: root.toggleChannel(channelCard.modelData)
                                        }

                                        QDButton {
                                            text: qsTr("Test")
                                            buttonType: QDButton.Type.Secondary
                                            onClicked: root.manager.testChannel(
                                                           Number(channelCard.modelData.id))
                                        }

                                        QDButton {
                                            text: qsTr("Deliveries")
                                            buttonType: QDButton.Type.Secondary
                                            onClicked: root.openDeliveries(channelCard.modelData)
                                        }

                                        QDButton {
                                            text: qsTr("Edit")
                                            buttonType: QDButton.Type.Secondary
                                            onClicked: root.openEditChannel(channelCard.modelData)
                                        }

                                        QDButton {
                                            text: qsTr("Delete")
                                            iconText: FluentIconGlyph.deleteGlyph
                                            buttonType: QDButton.Type.Danger
                                            onClicked: root.requestDeleteChannel(channelCard.modelData)
                                        }
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: channelCard.modelData.url || ""
                                        color: Theme.textSecondary
                                        font.pixelSize: Theme.fontSizeSmall
                                        elide: Text.ElideRight
                                    }

                                    Flow {
                                        Layout.fillWidth: true
                                        spacing: Theme.spacingSmall

                                        Repeater {
                                            model: channelCard.modelData.events || []

                                            QDChip {
                                                required property var modelData
                                                text: root.eventLabel(modelData)
                                                chipType: QDChip.Type.Info
                                                outlined: true
                                            }
                                        }
                                    }

                                    Text {
                                        Layout.fillWidth: true
                                        text: qsTr("Last triggered: %1")
                                              .arg(root.formatTime(
                                                   channelCard.modelData.last_triggered))
                                        color: Theme.textSecondary
                                        font.pixelSize: Theme.fontSizeSmall
                                    }
                                }
                            }
                        }

                        Text {
                            Layout.fillWidth: true
                            text: qsTr("Each delivery is signed with %1 using %2.")
                                  .arg(root.manager
                                       ? root.manager.signatureHeader
                                       : "X-Webhook-Signature")
                                  .arg(root.manager
                                       ? root.manager.signatureScheme
                                       : "sha256")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                            wrapMode: Text.WordWrap
                        }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: quietColumn.implicitHeight
                                            + Theme.spacingLarge * 2
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    ColumnLayout {
                        id: quietColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium

                        RowLayout {
                            Layout.fillWidth: true

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 2

                                Text {
                                    text: qsTr("Quiet hours")
                                    font.pixelSize: Theme.fontSizeLarge
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("Pause every channel during a snooze or recurring window.")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                }
                            }

                            QDChip {
                                text: root.manager && root.manager.quietHours.active === true
                                      ? qsTr("Active") : qsTr("Inactive")
                                chipType: root.manager && root.manager.quietHours.active === true
                                          ? QDChip.Type.Warning : QDChip.Type.Success
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            QDSwitch {
                                id: quietSnoozeSwitch
                                text: qsTr("Snooze until")
                                checked: root.quietSnoozeEnabled
                                onToggled: root.quietSnoozeEnabled = checked
                            }

                            QDDateTimePicker {
                                id: quietSnoozePicker
                                Layout.preferredWidth: 260
                                visible: root.quietSnoozeEnabled
                                showTime: true
                                selectedDate: root.quietSnoozeDate
                                onSelectedDateChanged: root.quietSnoozeDate = selectedDate
                            }

                            QDTextField {
                                id: quietTimezoneField
                                Layout.fillWidth: true
                                placeholderText: qsTr("IANA timezone, for example Asia/Shanghai")
                            }
                        }

                        NotificationWindowEditor {
                            Layout.fillWidth: true
                            windows: root.quietWindows
                            emptyText: qsTr("No recurring quiet windows")
                            onWindowsEdited: function(value) {
                                root.quietWindows = value
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true

                            QDButton {
                                text: qsTr("Save quiet hours")
                                buttonType: QDButton.Type.Primary
                                enabled: !!root.manager && !root.manager.busy
                                onClicked: root.saveQuietHours()
                            }

                            QDButton {
                                text: qsTr("Clear")
                                buttonType: QDButton.Type.Secondary
                                enabled: !!root.manager && !root.manager.busy
                                onClicked: root.clearQuietHours()
                            }

                            Item { Layout.fillWidth: true }

                            Text {
                                visible: root.manager
                                         && !!root.manager.quietHours.active_until
                                text: qsTr("Active until %1")
                                      .arg(root.formatTime(
                                           root.manager
                                           ? root.manager.quietHours.active_until : ""))
                                color: Theme.warning
                                font.pixelSize: Theme.fontSizeSmall
                            }
                        }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: offlineColumn.implicitHeight
                                            + Theme.spacingLarge * 2
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    ColumnLayout {
                        id: offlineColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium

                        RowLayout {
                            Layout.fillWidth: true

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 2

                                Text {
                                    text: qsTr("Sustained offline alerts")
                                    font.pixelSize: Theme.fontSizeLarge
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("Alert through the channels above after a device stays offline past the threshold.")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                }
                            }

                            QDChip {
                                visible: root.manager
                                         && root.manager.offlineAlert.in_maintenance_window === true
                                text: qsTr("In maintenance")
                                chipType: QDChip.Type.Warning
                            }
                        }

                        QDComboBox {
                            id: deviceCombo
                            Layout.preferredWidth: 320
                            model: deviceModel
                            textRole: "text"
                            valueRole: "value"
                            onActivated: {
                                root.selectedDeviceId = selectedValue
                                root.refreshDeviceAlerts()
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            QDSwitch {
                                id: offlineEnabledSwitch
                                text: qsTr("Enabled")
                            }

                            QDTextField {
                                id: offlineAfterField
                                Layout.preferredWidth: 190
                                placeholderText: qsTr("After minutes")
                            }

                            QDTextField {
                                id: offlineRepeatField
                                Layout.preferredWidth: 190
                                placeholderText: qsTr("Repeat minutes (0 = once)")
                            }

                            QDSwitch {
                                id: offlineSnoozeSwitch
                                text: qsTr("Snooze until")
                                checked: root.offlineSnoozeEnabled
                                onToggled: root.offlineSnoozeEnabled = checked
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            QDDateTimePicker {
                                id: offlineSnoozePicker
                                Layout.preferredWidth: 260
                                visible: root.offlineSnoozeEnabled
                                showTime: true
                                selectedDate: root.offlineSnoozeDate
                                onSelectedDateChanged: root.offlineSnoozeDate = selectedDate
                            }

                            QDTextField {
                                id: offlineTimezoneField
                                Layout.fillWidth: true
                                placeholderText: qsTr("Maintenance timezone")
                            }
                        }

                        NotificationWindowEditor {
                            Layout.fillWidth: true
                            windows: root.offlineWindows
                            emptyText: qsTr("No maintenance windows")
                            onWindowsEdited: function(value) {
                                root.offlineWindows = value
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true

                            QDButton {
                                text: qsTr("Save offline alert")
                                buttonType: QDButton.Type.Primary
                                enabled: root.selectedDeviceId !== ""
                                         && !!root.manager && !root.manager.busy
                                onClicked: root.saveOfflineAlert()
                            }

                            QDButton {
                                text: qsTr("Reset")
                                buttonType: QDButton.Type.Secondary
                                enabled: root.selectedDeviceId !== ""
                                         && !!root.manager && !root.manager.busy
                                onClicked: root.removeOfflineAlert()
                            }

                            Item { Layout.fillWidth: true }

                            Text {
                                visible: root.manager
                                         && !!root.manager.offlineAlert.offline_since
                                text: qsTr("Offline since %1")
                                      .arg(root.formatTime(
                                           root.manager
                                           ? root.manager.offlineAlert.offline_since : ""))
                                color: Theme.warning
                                font.pixelSize: Theme.fontSizeSmall
                            }
                        }
                    }
                }

                Rectangle {
                    Layout.fillWidth: true
                    Layout.preferredHeight: metricColumn.implicitHeight
                                            + Theme.spacingLarge * 2
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    ColumnLayout {
                        id: metricColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium

                        RowLayout {
                            Layout.fillWidth: true

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 2

                                Text {
                                    text: qsTr("Host metric alerts")
                                    font.pixelSize: Theme.fontSizeLarge
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("Alert when CPU, memory or disk usage stays above a threshold.")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                    wrapMode: Text.WordWrap
                                }
                            }

                            QDChip {
                                text: root.manager
                                      && root.manager.metricCurrentValue.captured_at
                                      ? qsTr("Live") : qsTr("Waiting")
                                chipType: root.manager
                                          && root.manager.metricCurrentValue.captured_at
                                          ? QDChip.Type.Success
                                          : QDChip.Type.Default
                                outlined: true
                            }
                        }

                        Flow {
                            Layout.fillWidth: true
                            spacing: Theme.spacingSmall

                            QDChip {
                                text: qsTr("CPU %1%")
                                      .arg(root.percentText(
                                           root.manager
                                           ? root.manager.metricCurrentValue.cpu_percent : 0))
                                chipType: QDChip.Type.Info
                                outlined: true
                            }

                            QDChip {
                                text: qsTr("Memory %1%")
                                      .arg(root.percentText(
                                           root.manager
                                           ? root.manager.metricCurrentValue.memory_percent : 0))
                                chipType: QDChip.Type.Info
                                outlined: true
                            }

                            QDChip {
                                text: qsTr("Disk %1% (%2)")
                                      .arg(root.percentText(
                                           root.manager
                                           ? root.manager.metricCurrentValue.disk_percent : 0))
                                      .arg(root.manager
                                           ? root.manager.metricCurrentValue.disk_mount || "-"
                                           : "-")
                                chipType: QDChip.Type.Info
                                outlined: true
                            }
                        }

                        Text {
                            visible: root.manager
                                     && !!root.manager.metricCurrentValue.captured_at
                            Layout.fillWidth: true
                            text: qsTr("Captured %1")
                                  .arg(root.formatTime(
                                       root.manager
                                       ? root.manager.metricCurrentValue.captured_at : ""))
                            color: Theme.textSecondary
                            font.pixelSize: Theme.fontSizeSmall
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            QDSwitch {
                                id: metricEnabledSwitch
                                text: qsTr("Enabled")
                            }

                            QDTextField {
                                id: metricCpuField
                                Layout.preferredWidth: 150
                                placeholderText: qsTr("CPU %")
                            }

                            QDTextField {
                                id: metricMemoryField
                                Layout.preferredWidth: 150
                                placeholderText: qsTr("Memory %")
                            }

                            QDTextField {
                                id: metricDiskField
                                Layout.preferredWidth: 150
                                placeholderText: qsTr("Disk %")
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            QDTextField {
                                id: metricSustainedField
                                Layout.preferredWidth: 220
                                placeholderText: qsTr("Sustained minutes")
                            }

                            QDTextField {
                                id: metricRepeatField
                                Layout.preferredWidth: 220
                                placeholderText: qsTr("Repeat minutes (0 = once)")
                            }

                            QDSwitch {
                                id: metricSnoozeSwitch
                                text: qsTr("Snooze until")
                                checked: root.metricSnoozeEnabled
                                onToggled: root.metricSnoozeEnabled = checked
                            }

                            QDDateTimePicker {
                                id: metricSnoozePicker
                                Layout.preferredWidth: 260
                                visible: root.metricSnoozeEnabled
                                showTime: true
                                selectedDate: root.metricSnoozeDate
                                onSelectedDateChanged: root.metricSnoozeDate = selectedDate
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true

                            QDButton {
                                text: qsTr("Save metric alert")
                                buttonType: QDButton.Type.Primary
                                enabled: root.selectedDeviceId !== ""
                                         && !!root.manager && !root.manager.busy
                                onClicked: root.saveMetricAlert()
                            }

                            QDButton {
                                text: qsTr("Reset")
                                buttonType: QDButton.Type.Secondary
                                enabled: root.selectedDeviceId !== ""
                                         && !!root.manager && !root.manager.busy
                                onClicked: root.removeMetricAlert()
                            }

                            Item { Layout.fillWidth: true }

                            Text {
                                visible: root.manager
                                         && !!root.manager.metricAlert.snoozed_until
                                text: qsTr("Snoozed until %1")
                                      .arg(root.formatTime(
                                           root.manager
                                           ? root.manager.metricAlert.snoozed_until : ""))
                                color: Theme.warning
                                font.pixelSize: Theme.fontSizeSmall
                            }
                        }
                    }
                }

                Rectangle {
                    id: deliverySection
                    Layout.fillWidth: true
                    visible: root.deliveryChannelId > 0
                    Layout.preferredHeight: deliveryColumn.implicitHeight
                                            + Theme.spacingLarge * 2
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    ColumnLayout {
                        id: deliveryColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingMedium

                        RowLayout {
                            Layout.fillWidth: true

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: 2

                                Text {
                                    text: qsTr("Delivery history")
                                    font.pixelSize: Theme.fontSizeLarge
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("%1 · %2 unacknowledged")
                                          .arg(root.deliveryChannelName())
                                          .arg(root.manager
                                               ? root.manager.unackedDeliveries : 0)
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }
                            }

                            QDButton {
                                text: qsTr("Reload")
                                buttonType: QDButton.Type.Secondary
                                onClicked: root.manager.loadDeliveries(
                                               root.deliveryChannelId)
                            }

                            QDButton {
                                text: qsTr("Retry failed")
                                buttonType: QDButton.Type.Secondary
                                onClicked: root.manager.retryFailedDeliveries(
                                               root.deliveryChannelId)
                            }

                            QDButton {
                                text: qsTr("Close")
                                buttonType: QDButton.Type.Ghost
                                onClicked: root.closeDeliveries()
                            }
                        }

                        Text {
                            visible: !root.manager || root.manager.deliveries.length === 0
                            Layout.fillWidth: true
                            text: root.manager && root.manager.busy
                                  ? qsTr("Loading delivery history...")
                                  : qsTr("No deliveries for this channel.")
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.textSecondary
                        }

                        Repeater {
                            model: root.manager ? root.manager.deliveries : []

                            Rectangle {
                                id: deliveryCard
                                required property var modelData

                                Layout.fillWidth: true
                                Layout.preferredHeight: deliveryRow.implicitHeight
                                                        + Theme.spacingMedium * 2
                                color: Theme.surfaceVariant
                                radius: Theme.radiusMedium
                                border.width: Theme.borderWidthThin
                                border.color: deliveryCard.modelData.status === "failed"
                                              ? Qt.rgba(Theme.error.r, Theme.error.g,
                                                        Theme.error.b, 0.45)
                                              : Theme.border

                                RowLayout {
                                    id: deliveryRow
                                    anchors.fill: parent
                                    anchors.margins: Theme.spacingMedium
                                    spacing: Theme.spacingMedium

                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        spacing: 2

                                        RowLayout {
                                            spacing: Theme.spacingSmall

                                            QDChip {
                                                text: deliveryCard.modelData.status || qsTr("Unknown")
                                                chipType: deliveryCard.modelData.status === "failed"
                                                          ? QDChip.Type.Error
                                                          : deliveryCard.modelData.status === "delivered"
                                                            ? QDChip.Type.Success : QDChip.Type.Warning
                                            }

                                            Text {
                                                text: root.eventLabel(
                                                          deliveryCard.modelData.event_type)
                                                font.pixelSize: Theme.fontSizeMedium
                                                font.weight: Font.DemiBold
                                                color: Theme.text
                                            }
                                        }

                                        Text {
                                            Layout.fillWidth: true
                                            text: qsTr("%1 · attempts %2 · queued %3")
                                                  .arg(deliveryCard.modelData.device_id
                                                       || qsTr("Account event"))
                                                  .arg(Number(deliveryCard.modelData.attempts || 0))
                                                  .arg(root.formatTime(
                                                       deliveryCard.modelData.created_at))
                                            font.pixelSize: Theme.fontSizeSmall
                                            color: Theme.textSecondary
                                            wrapMode: Text.WordWrap
                                        }

                                        Text {
                                            visible: !!deliveryCard.modelData.last_error
                                            Layout.fillWidth: true
                                            text: deliveryCard.modelData.last_error || ""
                                            font.pixelSize: Theme.fontSizeSmall
                                            color: Theme.error
                                            wrapMode: Text.WordWrap
                                        }

                                        Text {
                                            visible: !!deliveryCard.modelData.acked_at
                                            Layout.fillWidth: true
                                            text: qsTr("Acknowledged %1")
                                                  .arg(root.formatTime(
                                                       deliveryCard.modelData.acked_at))
                                            font.pixelSize: Theme.fontSizeSmall
                                            color: Theme.textSecondary
                                        }
                                    }

                                    QDButton {
                                        visible: deliveryCard.modelData.status === "failed"
                                        text: qsTr("Retry")
                                        buttonType: QDButton.Type.Secondary
                                        onClicked: root.manager.retryDelivery(
                                                       root.deliveryChannelId,
                                                       Number(deliveryCard.modelData.id))
                                    }

                                    QDButton {
                                        visible: !deliveryCard.modelData.acked_at
                                                 && (deliveryCard.modelData.status === "delivered"
                                                     || deliveryCard.modelData.status === "failed")
                                        text: qsTr("Acknowledge")
                                        buttonType: QDButton.Type.Secondary
                                        onClicked: root.manager.ackDelivery(
                                                       root.deliveryChannelId,
                                                       Number(deliveryCard.modelData.id))
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    QDDialog {
        id: channelDialog
        title: qsTr("Notification channel")
        dialogWidth: Math.min(760, root.width - 48)
        dialogHeight: Math.min(700, root.height - 48)
        closeOnOverlay: false

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingMedium

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingMedium

                QDTextField {
                    id: channelNameField
                    Layout.fillWidth: true
                    placeholderText: qsTr("Channel name")
                }

                QDSwitch {
                    id: channelEnabledSwitch
                    text: qsTr("Enabled")
                    checked: true
                }
            }

            QDTextField {
                id: channelUrlField
                Layout.fillWidth: true
                placeholderText: "https://example.com/quickdesk/webhook"
            }

            QDTextField {
                id: channelSecretField
                Layout.fillWidth: true
                echoMode: TextInput.Password
                placeholderText: root.editingChannelId > 0
                                 ? qsTr("Leave blank to keep the stored secret")
                                 : qsTr("Optional signing secret")
            }

            Text {
                text: qsTr("Events")
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.DemiBold
                color: Theme.text
            }

            Flow {
                Layout.fillWidth: true
                spacing: Theme.spacingSmall

                Repeater {
                    model: root.manager ? root.manager.supportedEvents : []

                    QDCheckBox {
                        required property var modelData
                        text: root.eventLabel(modelData)
                        checked: root.hasDraftEvent(modelData)
                        onToggled: root.toggleDraftEvent(modelData, checked)
                    }
                }
            }

            Item { Layout.fillHeight: true }

            RowLayout {
                Layout.fillWidth: true

                Item { Layout.fillWidth: true }

                QDButton {
                    text: qsTr("Cancel")
                    buttonType: QDButton.Type.Secondary
                    onClicked: channelDialog.hide()
                }

                QDButton {
                    text: qsTr("Save")
                    buttonType: QDButton.Type.Primary
                    enabled: !!root.manager && !root.manager.busy
                    onClicked: root.saveChannel()
                }
            }
        }
    }

    QDDialog {
        id: deleteChannelDialog
        title: qsTr("Delete notification channel")
        dialogWidth: Math.min(480, root.width - 48)
        dialogHeight: 220

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingLarge

            Text {
                Layout.fillWidth: true
                text: qsTr("Delete “%1”? Its delivery history will no longer be available.")
                      .arg(root.deleteChannelName)
                color: Theme.text
                wrapMode: Text.WordWrap
            }

            Item { Layout.fillHeight: true }

            RowLayout {
                Layout.fillWidth: true

                Item { Layout.fillWidth: true }

                QDButton {
                    text: qsTr("Cancel")
                    buttonType: QDButton.Type.Secondary
                    onClicked: deleteChannelDialog.hide()
                }

                QDButton {
                    text: qsTr("Delete")
                    buttonType: QDButton.Type.Danger
                    onClicked: root.deleteChannel()
                }
            }
        }
    }
}
