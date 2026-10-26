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

    property var manager: mainController && mainController.scheduledTaskManager
                          ? mainController.scheduledTaskManager : null
    property var cloudManager: mainController && mainController.cloudDeviceManager
                               ? mainController.cloudDeviceManager : null
    property bool isLoggedIn: mainController && mainController.authManager
                              ? mainController.authManager.isLoggedIn : false
    property string mode: "once"
    property date runDate: new Date(Date.now() + 3600000)
    property int pendingDeleteId: 0
    property string pendingDeleteKind: ""
    property bool refreshPending: false
    property var modeOptions: ["once", "interval", "calendar"]
    property var weekdayValues: ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]
    property var calendarValues: ["daily", "weekdays", "weekly", "monthly", "every"]

    function kindLabel(kind) {
        if (kind === "revoke_all_remote_sessions")
            return qsTr("Revoke all remote sessions")
        if (kind === "sign_out_all_sessions")
            return qsTr("Sign out all sessions")
        if (kind === "connect_prompt")
            return qsTr("Prompt to connect")
        return kind || qsTr("Unknown action")
    }

    function modeLabel(value) {
        if (value === "once") return qsTr("Once")
        if (value === "interval") return qsTr("Interval")
        return qsTr("Calendar")
    }

    function calendarLabel(value) {
        if (value === "daily") return qsTr("Every day")
        if (value === "weekdays") return qsTr("Workdays")
        if (value === "weekly") return qsTr("Every week")
        if (value === "monthly") return qsTr("Every month")
        return qsTr("Every N days")
    }

    function weekdayLabel(value) {
        if (value === "mon") return qsTr("Monday")
        if (value === "tue") return qsTr("Tuesday")
        if (value === "wed") return qsTr("Wednesday")
        if (value === "thu") return qsTr("Thursday")
        if (value === "fri") return qsTr("Friday")
        if (value === "sat") return qsTr("Saturday")
        return qsTr("Sunday")
    }

    function formatTime(value) {
        if (!value) return "-"
        var parsed = new Date(value)
        return isNaN(parsed.getTime()) ? String(value) : parsed.toLocaleString()
    }

    function formatInterval(seconds) {
        var minutes = Math.max(1, Math.round(Number(seconds || 0) / 60))
        if (minutes % 1440 === 0)
            return qsTr("%1 day(s)").arg(minutes / 1440)
        if (minutes % 60 === 0)
            return qsTr("%1 hour(s)").arg(minutes / 60)
        return qsTr("%1 minute(s)").arg(minutes)
    }

    function cadenceLabel(task) {
        if (task.schedule) {
            var parts = String(task.schedule).split("@")
            var time = parts[parts.length - 1]
            if (parts[0] === "daily")
                return qsTr("Every day at %1").arg(time)
            if (parts[0] === "weekdays")
                return qsTr("Workdays at %1").arg(time)
            if (parts[0] === "weekly")
                return qsTr("Every %1 at %2")
                    .arg(root.weekdayLabel(parts[1])).arg(time)
            if (parts[0] === "monthly")
                return qsTr("Day %1 at %2").arg(parts[1]).arg(time)
            if (parts[0] === "every")
                return qsTr("Every %1 days at %2").arg(parts[1]).arg(time)
            return task.schedule
        }
        if (Number(task.repeat_seconds) > 0)
            return qsTr("Every %1").arg(root.formatInterval(task.repeat_seconds))
        return qsTr("Runs once")
    }

    function taskMode(task) {
        if (task.schedule) return qsTr("Calendar")
        if (Number(task.repeat_seconds) > 0) return qsTr("Interval")
        return qsTr("Once")
    }

    function taskState(task) {
        if (task.last_status === "failed") return qsTr("Failed")
        if (!task.enabled) return qsTr("Finished")
        if (Number(task.attempts) > 0) return qsTr("Retrying")
        return qsTr("Scheduled")
    }

    function stateType(task) {
        if (task.last_status === "failed") return QDChip.Type.Error
        if (!task.enabled) return QDChip.Type.Default
        if (Number(task.attempts) > 0) return QDChip.Type.Warning
        return QDChip.Type.Success
    }

    function countTasks(predicate) {
        if (!root.manager) return 0
        var count = 0
        for (var i = 0; i < root.manager.tasks.length; ++i) {
            if (predicate(root.manager.tasks[i])) count += 1
        }
        return count
    }

    function rebuildKinds() {
        kindModel.clear()
        var values = root.manager && root.manager.supportedKinds.length > 0
                     ? root.manager.supportedKinds
                    // The same fallback the web client had to be corrected for:
                    // a second copy of the kind list that omits the newer kind
                    // means the create dialog cannot offer it before
                    // supported_kinds arrives.
                    : ["sign_out_all_sessions",
                       "revoke_all_remote_sessions",
                       "connect_prompt"]
        for (var i = 0; i < values.length; ++i) {
            kindModel.append({ value: values[i], text: root.kindLabel(values[i]) })
        }
        if (kindModel.count > 0) kindCombo.currentIndex = 0
    }

    function resetCreateForm() {
        root.mode = "once"
        root.runDate = new Date(Date.now() + 3600000)
        root.runDate.setSeconds(0, 0)
        maxAttemptsField.text = "1"
        repeatMinutesField.text = "60"
        calendarTypeCombo.currentIndex = 0
        weekdayCombo.currentIndex = 0
        monthDayField.text = "1"
        everyDaysField.text = "3"
        calendarTimeField.text = "22:00"
        timezoneField.text = ""
        skipHolidaysSwitch.checked = false
        rebuildKinds()
    }

    function openCreate() {
        resetCreateForm()
        createDialog.show()
    }

    function calendarSchedule() {
        var type = calendarTypeCombo.selectedValue
        var time = calendarTimeField.text.trim()
        if (type === "daily") return "daily@" + time
        if (type === "weekdays") return "weekdays@" + time
        if (type === "weekly")
            return "weekly@" + weekdayCombo.selectedValue + "@" + time
        if (type === "monthly")
            return "monthly@" + Number(monthDayField.text || 1) + "@" + time
        return "every@" + Number(everyDaysField.text || 1) + "@days@" + time
    }

    function submitCreate() {
        if (!root.manager || kindCombo.currentIndex < 0) return
        var attempts = Number(maxAttemptsField.text)
        if (!Number.isInteger(attempts) || attempts < 1 || attempts > 10) {
            root.showToast(qsTr("Retry attempts must be between 1 and 10"),
                           QDToast.Type.Warning)
            return
        }

        if (root.mode === "calendar") {
            if (!/^([01][0-9]|2[0-3]):[0-5][0-9]$/.test(calendarTimeField.text.trim())) {
                root.showToast(qsTr("Enter a valid time in HH:MM format"),
                               QDToast.Type.Warning)
                return
            }
            root.manager.createTask(kindCombo.selectedValue, "", 0, attempts,
                                    root.calendarSchedule(),
                                    timezoneField.text.trim(),
                                    skipHolidaysSwitch.checked)
            return
        }

        if (root.runDate.getTime() <= Date.now()) {
            root.showToast(qsTr("Choose a future start time"), QDToast.Type.Warning)
            return
        }
        var repeatSeconds = 0
        if (root.mode === "interval") {
            var minutes = Number(repeatMinutesField.text)
            if (!Number.isInteger(minutes) || minutes < 1) {
                root.showToast(qsTr("Enter a positive interval"),
                               QDToast.Type.Warning)
                return
            }
            repeatSeconds = minutes * 60
        }
        root.manager.createTask(kindCombo.selectedValue,
                                root.runDate.toISOString(),
                                repeatSeconds, attempts,
                                "", "", false)
    }

    function requestDelete(task) {
        root.pendingDeleteId = Number(task.id)
        root.pendingDeleteKind = task.kind
        deleteDialog.show()
    }

    function scheduleRefresh() {
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
            if (root.isLoggedIn) {
                Qt.callLater(root.manager.refresh)
            }
        }
    }

    Connections {
        target: root.manager

        function onRequestFailed(message) {
            root.showToast(message, QDToast.Type.Error)
        }

        function onTaskCreated(taskId) {
            createDialog.hide()
            root.showToast(qsTr("Scheduled task created"), QDToast.Type.Success)
        }

        function onTaskDeleted(taskId) {
            deleteDialog.hide()
            root.showToast(qsTr("Scheduled task deleted"), QDToast.Type.Success)
        }

        function onRealtimeTaskFailed(kind, taskId, error) {
            root.showToast(qsTr("%1 task failed: %2")
                           .arg(root.kindLabel(kind)).arg(error),
                           QDToast.Type.Error)
        }

        // QD-607: a scheduled connection came due. The server publishes it, the
        // web client consumes it, and the desktop client is the last place it has
        // to reach - the same layer-by-layer gap the notification subscription
        // and the userSync branch were.
        function onRealtimeConnectPrompted(taskId) {
            root.showToast(qsTr("A scheduled connection is due. Open the task to connect."),
                           QDToast.Type.Info)
        }

        function onRealtimeRefreshRequested() {
            root.scheduleRefresh()
        }

        function onSupportedKindsChanged() {
            root.rebuildKinds()
        }

        function onBusyChanged() {
            if (root.manager && !root.manager.busy && root.refreshPending) {
                root.scheduleRefresh()
            }
        }
    }

    Connections {
        target: root.cloudManager

        function onSyncMessage(message) {
            if (root.manager) root.manager.handleRealtimeEvent(message)
        }
    }

    Timer {
        id: realtimeRefreshTimer
        interval: 150
        repeat: false
        onTriggered: {
            if (!root.manager || !root.isLoggedIn) {
                root.refreshPending = false
                return
            }
            if (root.manager.busy) {
                root.refreshPending = true
                return
            }
            root.refreshPending = false
            root.manager.refresh()
        }
    }

    Timer {
        interval: 30000
        repeat: true
        running: root.isLoggedIn && root.manager && !root.manager.busy
        onTriggered: root.manager.refresh()
    }

    Component.onCompleted: {
        rebuildKinds()
        if (root.isLoggedIn) Qt.callLater(root.manager.refresh)
    }

    ListModel { id: kindModel }

    Item {
        anchors.fill: parent
        visible: !root.isLoggedIn

        ColumnLayout {
            anchors.centerIn: parent
            spacing: Theme.spacingLarge

            Text {
                text: FluentIconGlyph.calendarGlyph
                font.family: "Segoe Fluent Icons"
                font.pixelSize: 48
                color: Theme.textDisabled
                Layout.alignment: Qt.AlignHCenter
            }

            Text {
                text: qsTr("Please login to manage scheduled tasks")
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
                    text: qsTr("Scheduled Tasks")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                Text {
                    text: qsTr("Run control-plane actions once, on an interval, or by calendar.")
                    font.pixelSize: Theme.fontSizeSmall
                    color: Theme.textSecondary
                    wrapMode: Text.WordWrap
                }
            }

            QDButton {
                text: qsTr("Refresh")
                iconText: FluentIconGlyph.syncGlyph
                buttonType: QDButton.Type.Secondary
                loading: root.manager ? root.manager.busy : false
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.manager.refresh()
            }

            QDButton {
                text: qsTr("Create task")
                iconText: FluentIconGlyph.addGlyph
                buttonType: QDButton.Type.Primary
                enabled: !!root.manager && !root.manager.busy
                onClicked: root.openCreate()
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: Theme.spacingSmall

            Repeater {
                model: [
                    {
                        label: qsTr("Active"),
                        value: root.countTasks(function(task) { return task.enabled === true })
                    },
                    {
                        label: qsTr("Recurring"),
                        value: root.countTasks(function(task) {
                            return task.enabled === true
                                && (task.schedule || Number(task.repeat_seconds) > 0)
                        })
                    },
                    {
                        label: qsTr("Failed"),
                        value: root.countTasks(function(task) {
                            return task.last_status === "failed"
                        }),
                        danger: true
                    }
                ]

                Rectangle {
                    id: summaryTile
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.preferredHeight: 72
                    color: Theme.surface
                    radius: Theme.radiusMedium
                    border.width: Theme.borderWidthThin
                    border.color: Theme.border

                    Column {
                        anchors.centerIn: parent
                        spacing: 2

                        Text {
                            anchors.horizontalCenter: parent.horizontalCenter
                            text: summaryTile.modelData.value
                            font.pixelSize: Theme.fontSizeXLarge
                            font.weight: Font.Bold
                            color: summaryTile.modelData.danger
                                   ? Theme.error : Theme.text
                        }

                        Text {
                            anchors.horizontalCenter: parent.horizontalCenter
                            text: summaryTile.modelData.label
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }
                    }
                }
            }
        }

        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: Theme.surface
            radius: Theme.radiusMedium
            border.width: Theme.borderWidthThin
            border.color: Theme.border

            ListView {
                id: taskList
                anchors.fill: parent
                anchors.margins: Theme.spacingMedium
                model: root.manager ? root.manager.tasks : []
                spacing: Theme.spacingSmall
                clip: true
                ScrollBar.vertical: QDScrollBar {}

                delegate: Rectangle {
                    id: taskCard
                    required property var modelData

                    width: taskList.width
                    height: taskContent.implicitHeight + Theme.spacingLarge * 2
                    radius: Theme.radiusMedium
                    color: Theme.surfaceVariant
                    border.width: Theme.borderWidthThin
                    border.color: taskCard.modelData.last_status === "failed"
                                  ? Qt.rgba(Theme.error.r, Theme.error.g, Theme.error.b, 0.45)
                                  : Theme.border

                    RowLayout {
                        id: taskContent
                        anchors.fill: parent
                        anchors.margins: Theme.spacingLarge
                        spacing: Theme.spacingLarge

                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingXSmall

                            RowLayout {
                                spacing: Theme.spacingSmall

                                Text {
                                    text: root.kindLabel(taskCard.modelData.kind)
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.DemiBold
                                    color: Theme.text
                                }

                                QDChip {
                                    text: root.taskState(taskCard.modelData)
                                    chipType: root.stateType(taskCard.modelData)
                                }

                                QDChip {
                                    text: root.taskMode(taskCard.modelData)
                                    chipType: QDChip.Type.Default
                                    outlined: true
                                }
                            }

                            Text {
                                text: root.cadenceLabel(taskCard.modelData)
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            Text {
                                Layout.fillWidth: true
                                text: qsTr("Next: %1 · Attempts: %2 / %3 · #%4")
                                      .arg(root.formatTime(taskCard.modelData.next_run_at))
                                      .arg(Number(taskCard.modelData.attempts || 0))
                                      .arg(Number(taskCard.modelData.max_attempts || 1))
                                      .arg(Number(taskCard.modelData.id || 0))
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                                wrapMode: Text.WordWrap
                            }

                            Text {
                                visible: !!taskCard.modelData.last_run_at
                                Layout.fillWidth: true
                                text: qsTr("Last run: %1").arg(
                                          root.formatTime(taskCard.modelData.last_run_at))
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            Text {
                                visible: !!taskCard.modelData.last_error
                                Layout.fillWidth: true
                                text: qsTr("Last error: %1").arg(
                                          taskCard.modelData.last_error || "")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.error
                                wrapMode: Text.WordWrap
                            }
                        }

                        QDButton {
                            text: qsTr("Delete")
                            iconText: FluentIconGlyph.deleteGlyph
                            buttonType: QDButton.Type.Danger
                            onClicked: root.requestDelete(taskCard.modelData)
                        }
                    }
                }
            }

            ColumnLayout {
                anchors.centerIn: parent
                visible: (!root.manager || root.manager.tasks.length === 0)
                         && !(root.manager && root.manager.busy)
                spacing: Theme.spacingMedium

                Text {
                    text: FluentIconGlyph.calendarGlyph
                    font.family: "Segoe Fluent Icons"
                    font.pixelSize: 40
                    color: Theme.textDisabled
                    Layout.alignment: Qt.AlignHCenter
                }

                Text {
                    text: qsTr("No scheduled tasks yet")
                    font.pixelSize: Theme.fontSizeMedium
                    color: Theme.textSecondary
                }

                QDButton {
                    text: qsTr("Create task")
                    buttonType: QDButton.Type.Primary
                    Layout.alignment: Qt.AlignHCenter
                    onClicked: root.openCreate()
                }
            }

            QDSpinner {
                anchors.centerIn: parent
                visible: root.manager && root.manager.busy
                         && root.manager.tasks.length === 0
                running: visible
            }
        }
    }

    QDDialog {
        id: createDialog
        title: qsTr("Create scheduled task")
        dialogWidth: Math.min(760, root.width - 48)
        dialogHeight: Math.min(700, root.height - 48)
        closeOnOverlay: false

        ColumnLayout {
            anchors.fill: parent
            spacing: Theme.spacingMedium

            RowLayout {
                Layout.fillWidth: true
                spacing: Theme.spacingSmall

                Repeater {
                    model: root.modeOptions

                    QDButton {
                        required property var modelData
                        Layout.fillWidth: true
                        text: root.modeLabel(modelData)
                        buttonType: root.mode === modelData
                                     ? QDButton.Type.Primary : QDButton.Type.Secondary
                        onClicked: root.mode = modelData
                    }
                }
            }

            ScrollView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true
                ScrollBar.vertical: QDScrollBar {}

                ColumnLayout {
                    width: parent.width
                    spacing: Theme.spacingMedium

                    RowLayout {
                        Layout.fillWidth: true
                        spacing: Theme.spacingMedium

                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingXSmall

                            Text {
                                text: qsTr("Action")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            QDComboBox {
                                id: kindCombo
                                Layout.fillWidth: true
                                textRole: "text"
                                valueRole: "value"
                                model: kindModel
                            }
                        }

                        ColumnLayout {
                            Layout.preferredWidth: 180
                            spacing: Theme.spacingXSmall

                            Text {
                                text: qsTr("Max attempts")
                                font.pixelSize: Theme.fontSizeSmall
                                color: Theme.textSecondary
                            }

                            QDTextField {
                                id: maxAttemptsField
                                Layout.fillWidth: true
                                text: "1"
                                validator: IntValidator { bottom: 1; top: 10 }
                            }
                        }
                    }

                    ColumnLayout {
                        visible: root.mode !== "calendar"
                        Layout.fillWidth: true
                        spacing: Theme.spacingXSmall

                        Text {
                            text: root.mode === "once"
                                  ? qsTr("Run at") : qsTr("First run")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }

                        QDDateTimePicker {
                            id: runPicker
                            Layout.fillWidth: true
                            showTime: true
                            selectedDate: root.runDate
                            onDateChanged: function(newDate) {
                                root.runDate = newDate
                            }
                        }
                    }

                    ColumnLayout {
                        visible: root.mode === "interval"
                        Layout.fillWidth: true
                        spacing: Theme.spacingXSmall

                        Text {
                            text: qsTr("Repeat every N minutes")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                        }

                        QDTextField {
                            id: repeatMinutesField
                            Layout.fillWidth: true
                            text: "60"
                            validator: IntValidator { bottom: 1; top: 525600 }
                        }
                    }

                    ColumnLayout {
                        visible: root.mode === "calendar"
                        Layout.fillWidth: true
                        spacing: Theme.spacingMedium

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: Theme.spacingXSmall

                                Text {
                                    text: qsTr("Cadence")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                QDComboBox {
                                    id: calendarTypeCombo
                                    Layout.fillWidth: true
                                    textRole: "text"
                                    valueRole: "value"
                                    model: ListModel {
                                        Component.onCompleted: {
                                            for (var i = 0; i < root.calendarValues.length; ++i) {
                                                append({
                                                    value: root.calendarValues[i],
                                                    text: root.calendarLabel(root.calendarValues[i])
                                                })
                                            }
                                        }
                                    }
                                }
                            }

                            ColumnLayout {
                                Layout.preferredWidth: 150
                                spacing: Theme.spacingXSmall

                                Text {
                                    text: qsTr("Time (HH:MM)")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                QDTextField {
                                    id: calendarTimeField
                                    Layout.fillWidth: true
                                    text: "22:00"
                                    placeholderText: "22:00"
                                }
                            }
                        }

                        RowLayout {
                            Layout.fillWidth: true
                            spacing: Theme.spacingMedium

                            ColumnLayout {
                                visible: calendarTypeCombo.selectedValue === "weekly"
                                Layout.fillWidth: true
                                spacing: Theme.spacingXSmall

                                Text {
                                    text: qsTr("Weekday")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                QDComboBox {
                                    id: weekdayCombo
                                    Layout.fillWidth: true
                                    textRole: "text"
                                    valueRole: "value"
                                    model: ListModel {
                                        Component.onCompleted: {
                                            for (var i = 0; i < root.weekdayValues.length; ++i) {
                                                append({
                                                    value: root.weekdayValues[i],
                                                    text: root.weekdayLabel(root.weekdayValues[i])
                                                })
                                            }
                                        }
                                    }
                                }
                            }

                            ColumnLayout {
                                visible: calendarTypeCombo.selectedValue === "monthly"
                                Layout.fillWidth: true
                                spacing: Theme.spacingXSmall

                                Text {
                                    text: qsTr("Day of month")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                QDTextField {
                                    id: monthDayField
                                    Layout.fillWidth: true
                                    text: "1"
                                    validator: IntValidator { bottom: 1; top: 31 }
                                }
                            }

                            ColumnLayout {
                                visible: calendarTypeCombo.selectedValue === "every"
                                Layout.fillWidth: true
                                spacing: Theme.spacingXSmall

                                Text {
                                    text: qsTr("Every N days")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                QDTextField {
                                    id: everyDaysField
                                    Layout.fillWidth: true
                                    text: "3"
                                    validator: IntValidator { bottom: 1; top: 365 }
                                }
                            }

                            ColumnLayout {
                                Layout.fillWidth: true
                                spacing: Theme.spacingXSmall

                                Text {
                                    text: qsTr("IANA timezone")
                                    font.pixelSize: Theme.fontSizeSmall
                                    color: Theme.textSecondary
                                }

                                QDTextField {
                                    id: timezoneField
                                    Layout.fillWidth: true
                                    placeholderText: "Asia/Shanghai"
                                }
                            }
                        }

                        QDSwitch {
                            id: skipHolidaysSwitch
                            text: qsTr("Skip statutory holidays")
                        }

                        Text {
                            Layout.fillWidth: true
                            text: qsTr("Supported schedules: %1")
                                  .arg(root.manager
                                       ? root.manager.supportedSchedules.join(" · ")
                                       : "")
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.textSecondary
                            wrapMode: Text.WordWrap
                        }
                    }
                }
            }
        }

        footer: [
            QDButton {
                text: qsTr("Cancel")
                buttonType: QDButton.Type.Secondary
                onClicked: createDialog.reject()
            },
            QDButton {
                text: qsTr("Create")
                buttonType: QDButton.Type.Primary
                loading: root.manager ? root.manager.busy : false
                onClicked: root.submitCreate()
            }
        ]
    }

    QDDialog {
        id: deleteDialog
        title: qsTr("Delete scheduled task")
        dialogWidth: 460
        dialogHeight: 220
        closeOnOverlay: false

        Text {
            anchors.fill: parent
            text: qsTr("Delete the %1 task? This cannot be undone.")
                  .arg(root.kindLabel(root.pendingDeleteKind))
            color: Theme.text
            wrapMode: Text.WordWrap
        }

        footer: [
            QDButton {
                text: qsTr("Cancel")
                buttonType: QDButton.Type.Secondary
                onClicked: deleteDialog.reject()
            },
            QDButton {
                text: qsTr("Delete")
                buttonType: QDButton.Type.Danger
                loading: root.manager ? root.manager.busy : false
                onClicked: {
                    if (root.manager && root.pendingDeleteId > 0) {
                        root.manager.deleteTask(root.pendingDeleteId)
                    }
                }
            }
        ]
    }
}
