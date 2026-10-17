add_executable(quickdesk_certificate_pin_test
    certificatepin_test.cpp
    ../src/core/certificatepin.cpp
)

target_include_directories(quickdesk_certificate_pin_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

add_test(
    NAME quickdesk_certificate_pin_test
    COMMAND quickdesk_certificate_pin_test
)

add_executable(quickdesk_update_support_test
    updatesupport_test.cpp
    ../src/core/updatesupport.cpp
)

target_include_directories(quickdesk_update_support_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

add_test(
    NAME quickdesk_update_support_test
    COMMAND quickdesk_update_support_test
)

add_executable(quickdesk_update_manager_test
    updatemanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/core/db/appconfigdatabase.cpp
    ../src/core/localconfigcenter.cpp
    ../src/core/updatesupport.cpp
    ../src/manager/AuthManager.h
    ../src/manager/ServerManager.cpp
    ../src/manager/UpdateManager.cpp
    ../src/manager/UpdateManager.h
)

target_include_directories(quickdesk_update_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_update_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Network
    Qt6::Sql
)

if(APPLE)
    target_link_libraries(quickdesk_update_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_update_manager_test
    COMMAND quickdesk_update_manager_test
)

add_executable(quickdesk_cloud_device_manager_test
    clouddevicemanager_integration_test.cpp
    authmanager_stub.cpp
    hostmanager_stub.cpp
    ../src/core/db/appconfigdatabase.cpp
    ../src/core/localconfigcenter.cpp
    ../src/manager/AuthManager.h
    ../src/manager/CloudDeviceManager.cpp
    ../src/manager/CloudDeviceManager.h
    ../src/manager/ServerManager.cpp
)

target_include_directories(quickdesk_cloud_device_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_cloud_device_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Network
    Qt6::Sql
    Qt6::WebSockets
)

if(APPLE)
    target_link_libraries(quickdesk_cloud_device_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_cloud_device_manager_test
    COMMAND quickdesk_cloud_device_manager_test
)

add_executable(quickdesk_access_approval_manager_test
    accessapprovalmanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/manager/AuthManager.h
    ../src/manager/AccessApprovalManager.cpp
    ../src/manager/AccessApprovalManager.h
)

target_include_directories(quickdesk_access_approval_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_access_approval_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
)

if(APPLE)
    target_link_libraries(quickdesk_access_approval_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_access_approval_manager_test
    COMMAND quickdesk_access_approval_manager_test
)

find_package(Qt6 QUIET COMPONENTS Test)

add_executable(quickdesk_remote_session_manager_test
    remotesessionmanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/manager/AuthManager.h
    ../src/manager/RemoteSessionManager.cpp
    ../src/manager/RemoteSessionManager.h
)

target_include_directories(quickdesk_remote_session_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_remote_session_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_remote_session_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_remote_session_manager_test
    COMMAND quickdesk_remote_session_manager_test
)

add_executable(quickdesk_scheduled_task_manager_test
    scheduledtaskmanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/manager/AuthManager.h
    ../src/manager/ScheduledTaskManager.cpp
    ../src/manager/ScheduledTaskManager.h
)

target_include_directories(quickdesk_scheduled_task_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_scheduled_task_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_scheduled_task_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_scheduled_task_manager_test
    COMMAND quickdesk_scheduled_task_manager_test
)

add_executable(quickdesk_notification_manager_test
    notificationmanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/manager/AuthManager.h
    ../src/manager/NotificationManager.cpp
    ../src/manager/NotificationManager.h
)

target_include_directories(quickdesk_notification_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_notification_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_notification_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_notification_manager_test
    COMMAND quickdesk_notification_manager_test
)

add_executable(quickdesk_recording_manager_test
    recordingmanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/manager/AuthManager.h
    ../src/manager/RecordingManager.cpp
    ../src/manager/RecordingManager.h
)

target_include_directories(quickdesk_recording_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_recording_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_recording_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_recording_manager_test
    COMMAND quickdesk_recording_manager_test
)

find_package(Qt6 QUIET COMPONENTS QuickTest)
if(Qt6QuickTest_FOUND)
    add_executable(quickdesk_qml_test
        qml_test_main.cpp
    )

    target_link_libraries(quickdesk_qml_test
        PRIVATE
        Qt6::QuickTest
        Qt6::Qml
    )

    add_test(
        NAME quickdesk_device_display_qml_test
        COMMAND quickdesk_qml_test
            -input ${CMAKE_CURRENT_SOURCE_DIR}/qml
    )

    set_tests_properties(quickdesk_device_display_qml_test PROPERTIES
        ENVIRONMENT "QT_QPA_PLATFORM=offscreen"
    )
endif()
