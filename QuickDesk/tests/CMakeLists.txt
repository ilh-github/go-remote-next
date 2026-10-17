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
