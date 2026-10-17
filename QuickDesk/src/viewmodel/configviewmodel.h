#pragma once

#include <QObject>

class ConfigViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(int darkTheme READ darkTheme WRITE setDarkTheme NOTIFY darkThemeChanged)
    Q_PROPERTY(QString language READ language WRITE setLanguage NOTIFY languageChanged)
    Q_PROPERTY(int accessCodeRefreshInterval READ accessCodeRefreshInterval WRITE setAccessCodeRefreshInterval NOTIFY accessCodeRefreshIntervalChanged)
    Q_PROPERTY(QString preferredVideoCodec READ preferredVideoCodec WRITE setPreferredVideoCodec NOTIFY preferredVideoCodecChanged)
    Q_PROPERTY(bool autoStart READ autoStart WRITE setAutoStart NOTIFY autoStartChanged)
    Q_PROPERTY(bool autoPrivacyScreenOnConnect READ autoPrivacyScreenOnConnect WRITE setAutoPrivacyScreenOnConnect NOTIFY autoPrivacyScreenOnConnectChanged)
    Q_PROPERTY(bool remoteTabBarPinned READ remoteTabBarPinned WRITE setRemoteTabBarPinned NOTIFY remoteTabBarPinnedChanged)
    Q_PROPERTY(bool remoteDesktopMiniMapVisible READ remoteDesktopMiniMapVisible WRITE setRemoteDesktopMiniMapVisible NOTIFY remoteDesktopMiniMapVisibleChanged)
    Q_PROPERTY(QString apiKey READ apiKey WRITE setApiKey NOTIFY apiKeyChanged)
    Q_PROPERTY(QString signalingServerSpki READ signalingServerSpki WRITE setSignalingServerSpki NOTIFY signalingServerSpkiChanged)

public:
    ConfigViewModel(QObject* parent = nullptr);
    virtual ~ConfigViewModel();

    int darkTheme();
    void setDarkTheme(int value);
    
    QString language();
    void setLanguage(const QString& value);
    
    int accessCodeRefreshInterval();
    void setAccessCodeRefreshInterval(int value);
    
    QString preferredVideoCodec();
    void setPreferredVideoCodec(const QString& value);

    bool autoStart();
    void setAutoStart(bool value);

    bool autoPrivacyScreenOnConnect();
    void setAutoPrivacyScreenOnConnect(bool value);

    bool remoteTabBarPinned();
    void setRemoteTabBarPinned(bool value);

    bool remoteDesktopMiniMapVisible();
    void setRemoteDesktopMiniMapVisible(bool value);

    QString apiKey();
    void setApiKey(const QString& value);

    QString signalingServerSpki();
    void setSignalingServerSpki(const QString& value);
    Q_INVOKABLE bool isValidSignalingServerSpki(const QString& value) const;

signals:
    void darkThemeChanged(int value);
    void languageChanged(const QString& value);
    void accessCodeRefreshIntervalChanged(int value);
    void preferredVideoCodecChanged(const QString& value);
    void autoStartChanged(bool value);
    void autoPrivacyScreenOnConnectChanged(bool value);
    void remoteTabBarPinnedChanged(bool value);
    void remoteDesktopMiniMapVisibleChanged(bool value);
    void apiKeyChanged(const QString& value);
    void signalingServerSpkiChanged(const QString& value);
};
