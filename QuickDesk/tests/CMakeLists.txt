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
