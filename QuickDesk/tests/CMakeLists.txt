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
