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

add_executable(quickdesk_host_readiness_test
    hostreadiness_test.cpp
    ../src/core/hostreadiness.cpp
)

target_include_directories(quickdesk_host_readiness_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

add_test(
    NAME quickdesk_host_readiness_test
    COMMAND quickdesk_host_readiness_test
)

add_executable(quickdesk_control_plane_event_test
    controlplaneevent_test.cpp
    ../src/core/controlplaneevent.cpp
)

target_include_directories(quickdesk_control_plane_event_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_control_plane_event_test
    PRIVATE
    Qt6::Core
)

add_test(
    NAME quickdesk_control_plane_event_test
    COMMAND quickdesk_control_plane_event_test
)

add_executable(quickdesk_recording_lifecycle_test
    recordinglifecycle_test.cpp
    ../src/core/recordinglifecycle.cpp
)

target_include_directories(quickdesk_recording_lifecycle_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_recording_lifecycle_test
    PRIVATE
    Qt6::Test
    Qt6::Core
)

add_test(
    NAME quickdesk_recording_lifecycle_test
    COMMAND quickdesk_recording_lifecycle_test
)

add_executable(quickdesk_fake_skill_host
    skillhost_fake.cpp
)

target_link_libraries(quickdesk_fake_skill_host
    PRIVATE
    Qt6::Core
)

add_executable(quickdesk_fake_host
    fake_host.cpp
)

target_link_libraries(quickdesk_fake_host
    PRIVATE
    Qt6::Core
)

# HostManager reaches LocalConfigCenter for the API key, so those sources come
# along - the same set the skill-host test needs, and for the same reason.
add_executable(quickdesk_host_startup_integration_test
    hoststartup_integration_test.cpp
    ../src/core/db/appconfigdatabase.cpp
    ../src/core/localconfigcenter.cpp
    ../src/manager/HostManager.cpp
    ../src/manager/NativeMessaging.cpp
)

# The fake Host is spawned as a separate process, so it has to exist by the time
# this test runs. add_dependencies orders the build; the path is passed on the
# command line below.
add_dependencies(quickdesk_host_startup_integration_test quickdesk_fake_host)

target_include_directories(quickdesk_host_startup_integration_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_host_startup_integration_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_host_startup_integration_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_host_startup_integration_test
    COMMAND quickdesk_host_startup_integration_test
            $<TARGET_FILE:quickdesk_fake_host>
)

add_executable(quickdesk_skill_host_manager_test
    skillhostmanager_integration_test.cpp
    ../src/core/db/appconfigdatabase.cpp
    ../src/core/localconfigcenter.cpp
    ../src/manager/NativeMessaging.cpp
    ../src/manager/HostManager.cpp
    ../src/manager/HostManager.h
    ../src/manager/SkillHostManager.cpp
    ../src/manager/SkillHostManager.h
)

add_dependencies(quickdesk_skill_host_manager_test quickdesk_fake_skill_host)

target_include_directories(quickdesk_skill_host_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_skill_host_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Network
    Qt6::Sql
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_skill_host_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_skill_host_manager_test
    COMMAND quickdesk_skill_host_manager_test
            $<TARGET_FILE:quickdesk_fake_skill_host>
)

add_executable(quickdesk_session_chat_manager_test
    sessionchatmanager_test.cpp
    ../src/core/db/appconfigdatabase.cpp
    ../src/core/localconfigcenter.cpp
    ../src/manager/NativeMessaging.cpp
    ../src/manager/HostManager.cpp
    ../src/manager/HostManager.h
    ../src/manager/SessionChatManager.cpp
    ../src/manager/SessionChatManager.h
    ../src/manager/SkillHostManager.cpp
    ../src/manager/SkillHostManager.h
)

target_include_directories(quickdesk_session_chat_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_session_chat_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Network
    Qt6::Sql
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_session_chat_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_session_chat_manager_test
    COMMAND quickdesk_session_chat_manager_test
)

add_executable(quickdesk_update_manager_test
    updatemanager_integration_test.cpp
    authmanager_stub.cpp
    ../qml.qrc
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

add_executable(quickdesk_device_group_manager_test
    devicegroupmanager_integration_test.cpp
    authmanager_stub.cpp
    ../src/manager/AuthManager.h
    ../src/manager/DeviceGroupManager.cpp
    ../src/manager/DeviceGroupManager.h
)

target_include_directories(quickdesk_device_group_manager_test
    PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/../src
)

target_link_libraries(quickdesk_device_group_manager_test
    PRIVATE
    base
    infra
    Qt6::Core
    Qt6::Test
)

if(APPLE)
    target_link_libraries(quickdesk_device_group_manager_test PRIVATE "-framework Foundation")
endif()

add_test(
    NAME quickdesk_device_group_manager_test
    COMMAND quickdesk_device_group_manager_test
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
