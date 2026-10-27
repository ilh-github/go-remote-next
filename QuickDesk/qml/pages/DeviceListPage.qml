import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

import "../component"
import "../quickdeskcomponent"
import "../quickdeskcomponent/DeviceDisplay.js" as DeviceDisplay

Item {
    id: root

    required property var mainController

    signal connectToDevice(string deviceId, string accessCode)
    signal showToast(string message, int toastType)

    property bool isLoggedIn: mainController && mainController.authManager ? mainController.authManager.isLoggedIn : false

    property var myDevices: mainController && mainController.cloudDeviceManager
                            ? mainController.cloudDeviceManager.myDevices : []
    property var groupManager: mainController && mainController.deviceGroupManager
                               ? mainController.deviceGroupManager : null
    property int selectedGroupId: 0
    property string pendingConnectDeviceId: ""

    property var filteredDevices: {
        if (root.selectedGroupId <= 0 || !root.groupManager)
            return root.myDevices
        var groups = root.groupManager.groups || []
        for (var i = 0; i < groups.length; ++i) {
            if (Number(groups[i].id) !== root.selectedGroupId) continue
            var members = groups[i].device_ids || []
            return root.myDevices.filter(function(device) {
                return members.indexOf(device.device_id || "") >= 0
            })
        }
        return []
    }

    function rebuildGroupFilter() {
        var previousGroupId = root.selectedGroupId
        groupFilterModel.clear()
        groupFilterModel.append({
            groupId: 0,
            text: qsTr("All devices") + " (" + root.myDevices.length + ")"
        })
        var groups = root.groupManager ? root.groupManager.groups || [] : []
        var previousIndex = 0
        for (var i = 0; i < groups.length; ++i) {
            var group = groups[i]
            var groupId = Number(group.id || 0)
            groupFilterModel.append({
                groupId: groupId,
                text: (group.name || qsTr("Untitled group"))
                      + " (" + Number(group.device_count || 0) + ")"
            })
            if (groupId === previousGroupId)
                previousIndex = groupFilterModel.count - 1
        }
        var exists = previousGroupId === 0
        for (var j = 0; j < groups.length; ++j) {
            if (Number(groups[j].id) === previousGroupId) {
                exists = true
                break
            }
        }
        if (!exists) {
            root.selectedGroupId = 0
            previousIndex = 0
        }
        groupFilter.currentIndex = previousIndex
    }

    Component.onCompleted: root.rebuildGroupFilter()

    function requestConnection(deviceId, savedAccessCode) {
        if (!deviceId) return

        var accessCode = root.mainController.cloudDeviceManager.getDeviceAccessCode(deviceId)
        if (!accessCode) {
            accessCode = savedAccessCode
        }
        if (accessCode) {
            root.connectToDevice(deviceId, accessCode)
            return
        }

        root.pendingConnectDeviceId = deviceId
        root.mainController.cloudDeviceManager.requestDeviceAccessCode(deviceId)
    }

    Connections {
        target: root.mainController && root.mainController.cloudDeviceManager
                ? root.mainController.cloudDeviceManager : null

        function onDeviceAccessCodeReady(deviceId, accessCode) {
            if (root.pendingConnectDeviceId !== deviceId) return
            root.pendingConnectDeviceId = ""
            root.connectToDevice(deviceId, accessCode)
        }

        function onDeviceAccessCodeFailed(deviceId, error) {
            if (root.pendingConnectDeviceId !== deviceId) return
            root.pendingConnectDeviceId = ""
            root.showToast(error, 2)
        }

        function onMyDevicesChanged() {
            root.rebuildGroupFilter()
        }
    }

    Connections {
        target: root.groupManager

        function onGroupsChanged() {
            root.rebuildGroupFilter()
        }

        function onRequestFailed(message) {
            root.showToast(message, 2)
        }
    }

    ListModel {
        id: groupFilterModel
    }

    // Not logged in prompt
    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.contactInfoGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to view device list")
                font.pixelSize: Theme.fontSizeMedium
                color: Theme.textSecondary
                Layout.alignment: Qt.AlignHCenter
            }

            QDButton {
                text: qsTr("Login")
                highlighted: true
                Layout.alignment: Qt.AlignHCenter
                onClicked: loginDialog.open()
            }
        }
    }

    // Logged in content
    ColumnLayout {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        anchors.bottom: parent.bottom
        anchors.margins: Theme.spacingLarge
        visible: root.isLoggedIn
        spacing: Theme.spacingSmall

        // ---- User-owned address-book groups ----
        RowLayout {
            Layout.fillWidth: true
            Layout.preferredHeight: Theme.buttonHeightMedium
            spacing: Theme.spacingSmall

            Text {
                text: FluentIconGlyph.groupGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: Theme.iconSizeMedium
                color: Theme.primary
            }

            Text {
                text: qsTr("Device groups")
                font.pixelSize: Theme.fontSizeMedium
                font.weight: Font.DemiBold
                color: Theme.text
            }

            QDComboBox {
                id: groupFilter
                Layout.fillWidth: true
                textRole: "text"
                valueRole: "groupId"
                model: groupFilterModel
                onActivated: function(index) {
                    root.selectedGroupId = Number(groupFilterModel.get(index).groupId)
                }
            }

            QDSpinner {
                size: 20
                running: root.groupManager ? root.groupManager.busy : false
                visible: running
            }

            QDButton {
                text: qsTr("Manage groups")
                iconText: FluentIconGlyph.settingsGlyph
                onClicked: deviceGroupsDialog.open()
            }
        }

        // ---- My Devices Section ----
        QDAccordion {
            Layout.fillWidth: true
            title: qsTr("My Devices") + (deviceRepeater.count > 0 ? " (" + deviceRepeater.count + ")" : "")
            iconSource: FluentIconGlyph.contactInfoGlyph
            expanded: true

            Item {
                width: parent.width
                height: deviceRepeater.count === 0 ? emptyDevicesText.height
                                                   : Math.min(deviceRepeater.count, 4) * 42

                Text {
                    id: emptyDevicesText
                    visible: deviceRepeater.count === 0
                    text: qsTr("No devices bound yet")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                ListView {
                    id: deviceRepeater
                    anchors.fill: parent
                    visible: count > 0
                    clip: true
                    spacing: 2
                    boundsBehavior: Flickable.StopAtBounds
                    ScrollBar.vertical: QDScrollBar {}

                    model: root.filteredDevices

                    delegate: Rectangle {
                        required property var modelData
                        required property int index

                        width: deviceRepeater.width
                        height: 40
                        radius: Theme.radiusSmall
                        color: deviceRowHover.hovered ? Theme.surfaceHover : "transparent"

                        HoverHandler { id: deviceRowHover }

                        Behavior on color {
                            ColorAnimation { duration: Theme.animationDurationFast }
                        }

                        RowLayout {
                            anchors.fill: parent
                            anchors.leftMargin: Theme.spacingSmall
                            anchors.rightMargin: Theme.spacingSmall
                            spacing: Theme.spacingSmall

                            Rectangle {
                                width: 8; height: 8; radius: 4
                                color: (modelData.online && modelData.logged_in) ? Theme.success : Theme.textDisabled
                            }

                            Text {
                                Layout.fillWidth: true
                                text: DeviceDisplay.deviceLabel(
                                          modelData,
                                          root.mainController ? root.mainController.deviceId : "",
                                          qsTr("Device"),
                                          qsTr("This device"))
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                elide: Text.ElideRight
                            }

                            QDIconButton {
                                visible: modelData.online === true && modelData.logged_in === true
                                iconSource: FluentIconGlyph.remoteGlyph
                                buttonSize: QDIconButton.Size.Small

                                QDToolTip { visible: parent.hovered; text: qsTr("Connect") }

                                onClicked: root.requestConnection(modelData.device_id, "")
                            }
                        }

                        MouseArea {
                            anchors.fill: parent
                            hoverEnabled: true
                            acceptedButtons: Qt.LeftButton | Qt.RightButton
                            z: -1
                            onClicked: function(mouse) {
                                if (mouse.button === Qt.RightButton) {
                                    deviceContextMenu.deviceId = modelData.device_id
                                    deviceContextMenu.deviceRemark = modelData.remark || ""
                                    var pos = mapToItem(root, mouse.x, mouse.y)
                                    deviceContextMenu.x = pos.x
                                    deviceContextMenu.y = pos.y
                                    deviceContextMenu.open()
                                }
                            }
                            onDoubleClicked: {
                                if (modelData.online !== true || modelData.logged_in !== true) return
                                root.requestConnection(modelData.device_id, "")
                            }
                        }
                    }
                }
            }
        }

        // ---- My Favorites Section ----
        QDAccordion {
            Layout.fillWidth: true
            title: qsTr("My Favorites") + (favoriteRepeater.count > 0 ? " (" + favoriteRepeater.count + ")" : "")
            iconSource: FluentIconGlyph.favoriteStarGlyph
            expanded: true

            Item {
                width: parent.width
                height: favoriteRepeater.count === 0 ? emptyFavText.height
                                                     : Math.min(favoriteRepeater.count, 4) * 42

                Text {
                    id: emptyFavText
                    visible: favoriteRepeater.count === 0
                    text: qsTr("No favorites yet")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                ListView {
                    id: favoriteRepeater
                    anchors.fill: parent
                    visible: count > 0
                    clip: true
                    spacing: 2
                    boundsBehavior: Flickable.StopAtBounds
                    ScrollBar.vertical: QDScrollBar {}

                    model: root.mainController && root.mainController.cloudDeviceManager
                           ? root.mainController.cloudDeviceManager.myFavorites : []

                    delegate: Rectangle {
                        required property var modelData
                        required property int index

                        width: favoriteRepeater.width
                        height: 40
                        radius: Theme.radiusSmall
                        color: favRowHover.hovered ? Theme.surfaceHover : "transparent"

                        HoverHandler { id: favRowHover }

                        Behavior on color {
                            ColorAnimation { duration: Theme.animationDurationFast }
                        }

                        RowLayout {
                            anchors.fill: parent
                            anchors.leftMargin: Theme.spacingSmall
                            anchors.rightMargin: Theme.spacingSmall
                            spacing: Theme.spacingSmall

                            Text {
                                text: FluentIconGlyph.favoriteStarFillGlyph
                                font.family: "Segoe Fluent Icons"
                                font.pixelSize: 12
                                color: Theme.warning
                            }

                            Text {
                                Layout.fillWidth: true
                                text: DeviceDisplay.deviceLabel(
                                          modelData,
                                          root.mainController ? root.mainController.deviceId : "",
                                          qsTr("Device"),
                                          qsTr("This device"))
                                font.pixelSize: Theme.fontSizeMedium
                                color: Theme.text
                                elide: Text.ElideRight
                            }

                            QDIconButton {
                                iconSource: FluentIconGlyph.remoteGlyph
                                buttonSize: QDIconButton.Size.Small

                                QDToolTip { visible: parent.hovered; text: qsTr("Connect") }

                                onClicked: root.requestConnection(
                                               modelData.device_id,
                                               modelData.access_password || "")
                            }
                        }

                        MouseArea {
                            anchors.fill: parent
                            hoverEnabled: true
                            acceptedButtons: Qt.LeftButton | Qt.RightButton
                            z: -1
                            onClicked: function(mouse) {
                                if (mouse.button === Qt.RightButton) {
                                    favContextMenu.deviceId = modelData.device_id
                                    favContextMenu.deviceName = modelData.device_name || ""
                                    var pos = mapToItem(root, mouse.x, mouse.y)
                                    favContextMenu.x = pos.x
                                    favContextMenu.y = pos.y
                                    favContextMenu.open()
                                }
                            }
                            onDoubleClicked: {
                                root.requestConnection(
                                            modelData.device_id,
                                            modelData.access_password || "")
                            }
                        }
                    }
                }
            }
        }

        // ---- Connection Logs Section ----
        QDAccordion {
            Layout.fillWidth: true
            Layout.fillHeight: true
            title: qsTr("Connection Logs") + (logsRepeater.count > 0 ? " (" + logsRepeater.count + ")" : "")
            iconSource: FluentIconGlyph.historyGlyph
            expanded: false

            Item {
                width: parent.width
                height: logsRepeater.count === 0 ? emptyLogsText.height : Math.min(logsRepeater.count, 6) * 42

                Text {
                    id: emptyLogsText
                    visible: logsRepeater.count === 0
                    text: qsTr("No connection logs")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                }

                ListView {
                    id: logsRepeater
                    anchors.fill: parent
                    visible: count > 0
                    clip: true
                    spacing: 2
                    boundsBehavior: Flickable.StopAtBounds
                    ScrollBar.vertical: QDScrollBar {}

                    model: {
                        var logs = root.mainController && root.mainController.cloudDeviceManager
                                  ? root.mainController.cloudDeviceManager.connectionLogs : []
                        return logs.length > 20 ? logs.slice(0, 20) : logs
                    }

                    delegate: Rectangle {
                        required property var modelData
                        required property int index

                        width: logsRepeater.width
                        height: 40
                        radius: Theme.radiusSmall
                        color: "transparent"

                        RowLayout {
                            anchors.fill: parent
                            anchors.leftMargin: Theme.spacingSmall
                            anchors.rightMargin: Theme.spacingSmall
                            spacing: Theme.spacingSmall

                            Rectangle {
                                width: 8; height: 8; radius: 4
                                color: modelData.status === "success" ? Theme.success : Theme.error
                            }

                            Text {
                                Layout.fillWidth: true
                                text: {
                                    var parts = [modelData.device_id || ""]
                                    if (modelData.created_at) parts.push(new Date(modelData.created_at).toLocaleString())
                                    if (modelData.duration > 0) {
                                        var m = Math.floor(modelData.duration / 60)
                                        var s = modelData.duration % 60
                                        parts.push(m + "m" + s + "s")
                                    }
                                    if (modelData.error_msg) parts.push(modelData.error_msg)
                                    return parts.join(" · ")
                                }
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                elide: Text.ElideRight
                            }
                        }
                    }
                }
            }
        }
    }

    // Context menu for My Devices
    QDMenu {
        id: deviceContextMenu
        property string deviceId: ""
        property string deviceRemark: ""

        QDMenuItem {
            text: qsTr("Set Remark")
            iconText: FluentIconGlyph.editGlyph
            onTriggered: {
                remarkDialog.deviceId = deviceContextMenu.deviceId
                remarkDialog.isFavorite = false
                remarkDialog.currentRemark = deviceContextMenu.deviceRemark
                remarkDialog.open()
            }
        }
        QDMenuItem {
            text: qsTr("Remove")
            iconText: FluentIconGlyph.deleteGlyph
            isDestructive: true
            onTriggered: root.mainController.cloudDeviceManager.unbindDevice(deviceContextMenu.deviceId)
        }
    }

    // Context menu for Favorites
    QDMenu {
        id: favContextMenu
        property string deviceId: ""
        property string deviceName: ""

        QDMenuItem {
            text: qsTr("Edit Remark")
            iconText: FluentIconGlyph.editGlyph
            onTriggered: {
                remarkDialog.deviceId = favContextMenu.deviceId
                remarkDialog.isFavorite = true
                remarkDialog.currentRemark = favContextMenu.deviceName
                remarkDialog.open()
            }
        }
        QDMenuItem {
            text: qsTr("Remove Favorite")
            iconText: FluentIconGlyph.favoriteStarGlyph
            isDestructive: true
            onTriggered: root.mainController.cloudDeviceManager.removeFavorite(favContextMenu.deviceId)
        }
    }

    // Remark edit dialog
    Popup {
        id: remarkDialog
        modal: true
        anchors.centerIn: parent
        width: 300
        padding: Theme.spacingLarge

        property string deviceId: ""
        property bool isFavorite: false
        property string currentRemark: ""

        background: Rectangle {
            color: Theme.surface
            radius: Theme.radiusLarge
            border.width: Theme.borderWidthThin
            border.color: Theme.border
        }

        onOpened: remarkField.text = currentRemark

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingMedium

            Text {
                text: qsTr("Set Remark")
                font.pixelSize: Theme.fontSizeLarge
                font.weight: Font.Bold
                color: Theme.text
            }

            QDTextField {
                id: remarkField
                Layout.fillWidth: true
                placeholderText: qsTr("Enter remark name")
            }

            RowLayout {
                Layout.alignment: Qt.AlignRight
                spacing: Theme.spacingSmall

                QDButton {
                    text: qsTr("Cancel")
                    onClicked: remarkDialog.close()
                }

                QDButton {
                    text: qsTr("Save")
                    highlighted: true
                    enabled: remarkField.text.length > 0
                    onClicked: {
                        var id = remarkDialog.deviceId
                        var text = remarkField.text
                        var isOwnDevice = root.myDevices.some(function(d) { return d.device_id === id })
                        var isFav = root.mainController.cloudDeviceManager.myFavorites.some(function(f) { return f.device_id === id })
                        if (isOwnDevice)
                            root.mainController.cloudDeviceManager.setDeviceRemark(id, text)
                        if (isFav)
                            root.mainController.cloudDeviceManager.updateFavorite(id, text, "")
                        remarkDialog.close()
                    }
                }
            }
        }
    }

    // Login dialog
    LoginDialog {
        id: loginDialog
        mainController: root.mainController
    }

    // The host binds itself on startup. When the account is already at its
    // device limit that bind is refused, and without this the machine would sit
    // here logged in, online, and absent from the owner's list with nothing on
    // screen to say why. The server's detail already names the two ways out
    // (unbind something, or have the limit raised), so it is shown verbatim.
    Connections {
        target: root.mainController ? root.mainController.cloudDeviceManager : null

        function onDeviceBindQuotaExceeded(deviceId, detail) {
            root.showToast(detail, 2)
        }
    }

    DeviceGroupManagerDialog {
        id: deviceGroupsDialog
        groupManager: root.groupManager
        myDevices: root.myDevices

        onShowToast: function(message, toastType) {
            root.showToast(message, toastType)
        }
    }
}
