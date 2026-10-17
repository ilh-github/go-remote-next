import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "../component"

Item {
    id: root

    property var mainController
    property var updateManager: mainController ? mainController.updateManager : null

    signal showToast(string message, int toastType)

    function updateStatusText() {
        if (!updateManager) return qsTr("Update service unavailable")
        switch (updateManager.state) {
        case "checking":
            return qsTr("Checking for updates...")
        case "up-to-date":
            return qsTr("QuickDesk is up to date.")
        case "available":
            return qsTr("Version %1 is available.").arg(updateManager.latestVersion)
        case "downloading":
            return qsTr("Downloading %1: %2%")
                .arg(updateManager.latestVersion)
                .arg(Math.round(updateManager.downloadProgress * 100))
        case "ready":
            return qsTr("Version %1 is downloaded and verified.").arg(updateManager.latestVersion)
        case "installing":
            return qsTr("Starting the installer...")
        case "failed":
            return updateManager.errorMessage
        default:
            return qsTr("Check the self-hosted update channel for a newer build.")
        }
    }

    Rectangle {
        anchors.fill: parent
        color: Theme.background

        Flickable {
            anchors.fill: parent
            contentWidth: width
            contentHeight: contentColumn.implicitHeight
            clip: true

            ScrollBar.vertical: QDScrollBar {}

            Column {
                id: contentColumn
                width: parent.width
                spacing: Theme.spacingLarge

                Item { width: 1; height: Theme.spacingMedium }

                Text {
                    x: Theme.spacingXLarge
                    text: qsTr("About")
                    font.pixelSize: Theme.fontSizeHeading
                    font.weight: Font.Bold
                    color: Theme.text
                }

                // App Info Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: appInfoColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: appInfoColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingLarge

                        Row {
                            width: parent.width
                            spacing: Theme.spacingLarge

                            Image {
                                width: 64
                                height: 64
                                source: "qrc:/image/tray/logo.png"
                                fillMode: Image.PreserveAspectFit
                                smooth: true
                            }

                            Column {
                                width: parent.width - 64 - parent.spacing
                                spacing: Theme.spacingXSmall
                                anchors.verticalCenter: parent.verticalCenter

                                Text {
                                    text: "QuickDesk"
                                    font.pixelSize: Theme.fontSizeXLarge
                                    font.weight: Font.Bold
                                    color: Theme.text
                                }

                                Text {
                                    text: qsTr("Version") + " " + APP_VERSION
                                    font.pixelSize: Theme.fontSizeMedium
                                    color: Theme.textSecondary
                                }
                            }
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        Text {
                            width: parent.width
                            text: qsTr("The first AI-native remote desktop. Built-in MCP Server lets any AI agent see and control remote computers. Built on Chromium Remoting — industrial-grade performance, security, and stability refined by Google for over a decade.")
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                            wrapMode: Text.WordWrap
                            lineHeight: 1.4
                        }
                    }
                }

                // Updates Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: updatesColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: updatesColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingMedium

                        Text {
                            text: qsTr("Updates")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        Text {
                            width: parent.width
                            text: root.updateStatusText()
                            font.pixelSize: Theme.fontSizeMedium
                            color: root.updateManager && root.updateManager.state === "failed"
                                   ? Theme.error : Theme.textSecondary
                            wrapMode: Text.WordWrap
                        }

                        Text {
                            visible: root.updateManager &&
                                     root.updateManager.releaseNotes.length > 0 &&
                                     (root.updateManager.state === "available" ||
                                      root.updateManager.state === "ready")
                            width: parent.width
                            text: root.updateManager ? root.updateManager.releaseNotes : ""
                            font.pixelSize: Theme.fontSizeSmall
                            color: Theme.text
                            wrapMode: Text.WordWrap
                            maximumLineCount: 6
                            elide: Text.ElideRight
                        }

                        QDProgressBar {
                            visible: root.updateManager &&
                                     (root.updateManager.state === "downloading" ||
                                      root.updateManager.state === "ready")
                            width: parent.width
                            from: 0
                            to: 1
                            value: root.updateManager ? root.updateManager.downloadProgress : 0
                        }

                        Row {
                            spacing: Theme.spacingSmall

                            QDButton {
                                text: qsTr("Check for Updates")
                                iconText: FluentIconGlyph.syncGlyph
                                enabled: root.updateManager &&
                                         root.updateManager.state !== "checking" &&
                                         root.updateManager.state !== "downloading" &&
                                         root.updateManager.state !== "installing"
                                onClicked: root.updateManager.checkForUpdates(true)
                            }

                            QDButton {
                                visible: root.updateManager &&
                                         (root.updateManager.state === "available" ||
                                          root.updateManager.state === "failed")
                                text: qsTr("Download Update")
                                iconText: FluentIconGlyph.downloadGlyph
                                buttonType: QDButton.Type.Primary
                                onClicked: root.updateManager.downloadUpdate()
                            }

                            QDButton {
                                visible: root.updateManager && root.updateManager.state === "ready"
                                text: qsTr("Install and Restart")
                                iconText: FluentIconGlyph.acceptGlyph
                                buttonType: QDButton.Type.Primary
                                onClicked: root.updateManager.installUpdate()
                            }
                        }
                    }
                }

                // Links Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: linksColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: linksColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingMedium

                        Text {
                            text: qsTr("Links")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        // GitHub link
                        Rectangle {
                            width: parent.width
                            height: 48
                            color: githubMouseArea.containsMouse ? Theme.surfaceHover : "transparent"
                            radius: Theme.radiusSmall

                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingMedium
                                anchors.rightMargin: Theme.spacingMedium
                                spacing: Theme.spacingMedium

                                Text {
                                    text: FluentIconGlyph.globeGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 16
                                    color: Theme.primary
                                }

                                Column {
                                    Layout.fillWidth: true
                                    spacing: 2

                                    Text {
                                        text: qsTr("Source Code")
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }

                                    Text {
                                        text: "https://github.com/barry-ran/QuickDesk"
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                    }
                                }

                                Text {
                                    text: FluentIconGlyph.goGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 12
                                    color: Theme.textSecondary
                                }
                            }

                            MouseArea {
                                id: githubMouseArea
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: Qt.openUrlExternally("https://github.com/barry-ran/QuickDesk")
                            }
                        }

                        // Chromium Remoting link
                        Rectangle {
                            width: parent.width
                            height: 48
                            color: chromiumMouseArea.containsMouse ? Theme.surfaceHover : "transparent"
                            radius: Theme.radiusSmall

                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: Theme.spacingMedium
                                anchors.rightMargin: Theme.spacingMedium
                                spacing: Theme.spacingMedium

                                Text {
                                    text: FluentIconGlyph.linkGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 16
                                    color: Theme.primary
                                }

                                Column {
                                    Layout.fillWidth: true
                                    spacing: 2

                                    Text {
                                        text: qsTr("Based on Chromium Remoting")
                                        font.pixelSize: Theme.fontSizeMedium
                                        color: Theme.text
                                    }

                                    Text {
                                        text: "https://github.com/barry-ran/quickdesk-remoting"
                                        font.pixelSize: Theme.fontSizeSmall
                                        color: Theme.textSecondary
                                        elide: Text.ElideRight
                                        width: parent.width
                                    }
                                }

                                Text {
                                    text: FluentIconGlyph.goGlyph
                                    font.family: "Segoe Fluent Icons"
                                    font.pixelSize: 12
                                    color: Theme.textSecondary
                                }
                            }

                            MouseArea {
                                id: chromiumMouseArea
                                anchors.fill: parent
                                hoverEnabled: true
                                cursorShape: Qt.PointingHandCursor
                                onClicked: Qt.openUrlExternally("https://github.com/barry-ran/quickdesk-remoting")
                            }
                        }
                    }
                }

                // License Card
                QDCard {
                    x: Theme.spacingXLarge
                    width: parent.width - Theme.spacingXLarge * 2
                    height: licenseColumn.implicitHeight + Theme.spacingLarge * 2

                    Column {
                        id: licenseColumn
                        width: parent.width - Theme.spacingLarge * 2
                        anchors.centerIn: parent
                        spacing: Theme.spacingMedium

                        Text {
                            text: qsTr("License")
                            font.pixelSize: Theme.fontSizeLarge
                            font.weight: Font.DemiBold
                            color: Theme.text
                        }

                        Rectangle { width: parent.width; height: 1; color: Theme.border }

                        Text {
                            width: parent.width
                            text: qsTr("QuickDesk is licensed under the MIT License. The bundled quickdesk-remoting component is based on Chromium and licensed under the BSD 3-Clause License.")
                            font.pixelSize: Theme.fontSizeMedium
                            color: Theme.text
                            wrapMode: Text.WordWrap
                            lineHeight: 1.4
                        }
                    }
                }

                Item { width: 1; height: Theme.spacingXLarge }
            }
        }
    }

    Connections {
        target: root.updateManager

        function onCheckFinished(userInitiated, updateAvailable, error) {
            if (!userInitiated) return
            if (error.length > 0) {
                root.showToast(error, QDToast.Type.Error)
            } else if (updateAvailable) {
                root.showToast(qsTr("Update available."), QDToast.Type.Info)
            } else {
                root.showToast(qsTr("QuickDesk is up to date."), QDToast.Type.Success)
            }
        }

        function onDownloadFinished(success, error) {
            if (success) {
                root.showToast(qsTr("Update downloaded and verified."), QDToast.Type.Success)
            } else if (error.length > 0 && error !== qsTr("Download cancelled")) {
                root.showToast(error, QDToast.Type.Error)
            }
        }
    }
}
