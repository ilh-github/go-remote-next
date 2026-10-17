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

    property var manager: mainController && mainController.remoteSessionManager
                          ? mainController.remoteSessionManager : null
    property bool isLoggedIn: mainController && mainController.authManager
                              ? mainController.authManager.isLoggedIn : false
    property string statusFilter: "active"
    property string deviceFilter: ""
    property string expandedSessionId: ""
    property int confirmMode: 0
    property string pendingSessionId: ""
    property string pendingDeviceId: ""
    property string pendingSource: ""

    function formatTime(value) {
        if (!value) return "-"
        var parsed = new Date(value)
        return isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString()
    }

    function formatDuration(seconds) {
        var total = Math.max(0, Math.floor(Number(seconds) || 0))
        var hours = Math.floor(total / 3600)
        var minutes = Math.floor((total % 3600) / 60)
        var secs = total % 60
        var pad = function(value) {
            return value < 10 ? "0" + value : String(value)
        }
        return hours > 0
            ? hours + ":" + pad(minutes) + ":" + pad(secs)
            : minutes + ":" + pad(secs)
    }

    function formatSize(bytes) {
        var value = Number(bytes) || 0
        if (value <= 0) return "-"
        if (value < 1024) return value + " B"
        var units = ["KB", "MB", "GB"]
        var scaled = value / 1024
        var unit = 0
        while (scaled >= 1024 && unit < units.length - 1) {
            scaled /= 1024
            unit += 1
        }
        return scaled.toFixed(scaled >= 10 ? 0 : 1) + " " + units[unit]
    }

    function endReasonText(reason) {
        if (reason === "disconnected") return qsTr("Disconnected")
        if (reason === "revoked") return qsTr("Revoked")
        if (reason === "server_restart") return qsTr("Server restart")
        return reason || "-"
    }

    function applyFilters() {
        if (!manager) return
        deviceFilter = deviceField.text.trim()
        expandedSessionId = ""
        manager.refresh(statusFilter, deviceFilter)
        manager.fetchDeniedSources(deviceFilter)
    }

    function requestKick(sessionId, deviceId) {
        confirmMode = 1
        pendingSessionId = sessionId
        pendingDeviceId = deviceId
        actionDialog.title = qsTr("Kick controller")
        actionDialog.message = qsTr("Disconnect the controller from %1?").arg(deviceId)
        actionDialog.show()
    }

    function requestClearDevice() {
        var deviceId = deviceField.text.trim()
        if (!deviceId) {
            showToast(qsTr("Enter a device ID first"), QDToast.Type.Warning)
            return
        }
        confirmMode = 2
        pendingSessionId = ""
        pendingDeviceId = deviceId
        deviceFilter = deviceId
        actionDialog.title = qsTr("Clear device")
        actionDialog.message = qsTr("Disconnect every controller from %1?").arg(deviceId)
        actionDialog.show()
    }

    function requestBanSource(deviceId, source) {
        confirmMode = 3
        pendingSessionId = ""
        pendingDeviceId = deviceId
        pendingSource = source
        actionDialog.title = qsTr("Ban source")
        actionDialog.message = qsTr("Permanently ban %1? It will be refused even with a valid access code.")
                                 .arg(source)
        actionDialog.show()
    }

    function requestUnbanSource(deviceId, source) {
        confirmMode = 4
        pendingSessionId = ""
        pendingDeviceId = deviceId
        pendingSource = source
        actionDialog.title = qsTr("Unban source")
        actionDialog.message = qsTr("Remove the permanent ban on %1?").arg(source)
        actionDialog.show()
    }

    function confirmActionText() {
        if (confirmMode === 1) return qsTr("Kick")
        if (confirmMode === 2) return qsTr("Clear device")
        if (confirmMode === 3) return qsTr("Ban source")
        return qsTr("Unban source")
    }

    Connections {
        target: root.mainController && root.mainController.authManager
                ? root.mainController.authManager : null

        function onLoginStateChanged() {
            if (root.isLoggedIn) {
                Qt.callLater(root.applyFilters)
            } else {
                root.expandedSessionId = ""
            }
        }
    }

    Connections {
        target: root.manager

        function onRequestFailed(message) {
            root.showToast(message, QDToast.Type.Error)
        }

        function onSessionRevoked(sessionId) {
            root.expandedSessionId = ""
            root.showToast(qsTr("Controller disconnected"), QDToast.Type.Success)
            if (root.manager) root.manager.refresh(root.statusFilter, root.deviceFilter)
        }

        function onDeviceSessionsRevoked(deviceId, revoked, failed, truncated) {
            root.expandedSessionId = ""
            var message = qsTr("Disconnected %1 session(s), %2 failed")
                          .arg(revoked).arg(failed)
            if (truncated) {
                message += " " + qsTr("More sessions may remain because the server handled only one batch.")
            }
            root.showToast(message, failed > 0 ? QDToast.Type.Warning : QDToast.Type.Success)
            if (root.manager) root.manager.refresh(root.statusFilter, root.deviceFilter)
        }

        function onSourceBanned(deviceId, source) {
            root.showToast(qsTr("Permanently banned %1").arg(source),
                           QDToast.Type.Success)
            if (root.manager) root.manager.fetchDeniedSources(deviceId)
        }

        function onSourceUnbanned(deviceId, source) {
            root.showToast(qsTr("Unbanned %1").arg(source),
                           QDToast.Type.Success)
        }
    }

    Timer {
        interval: 10000
        repeat: true
        running: root.isLoggedIn && root.statusFilter === "active"
                 && root.manager && !root.manager.busy
        onTriggered: {
            if (root.manager) {
                root.manager.refresh(root.statusFilter, root.deviceFilter)
            }
        }
    }

    Component.onCompleted: {
        if (root.isLoggedIn) {
            Qt.callLater(root.applyFilters)
        }
    }

    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.historyGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to view remote sessions")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.textSecondary
                Layout.alignment: Qt.AlignHCenter
            }

            QDButton {
                text: qsTr("Login")
                highlighted: true
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
                    text: qsTr("Remote Sessions")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                Text {
                    text: qsTr("Who is connected to your devices, and what happened in each session")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    elide: Text.ElideRight
                    Layout.fillWidth: true
                }
            }

            QDButton {
                text: qsTr("Refresh")
                iconText: FluentIconGlyph.refreshGlyph
                buttonType: QDButton.Type.Secondary
                loading: root.manager ? root.manager.busy : false
                onClicked: root.applyFilters()
            }

            QDButton {
                text: qsTr("Clear device")
                iconText: FluentIconGlyph.deleteGlyph
                buttonType: QDButton.Type.Danger
                enabled: deviceField.text.trim().length > 0
                         && root.manager && !root.manager.busy
                onClicked: root.requestClearDevice()
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall

            QDComboBox {
                id: statusCombo
                Layout.preferredWidth: 130
                textRole: "text"
                valueRole: "value"
                model: ListModel {
                    id: statusModel
                }
                currentIndex: 0
                Component.onCompleted: {
                    statusModel.append({ "text": qsTr("Active"), "value": "active" })
                    statusModel.append({ "text": qsTr("Ended"), "value": "ended" })
                    statusModel.append({ "text": qsTr("All"), "value": "all" })
                    currentIndex = 0
                }
                onActivated: {
                    root.statusFilter = selectedValue
                    root.applyFilters()
                }
            }

            QDTextField {
                id: deviceField
                Layout.fillWidth: true
                placeholderText: qsTr("Filter by device ID")
                prefixIcon: FluentIconGlyph.contactInfoGlyph
                onAccepted: root.applyFilters()
            }

            Text {
                text: root.manager
                      ? qsTr("Sessions: %1").arg(root.manager.sessions.length)
                      : qsTr("Sessions: 0")
                font.pixelSize: Theme.fontSizeSmall
                color: Theme.textSecondary
            }
        }

        Rectangle {
            visible: root.deviceFilter.length > 0
            Layout.fillWidth: true
            Layout.preferredHeight: blockedContent.implicitHeight
                                 + Theme.spacingLarge * 2
            radius: Theme.radiusMedium
            color: Theme.surface
            border.width: Theme.borderWidthThin
            border.color: Theme.border

            ColumnLayout {
                id: blockedContent
                anchors.left: parent.left
                anchors.right: parent.right
                anchors.top: parent.top
                anchors.margins: Theme.spacingLarge
                spacing: Theme.spacingSmall

                Text {
                    text: qsTr("Blocked sources")
                    font.weight: Font.DemiBold
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.text
                }

                Text {
                    visible: !root.manager || root.manager.deniedSources.length === 0
                    text: qsTr("No permanently blocked sources")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                Repeater {
                    model: root.manager ? root.manager.deniedSources : []

                    RowLayout {
                        required property var modelData
                        Layout.fillWidth: true
                        spacing: Theme.spacingSmall

                        Text {
                            Layout.fillWidth: true
                            text: modelData
                            font.family: "Menlo"
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.text
                        }

                        QDButton {
                            text: qsTr("Unban")
                            buttonType: QDButton.Type.Secondary
                            implicitWidth: 90
                            onClicked: root.requestUnbanSource(root.deviceFilter,
                                                               modelData)
                        }
                    }
                }
            }
        }

        Item {
            Layout.fillWidth: true
            Layout.fillHeight: true

            ListView {
                id: sessionList
                anchors.fill: parent
                model: root.manager ? root.manager.sessions : []
                spacing: Theme.spacingSmall
                clip: true
                boundsBehavior: Flickable.StopAtBounds
                ScrollBar.vertical: QDScrollBar {}

                delegate: Rectangle {
                    id: sessionCard
                    required property var modelData
                    required property int index

                    property bool expanded: root.expandedSessionId === modelData.session_id

                    width: sessionList.width
                    height: cardContent.implicitHeight + Theme.spacingLarge * 2
                    radius: Theme.radiusMedium
                    color: Theme.surface
                    border.width: Theme.borderWidthThin
                    border.color: expanded ? Theme.primary : Theme.border

                    Behavior on border.color {
                        ColorAnimation { duration: Theme.animationDurationFast }
                    }

                    ColumnLayout {
                        id: cardContent
                        anchors.left: parent.left
                        anchors.right: parent.right
                        anchors.top: parent.top
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingSmall

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingSmall

                            Rectangle {
                                Layout.preferredWidth: 8
                                Layout.preferredHeight: 8
                                radius: 4
                                color: modelData.active ? Theme.success : Theme.textDisabled
                            }

                            Text {
                                text: modelData.device_id || "-"
                                font.pixelSize: Theme.fontSizeLarge
                                font.weight: Font.DemiBold
                                color: Theme.text
                                elide: Text.ElideRight
                                Layout.fillWidth: true
                            }

                            Rectangle {
                                Layout.preferredWidth: stateLabel.implicitWidth
                                                       + Theme.spacingMedium * 2
                                Layout.preferredHeight: 24
                                radius: 12
                                color: modelData.active
                                       ? Qt.rgba(Theme.success.r, Theme.success.g, Theme.success.b, 0.12)
                                       : Theme.surfaceVariant

                                Text {
                                    id: stateLabel
                                    anchors.centerIn: parent
                                    text: modelData.active ? qsTr("Active") : qsTr("Ended")
                                    font.pixelSize: Theme.fontSizeSmall
                                    font.weight: Font.DemiBold
                                    color: modelData.active ? Theme.success : Theme.textSecondary
                                }
                            }

                            Text {
                                text: root.formatDuration(modelData.duration_seconds)
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            QDButton {
                                visible: modelData.active === true
                                text: qsTr("Kick")
                                buttonType: QDButton.Type.Danger
                                implicitWidth: 82
                                onClicked: root.requestKick(modelData.session_id,
                                                            modelData.device_id)
                            }

                            QDButton {
                                visible: modelData.active === true
                                         && (modelData.client_ip || "").length > 0
                                text: qsTr("Ban source")
                                buttonType: QDButton.Type.Secondary
                                implicitWidth: 104
                                onClicked: root.requestBanSource(modelData.device_id,
                                                                 modelData.client_ip)
                            }

                            QDIconButton {
                                iconSource: sessionCard.expanded
                                            ? FluentIconGlyph.chevronUpGlyph
                                            : FluentIconGlyph.chevronDownGlyph
                                toolTipText: sessionCard.expanded ? qsTr("Collapse") : qsTr("Details")
                                buttonStyle: QDIconButton.Style.Transparent
                                Accessible.name: toolTipText
                                onClicked: {
                                    if (sessionCard.expanded) {
                                        root.expandedSessionId = ""
                                        return
                                    }
                                    root.expandedSessionId = modelData.session_id
                                    if (!modelData.detail_loaded && root.manager) {
                                        root.manager.fetchDetail(modelData.session_id)
                                    }
                                }
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            Text {
                                text: qsTr("Started: %1").arg(root.formatTime(modelData.started_at))
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                elide: Text.ElideRight
                                Layout.fillWidth: true
                            }

                            Text {
                                text: modelData.client_id || "-"
                                font.family: "Menlo"
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                elide: Text.ElideRight
                                Layout.preferredWidth: 130
                            }

                            Text {
                                text: modelData.client_ip || "-"
                                font.family: "Menlo"
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                elide: Text.ElideRight
                                Layout.preferredWidth: 110
                            }

                            Text {
                                visible: !modelData.active && modelData.end_reason
                                text: qsTr("End reason: %1").arg(root.endReasonText(modelData.end_reason))
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                elide: Text.ElideRight
                                Layout.preferredWidth: 130
                            }
                        }

                        Rectangle {
                            visible: sessionCard.expanded
                            Layout.fillWidth: true
                            Layout.preferredHeight: Theme.borderWidthThin
                            color: Theme.border
                        }

                        RowLayout {
                            visible: sessionCard.expanded
                            Layout.fillWidth: true
                            Layout.topMargin: Theme.spacingSmall
                            spacing: Theme.spacingLarge

                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.alignment: Qt.AlignTop
                                spacing: Theme.spacingSmall

                                Text {
                                    text: qsTr("Recordings (%1)")
                                          .arg(modelData.recording_count || 0)
                                    font.weight: Font.DemiBold
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }

                                Text {
                                    visible: !modelData.detail_loaded
                                    text: qsTr("Loading...")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                Text {
                                    visible: modelData.detail_loaded
                                             && (modelData.recordings || []).length === 0
                                    text: qsTr("None")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                Repeater {
                                    model: modelData.recordings || []

                                    Text {
                                        required property var modelData
                                        Layout.fillWidth: true
                                        text: "#" + modelData.id + "  "
                                              + modelData.status + "  "
                                              + root.formatSize(modelData.size_bytes)
                                        elide: Text.ElideRight
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.text
                                    }
                                }
                            }

                            ColumnLayout {
                                Layout.fillWidth: true
                                Layout.alignment: Qt.AlignTop
                                spacing: Theme.spacingSmall

                                Text {
                                    text: qsTr("Access log (%1)")
                                          .arg(modelData.access_log_count || 0)
                                    font.weight: Font.DemiBold
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.text
                                }

                                Text {
                                    visible: !modelData.detail_loaded
                                    text: qsTr("Loading...")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                Text {
                                    visible: modelData.detail_loaded
                                             && (modelData.access_log || []).length === 0
                                    text: qsTr("None")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                Repeater {
                                    model: modelData.access_log || []

                                    Text {
                                        required property var modelData
                                        Layout.fillWidth: true
                                        text: root.formatTime(modelData.created_at)
                                              + "  " + (modelData.result || "-")
                                              + "  " + (modelData.ip || "-")
                                        elide: Text.ElideRight
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.text
                                    }
                                }
                            }
                        }
                    }
                }

                footer: Item {
                    width: sessionList.width
                    height: root.manager && root.manager.hasMore
                            ? Theme.buttonHeightMedium + Theme.spacingLarge * 2 : 0

                    QDButton {
                        anchors.centerIn: parent
                        visible: root.manager && root.manager.hasMore
                        text: qsTr("Load more")
                        buttonType: QDButton.Type.Secondary
                        loading: root.manager ? root.manager.busy : false
                        onClicked: {
                            if (root.manager) root.manager.loadMore()
                        }
                    }
                }
            }

            ColumnLayout {
                anchors.centerIn: parent
                visible: sessionList.count === 0 && !(root.manager && root.manager.busy)
                spacing: Theme.spacingMedium

                Text {
                    text: FluentIconGlyph.historyGlyph
                    font.family: "Segoe Fluent Icons"
                    font.pixelSize: 40
                    color: Theme.textDisabled
                    Layout.alignment: Qt.AlignHCenter
                }

                Text {
                    text: qsTr("No sessions match these filters")
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.textSecondary
                    Layout.alignment: Qt.AlignHCenter
                }
            }

            QDSpinner {
                anchors.centerIn: parent
                visible: sessionList.count === 0 && root.manager && root.manager.busy
                running: visible
            }
        }
    }

    QDDialog {
        id: actionDialog
        dialogWidth: 440
        dialogHeight: 230
        property string message: ""
        closeOnOverlay: false

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingLarge

            Text {
                text: actionDialog.message
                wrapMode: Text.WordWrap
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.text
                Layout.fillWidth: true
            }

            Item { Layout.fillHeight: true }
        }

        footer: [
            QDButton {
                text: qsTr("Cancel")
                buttonType: QDButton.Type.Secondary
                onClicked: actionDialog.reject()
            },
            QDButton {
                text: root.confirmActionText()
                buttonType: QDButton.Type.Danger
                onClicked: {
                    var mode = root.confirmMode
                    var sessionId = root.pendingSessionId
                    var deviceId = root.pendingDeviceId
                    var source = root.pendingSource
                    actionDialog.accept()
                    if (!root.manager) return
                    if (mode === 1) {
                        root.manager.revokeSession(sessionId)
                    } else if (mode === 2) {
                        root.manager.revokeDeviceSessions(deviceId)
                    } else if (mode === 3) {
                        root.manager.banSource(deviceId, source)
                    } else if (mode === 4) {
                        root.manager.removeDeniedSource(deviceId, source)
                    }
                }
            }
        ]
    }
}
