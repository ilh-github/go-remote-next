#include "configviewmodel.h"

#include "core/certificatepin.h"
#include "core/localconfigcenter.h"
#include "infra/autostart/autostartmanager.h"

ConfigViewModel::ConfigViewModel(QObject* parent)
    : QObject(parent)
{
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalDarkThemeChanged, this, &ConfigViewModel::darkThemeChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalLanguageChanged, this, &ConfigViewModel::languageChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalAccessCodeRefreshIntervalChanged, this, &ConfigViewModel::accessCodeRefreshIntervalChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalPreferredVideoCodecChanged, this, &ConfigViewModel::preferredVideoCodecChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalAutoPrivacyScreenOnConnectChanged, this, &ConfigViewModel::autoPrivacyScreenOnConnectChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalRemoteTabBarPinnedChanged, this, &ConfigViewModel::remoteTabBarPinnedChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalRemoteDesktopMiniMapVisibleChanged, this, &ConfigViewModel::remoteDesktopMiniMapVisibleChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalApiKeyChanged, this, &ConfigViewModel::apiKeyChanged);
    connect(&core::LocalConfigCenter::instance(), &core::LocalConfigCenter::signalSignalingServerSpkiChanged, this, &ConfigViewModel::signalingServerSpkiChanged);
}

ConfigViewModel::~ConfigViewModel()
{
}

int ConfigViewModel::darkTheme()
{
    return core::LocalConfigCenter::instance().darkTheme();
}

void ConfigViewModel::setDarkTheme(int value)
{
    core::LocalConfigCenter::instance().setDarkTheme(value);
}

QString ConfigViewModel::language()
{
    return core::LocalConfigCenter::instance().language();
}

void ConfigViewModel::setLanguage(const QString& value)
{
    core::LocalConfigCenter::instance().setLanguage(value);
}

int ConfigViewModel::accessCodeRefreshInterval()
{
    return core::LocalConfigCenter::instance().accessCodeRefreshInterval();
}

void ConfigViewModel::setAccessCodeRefreshInterval(int value)
{
    core::LocalConfigCenter::instance().setAccessCodeRefreshInterval(value);
}

QString ConfigViewModel::preferredVideoCodec()
{
    return core::LocalConfigCenter::instance().preferredVideoCodec();
}

void ConfigViewModel::setPreferredVideoCodec(const QString& value)
{
    core::LocalConfigCenter::instance().setPreferredVideoCodec(value);
}

bool ConfigViewModel::autoStart()
{
    return infra::AutoStartManager::isAutoStartEnabled();
}

void ConfigViewModel::setAutoStart(bool value)
{
    if (autoStart() == value) {
        return;
    }
    infra::AutoStartManager::setAutoStartEnabled(value);
    Q_EMIT autoStartChanged(value);
}

bool ConfigViewModel::autoPrivacyScreenOnConnect()
{
    return core::LocalConfigCenter::instance().autoPrivacyScreenOnConnect();
}

void ConfigViewModel::setAutoPrivacyScreenOnConnect(bool value)
{
    core::LocalConfigCenter::instance().setAutoPrivacyScreenOnConnect(value);
}

bool ConfigViewModel::remoteTabBarPinned()
{
    return core::LocalConfigCenter::instance().remoteTabBarPinned();
}

void ConfigViewModel::setRemoteTabBarPinned(bool value)
{
    core::LocalConfigCenter::instance().setRemoteTabBarPinned(value);
}

bool ConfigViewModel::remoteDesktopMiniMapVisible()
{
    return core::LocalConfigCenter::instance().remoteDesktopMiniMapVisible();
}

void ConfigViewModel::setRemoteDesktopMiniMapVisible(bool value)
{
    core::LocalConfigCenter::instance().setRemoteDesktopMiniMapVisible(value);
}

QString ConfigViewModel::apiKey()
{
    return core::LocalConfigCenter::instance().apiKey();
}

void ConfigViewModel::setApiKey(const QString& value)
{
    core::LocalConfigCenter::instance().setApiKey(value);
}

QString ConfigViewModel::signalingServerSpki()
{
    return core::LocalConfigCenter::instance().signalingServerSpki();
}

void ConfigViewModel::setSignalingServerSpki(const QString& value)
{
    const auto parsed =
        quickdesk::ParseCertificatePinList(value.trimmed().toStdString());
    if (!parsed.valid) {
        return;
    }
    core::LocalConfigCenter::instance().setSignalingServerSpki(
        QString::fromStdString(parsed.normalized));
}

bool ConfigViewModel::isValidSignalingServerSpki(const QString& value) const
{
    return quickdesk::ParseCertificatePinList(value.trimmed().toStdString()).valid;
}
