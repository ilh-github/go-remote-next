// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"

#include <QString>

namespace {
QString g_updateTestBaseUrl;
}

namespace quickdesk {

void SetUpdateTestAuthBaseUrl(const QString& baseUrl) {
    g_updateTestBaseUrl = baseUrl;
}

AuthManager::AuthManager(ServerManager*, QObject* parent)
    : QObject(parent)
{
}

QString AuthManager::httpBaseUrl() const {
    return g_updateTestBaseUrl;
}

QList<QPair<QString, QString>> AuthManager::publicHeaders() const {
    return {qMakePair(QStringLiteral("X-API-Key"),
                      QStringLiteral("update-test-key"))};
}

void AuthManager::fetchFeatures() {}

void AuthManager::login(const QString&, const QString&) {}

void AuthManager::loginWithSms(const QString&, const QString&) {}

void AuthManager::sendSmsCode(const QString&, const QString&) {}

void AuthManager::registerUser(const QString&, const QString&,
                               const QString&, const QString&,
                               const QString&) {}

void AuthManager::logout(const QString&) {}

void AuthManager::fetchUserInfo() {}

bool AuthManager::isLoggedIn() const {
    return false;
}

QString AuthManager::username() const {
    return {};
}

uint AuthManager::userId() const {
    return 0;
}

QString AuthManager::token() const {
    return {};
}

bool AuthManager::smsEnabled() const {
    return false;
}

}  // namespace quickdesk
