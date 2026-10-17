// Copyright 2026 QuickDesk Authors

#pragma once

#include <QJsonObject>
#include <QString>

namespace quickdesk {

bool isWorkflowTriggerEvent(const QString& type);
QJsonObject workflowTriggerEventData(const QJsonObject& message);

}  // namespace quickdesk
