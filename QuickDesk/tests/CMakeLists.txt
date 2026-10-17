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
