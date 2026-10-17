// Copyright 2026 QuickDesk Authors

#include "manager/AuthManager.h"

#include <QString>
#include <QUrl>

#include <functional>
#include <utility>

namespace {
QString g_updateTestBaseUrl;
bool g_testLoggedIn = false;
std::function<void(const QString&,
                   const QUrl&,
                   const QString&,
                   quickdesk::AuthManager::ResponseCallback)>
    g_requestHandler;
}

namespace quickdesk {

void SetUpdateTestAuthBaseUrl(const QString& baseUrl) {
    g_updateTestBaseUrl = baseUrl;
}

void SetAuthTestLoggedIn(bool loggedIn) {
    g_testLoggedIn = loggedIn;
}

void SetAuthTestRequestHandler(
    std::function<void(const QString&,
                       const QUrl&,
                       const QString&,
                       AuthManager::ResponseCallback)> handler) {
    g_requestHandler = std::move(handler);
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
    return g_testLoggedIn;
}

QString AuthManager::username() const {
    return {};
}

uint AuthManager::userId() const {
    return 0;
}

QString AuthManager::token() const {
    return QStringLiteral("test-access-token");
}

bool AuthManager::smsEnabled() const {
    return false;
}

void AuthManager::request(const QString& method,
                          const QUrl& url,
                          const QString& body,
                          ResponseCallback callback) {
    if (g_requestHandler) {
        g_requestHandler(method, url, body, std::move(callback));
        return;
    }
    callback(503, "no test request handler", "");
}

void AuthManager::handleServerSessionRevoked() {}

}  // namespace quickdesk
