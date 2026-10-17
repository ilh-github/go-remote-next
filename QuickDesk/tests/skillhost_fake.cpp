// Copyright 2026 QuickDesk Authors

#include <QCoreApplication>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QTextStream>

#include <cstdio>

int main(int argc, char** argv)
{
    QCoreApplication app(argc, argv);
    QTextStream input(stdin);
    QTextStream output(stdout);

    QString line;
    while (input.readLineInto(&line)) {
        const QJsonObject request =
            QJsonDocument::fromJson(line.toUtf8()).object();
        if (request.value(QStringLiteral("type")).toString()
            != QStringLiteral("toolCall")) {
            continue;
        }

        QJsonObject response;
        response[QStringLiteral("id")] =
            request.value(QStringLiteral("id")).toString();
        response[QStringLiteral("type")] =
            QStringLiteral("toolResult");
        if (request.value(QStringLiteral("tool")).toString()
            == QStringLiteral("get_system_info")) {
            response[QStringLiteral("result")] = QJsonObject{
                {QStringLiteral("hostname"), QStringLiteral("fake-host")},
                {QStringLiteral("cpu"),
                 QJsonObject{{QStringLiteral("usage_percent"),
                              QStringLiteral("12.5")}}},
                {QStringLiteral("memory"),
                 QJsonObject{{QStringLiteral("usage_percent"),
                              QStringLiteral("34.5")}}},
                {QStringLiteral("disks"),
                 QJsonArray{
                     QJsonObject{
                         {QStringLiteral("mount"), QStringLiteral("/")},
                         {QStringLiteral("total_gb"), 100.0},
                         {QStringLiteral("available_gb"), 20.0},
                     },
                 }},
            };
        } else {
            response[QStringLiteral("result")] = QJsonObject{
                {QStringLiteral("echo"), request.value(QStringLiteral("tool"))},
            };
        }
        output << QString::fromUtf8(
                      QJsonDocument(response)
                          .toJson(QJsonDocument::Compact))
               << '\n';
        output.flush();
    }
    return 0;
}
